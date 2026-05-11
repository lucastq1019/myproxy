package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	rpprof "runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"time"

	"myproxy.com/p/internal/model"
	"myproxy.com/p/internal/store"
)

const (
	defaultDiagnosticsDirName    = "diagnostics"
	defaultDiagnosticsSampleSecs = 5
	diagnosticsHistoryLimit      = 180
)

// DiagnosticsService 提供运行时采样、摘要导出和 pprof 管理能力。
type DiagnosticsService struct {
	config *ConfigService
	store  *store.Store

	mu        sync.RWMutex
	history   []model.DiagnosticSnapshot
	current   model.DiagnosticSnapshot
	stopCh    chan struct{}
	stoppedCh chan struct{}
	running   bool

	pprofMu     sync.Mutex
	pprofServer *http.Server
	pprofAddr   string
}

// NewDiagnosticsService 创建诊断服务。
func NewDiagnosticsService(config *ConfigService, dataStore *store.Store) *DiagnosticsService {
	return &DiagnosticsService{
		config: config,
		store:  dataStore,
	}
}

// Start 启动运行时采样和 pprof 配置应用。
func (ds *DiagnosticsService) Start() error {
	ds.mu.Lock()
	if ds.running {
		ds.mu.Unlock()
		return ds.applyPprofConfig()
	}

	ds.stopCh = make(chan struct{})
	ds.stoppedCh = make(chan struct{})
	ds.running = true
	ds.history = make([]model.DiagnosticSnapshot, 0, diagnosticsHistoryLimit)
	ds.sampleOnceLocked()
	stopCh := ds.stopCh
	stoppedCh := ds.stoppedCh
	ds.mu.Unlock()

	go ds.sampleLoop(stopCh, stoppedCh)

	return ds.applyPprofConfig()
}

// Stop 停止采样和 pprof 服务。
func (ds *DiagnosticsService) Stop() {
	ds.mu.Lock()
	if ds.running {
		close(ds.stopCh)
		stoppedCh := ds.stoppedCh
		ds.running = false
		ds.mu.Unlock()
		<-stoppedCh
	} else {
		ds.mu.Unlock()
	}
	ds.stopPprofServer()
}

func (ds *DiagnosticsService) sampleLoop(stopCh <-chan struct{}, stoppedCh chan<- struct{}) {
	defer close(stoppedCh)

	for {
		timer := time.NewTimer(time.Duration(ds.getSampleIntervalSeconds()) * time.Second)
		select {
		case <-timer.C:
			ds.mu.Lock()
			ds.sampleOnceLocked()
			ds.mu.Unlock()
		case <-stopCh:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return
		}
	}
}

func (ds *DiagnosticsService) sampleOnceLocked() {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	snapshot := model.DiagnosticSnapshot{
		Timestamp:  time.Now(),
		Alloc:      stats.Alloc,
		HeapAlloc:  stats.HeapAlloc,
		HeapInuse:  stats.HeapInuse,
		Sys:        stats.Sys,
		NumGC:      stats.NumGC,
		Goroutines: runtime.NumGoroutine(),
	}

	ds.current = snapshot
	ds.history = append(ds.history, snapshot)
	if len(ds.history) > diagnosticsHistoryLimit {
		ds.history = ds.history[len(ds.history)-diagnosticsHistoryLimit:]
	}
}

// CurrentSnapshot 返回当前采样值。
func (ds *DiagnosticsService) CurrentSnapshot() model.DiagnosticSnapshot {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.current
}

// History 返回采样历史副本。
func (ds *DiagnosticsService) History() []model.DiagnosticSnapshot {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	out := make([]model.DiagnosticSnapshot, len(ds.history))
	copy(out, ds.history)
	return out
}

// GetSummary 返回诊断摘要。
func (ds *DiagnosticsService) GetSummary(proxyRunning bool, proxyPort int, serverName string) model.DiagnosticSummary {
	current := ds.CurrentSnapshot()
	executablePath, _ := os.Executable()

	return model.DiagnosticSummary{
		Timestamp:                time.Now(),
		GoVersion:                runtime.Version(),
		ExecutablePath:           executablePath,
		DiagnosticsDir:           ds.getDiagnosticsDir(),
		PprofEnabled:             ds.IsPprofEnabled(),
		PprofAddr:                ds.GetPprofAddr(),
		ProxyRunning:             proxyRunning,
		ProxyPort:                proxyPort,
		CurrentServerName:        serverName,
		LastNodeSwitchAt:         ds.getConfigTime("lastNodeSwitchAt"),
		LastSubscriptionUpdateAt: ds.getConfigTime("lastSubscriptionUpdateAt"),
		LastDiagnosticExport:     ds.getConfigValue("lastDiagnosticExport"),
		Current:                  current,
	}
}

// ExportHeapProfile 导出堆快照。
func (ds *DiagnosticsService) ExportHeapProfile() (string, error) {
	if err := os.MkdirAll(ds.getDiagnosticsDir(), 0755); err != nil {
		return "", fmt.Errorf("创建诊断目录失败: %w", err)
	}

	filePath := filepath.Join(ds.getDiagnosticsDir(), "heap_"+time.Now().Format("20060102_150405")+".pprof")
	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("创建堆快照文件失败: %w", err)
	}
	defer file.Close()

	runtime.GC()
	if err := rpprof.WriteHeapProfile(file); err != nil {
		return "", fmt.Errorf("写入堆快照失败: %w", err)
	}

	ds.recordLastExport(filePath)
	return filePath, nil
}

// ExportGoroutineProfile 导出 goroutine 快照。
func (ds *DiagnosticsService) ExportGoroutineProfile() (string, error) {
	if err := os.MkdirAll(ds.getDiagnosticsDir(), 0755); err != nil {
		return "", fmt.Errorf("创建诊断目录失败: %w", err)
	}

	filePath := filepath.Join(ds.getDiagnosticsDir(), "goroutine_"+time.Now().Format("20060102_150405")+".pprof")
	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("创建 goroutine 快照文件失败: %w", err)
	}
	defer file.Close()

	profile := rpprof.Lookup("goroutine")
	if profile == nil {
		return "", fmt.Errorf("goroutine profile 不可用")
	}
	if err := profile.WriteTo(file, 2); err != nil {
		return "", fmt.Errorf("写入 goroutine 快照失败: %w", err)
	}

	ds.recordLastExport(filePath)
	return filePath, nil
}

// ExportSummaryJSON 导出诊断摘要 JSON。
func (ds *DiagnosticsService) ExportSummaryJSON(summary model.DiagnosticSummary) (string, error) {
	if err := os.MkdirAll(ds.getDiagnosticsDir(), 0755); err != nil {
		return "", fmt.Errorf("创建诊断目录失败: %w", err)
	}

	filePath := filepath.Join(ds.getDiagnosticsDir(), "summary_"+time.Now().Format("20060102_150405")+".json")
	payload, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return "", fmt.Errorf("序列化诊断摘要失败: %w", err)
	}
	if err := os.WriteFile(filePath, payload, 0644); err != nil {
		return "", fmt.Errorf("写入诊断摘要失败: %w", err)
	}

	ds.recordLastExport(filePath)
	return filePath, nil
}

// GenerateHeapFlameGraph 导出 heap profile 并尝试生成 svg 火焰图。
func (ds *DiagnosticsService) GenerateHeapFlameGraph() (string, string, error) {
	profilePath, err := ds.ExportHeapProfile()
	if err != nil {
		return "", "", err
	}

	executablePath, err := os.Executable()
	if err != nil {
		return profilePath, "", fmt.Errorf("获取可执行文件路径失败: %w", err)
	}
	if _, err := os.Stat(executablePath); err != nil {
		return profilePath, "", fmt.Errorf("可执行文件不可访问，无法生成火焰图: %w", err)
	}

	svgPath := strings.TrimSuffix(profilePath, ".pprof") + ".svg"
	cmd := exec.Command("go", "tool", "pprof", "-svg", "-output", svgPath, executablePath, profilePath)
	if output, err := cmd.CombinedOutput(); err != nil {
		trimmed := strings.TrimSpace(string(output))
		if strings.Contains(trimmed, `no such tool "pprof"`) {
			return profilePath, "", fmt.Errorf("当前 Go 环境未提供 pprof 工具，无法生成火焰图")
		}
		return profilePath, "", fmt.Errorf("生成火焰图失败: %v: %s", err, trimmed)
	}

	ds.recordLastExport(svgPath)
	return profilePath, svgPath, nil
}

// OpenDiagnosticsDirectory 尝试打开诊断目录。
func (ds *DiagnosticsService) OpenDiagnosticsDirectory() error {
	dir := ds.getDiagnosticsDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建诊断目录失败: %w", err)
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", dir)
	case "windows":
		cmd = exec.Command("explorer", dir)
	default:
		cmd = exec.Command("xdg-open", dir)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("打开诊断目录失败: %w", err)
	}
	return nil
}

// ApplyPprofConfig 根据当前配置启停 pprof 服务。
func (ds *DiagnosticsService) ApplyPprofConfig() error {
	return ds.applyPprofConfig()
}

func (ds *DiagnosticsService) applyPprofConfig() error {
	if !ds.IsPprofEnabled() {
		ds.stopPprofServer()
		return nil
	}

	addr := ds.GetPprofAddr()
	if addr == "" {
		addr = "127.0.0.1:6060"
	}
	if !isLocalPprofAddr(addr) {
		return fmt.Errorf("pprof 地址仅允许监听 localhost 或 127.0.0.1")
	}

	ds.pprofMu.Lock()
	if ds.pprofServer != nil && ds.pprofAddr == addr {
		ds.pprofMu.Unlock()
		return nil
	}
	ds.pprofMu.Unlock()

	ds.stopPprofServer()

	server := &http.Server{
		Addr:              addr,
		Handler:           http.DefaultServeMux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ds.pprofMu.Lock()
	ds.pprofServer = server
	ds.pprofAddr = addr
	ds.pprofMu.Unlock()

	go func() {
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			ds.pprofMu.Lock()
			if ds.pprofServer == server {
				ds.pprofServer = nil
			}
			ds.pprofMu.Unlock()
		}
	}()

	return nil
}

func (ds *DiagnosticsService) stopPprofServer() {
	ds.pprofMu.Lock()
	server := ds.pprofServer
	ds.pprofServer = nil
	ds.pprofAddr = ""
	ds.pprofMu.Unlock()

	if server == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}

// IsPprofEnabled 返回配置中的 pprof 开关。
func (ds *DiagnosticsService) IsPprofEnabled() bool {
	if ds.config == nil {
		return false
	}
	return ds.config.GetDebugPprofEnabled()
}

// GetPprofAddr 返回配置中的 pprof 地址。
func (ds *DiagnosticsService) GetPprofAddr() string {
	if ds.config == nil {
		return "127.0.0.1:6060"
	}
	return ds.config.GetDebugPprofAddr()
}

func (ds *DiagnosticsService) getSampleIntervalSeconds() int {
	if ds.config == nil {
		return defaultDiagnosticsSampleSecs
	}
	secs := ds.config.GetDiagnosticsSamplingSeconds()
	if secs <= 0 {
		return defaultDiagnosticsSampleSecs
	}
	return secs
}

func (ds *DiagnosticsService) getDiagnosticsDir() string {
	if ds.config != nil {
		if dir := strings.TrimSpace(ds.config.GetDiagnosticsDir()); dir != "" {
			return dir
		}
	}

	workDir, err := os.Getwd()
	if err != nil {
		return filepath.Join("data", defaultDiagnosticsDirName)
	}
	return filepath.Join(workDir, "data", defaultDiagnosticsDirName)
}

func (ds *DiagnosticsService) recordLastExport(path string) {
	if ds.config != nil {
		_ = ds.config.Set("lastDiagnosticExport", path)
	}
}

func (ds *DiagnosticsService) getConfigValue(key string) string {
	if ds.config == nil {
		return ""
	}
	value, err := ds.config.Get(key)
	if err != nil {
		return ""
	}
	return value
}

func (ds *DiagnosticsService) getConfigTime(key string) time.Time {
	raw := ds.getConfigValue(key)
	if raw == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}
	}
	return t
}

func isLocalPprofAddr(addr string) bool {
	addr = strings.ToLower(strings.TrimSpace(addr))
	return strings.HasPrefix(addr, "127.0.0.1:") || strings.HasPrefix(addr, "localhost:")
}

func findFreeLocalPort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	_ = l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// validatePprofHTTPAccess 校验 pprof 已启用且地址为本机，返回 host:port（无 scheme）。
func (ds *DiagnosticsService) validatePprofHTTPAccess() (string, error) {
	if !ds.IsPprofEnabled() {
		return "", fmt.Errorf("请先启用本地 pprof")
	}
	addr := strings.TrimSpace(ds.GetPprofAddr())
	if addr == "" {
		addr = "127.0.0.1:6060"
	}
	if !isLocalPprofAddr(addr) {
		return "", fmt.Errorf("pprof 仅允许本机地址，当前为 %s", addr)
	}
	return addr, nil
}

// PprofIndexPageURL 返回标准库 pprof 索引页 URL（浏览器内可点进各 profile）。
func (ds *DiagnosticsService) PprofIndexPageURL() (string, error) {
	host, err := ds.validatePprofHTTPAccess()
	if err != nil {
		return "", err
	}
	return "http://" + host + "/debug/pprof/", nil
}

// PprofHeapTextURL 返回堆内存文本视图（debug=1），便于在浏览器中直接阅读分配情况。
func (ds *DiagnosticsService) PprofHeapTextURL() (string, error) {
	host, err := ds.validatePprofHTTPAccess()
	if err != nil {
		return "", err
	}
	return "http://" + host + "/debug/pprof/heap?debug=1", nil
}

// StartGoToolPprofHeapWebViewer 启动 `go tool pprof` 自带的 Web UI（含火焰图、Top、Source 等）。
// 依赖本机已安装 Go 且 go 在 PATH 中；返回应在浏览器中打开的地址（根路径）。
func (ds *DiagnosticsService) StartGoToolPprofHeapWebViewer() (string, error) {
	host, err := ds.validatePprofHTTPAccess()
	if err != nil {
		return "", err
	}
	port, err := findFreeLocalPort()
	if err != nil {
		return "", fmt.Errorf("分配本地端口失败: %w", err)
	}
	heapSource := "http://" + host + "/debug/pprof/heap"
	httpArg := "127.0.0.1:" + strconv.Itoa(port)
	goExe, err := exec.LookPath("go")
	if err != nil {
		return "", fmt.Errorf("未找到 go 可执行文件，无法启动交互式 pprof（需安装 Go 并配置 PATH）: %w", err)
	}
	cmd := exec.Command(goExe, "tool", "pprof", "-http="+httpArg, heapSource)
	cmd.Env = os.Environ()
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("启动 go tool pprof 失败: %w", err)
	}
	go func() { _ = cmd.Wait() }()
	// 等待 pprof Web 监听就绪后再让浏览器打开
	time.Sleep(500 * time.Millisecond)
	return "http://" + httpArg + "/", nil
}
