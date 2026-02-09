package ui

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/fsnotify/fsnotify"
)

// LogEntry 表示一条日志条目
type LogEntry struct {
	Timestamp time.Time
	Level     string
	Type      string
	Message   string
	Line      string // 完整的日志行
}

// LogsPanel 管理应用日志和代理日志的显示。
// 它支持按日志级别和类型过滤，并提供追加日志功能。
type LogsPanel struct {
	appState       *AppState
	logContent     *widget.RichText // 使用 RichText 以支持自定义文本颜色
	levelSel       *widget.Select
	typeSel        *widget.Select
	logBuffer      []LogEntry         // 日志缓冲区
	bufferMutex    sync.Mutex         // 保护日志缓冲区的互斥锁
	maxBufferSize  int                // 最大缓冲区大小
	fileWatcher    *fsnotify.Watcher  // 文件监控器
	ctx            context.Context    // 上下文，用于控制监控 goroutine
	cancel         context.CancelFunc // 取消函数
	lastReadPos    int64              // 最后读取的位置
	isCollapsed    bool               // 是否折叠
	collapseBtn    *widget.Button     // 折叠/展开按钮
	logScroll      *container.Scroll  // 日志滚动容器
	panelContainer fyne.CanvasObject  // 面板容器
}

// NewLogsPanel 创建并初始化日志显示面板。
// 该方法会创建日志内容区域、过滤控件，并加载初始日志。
// 参数：
//   - appState: 应用状态实例
//
// 返回：初始化后的日志面板实例
func NewLogsPanel(appState *AppState) *LogsPanel {
	lp := &LogsPanel{
		appState:      appState,
		logBuffer:     make([]LogEntry, 0),
		maxBufferSize: 1000, // 最多保存1000条日志
		isCollapsed:   true, // 默认折叠，符合“默认隐藏，需要时深入”的设计
	}

	// 从 ConfigService 加载折叠状态（优先用户之前的选择）
	if appState != nil && appState.ConfigService != nil {
		lp.isCollapsed = appState.ConfigService.GetLogsCollapsed()
	}

	// 日志内容 - 使用 RichText 以支持自定义文本颜色
	lp.logContent = widget.NewRichText()
	lp.logContent.Wrapping = fyne.TextWrapWord // 启用自动换行，适配窄屏显示
	// 设置等宽字体样式和初始段落样式（优化行高）
	lp.logContent.Segments = []widget.RichTextSegment{}

	// 日志级别选择器：仅用于过滤显示，不影响日志输出级别
	lp.levelSel = widget.NewSelect(
		[]string{"全部", "DEBUG", "INFO", "WARN", "ERROR", "FATAL"},
		func(value string) {
			if lp.typeSel != nil { // 确保 typeSel 已初始化
				lp.refreshDisplay() // 仅刷新显示，不修改日志级别配置
			}
		},
	)

	// 日志类型选择器：仅用于过滤显示
	lp.typeSel = widget.NewSelect(
		[]string{"全部", "app", "proxy", "xray"},
		func(value string) {
			if lp.levelSel != nil { // 确保 levelSel 已初始化
				lp.refreshDisplay() // 仅刷新显示
			}
		},
	)

	// 等所有组件创建完成后再设置默认值和刷新
	lp.levelSel.SetSelected("全部")
	lp.typeSel.SetSelected("全部")

	// 创建上下文用于控制监控 goroutine
	lp.ctx, lp.cancel = context.WithCancel(context.Background())

	// 初始加载历史日志
	lp.loadInitialLogs()

	// 注意：文件监控将在Logger初始化后启动（在cmd/gui/main.go中）

	return lp
}

// Build 构建并返回日志显示面板的 UI 组件。
// 返回：包含过滤控件和日志内容的容器组件
func (lp *LogsPanel) Build() fyne.CanvasObject {
	// 级别标签（使用副标题样式）
	levelLabel := widget.NewLabel("级别")

	// 类型标签（使用副标题样式）
	typeLabel := widget.NewLabel("类型")

	// 顶部控制栏：仅保留级别/类型筛选（无刷新按钮），在窄宽度下自然纵向排列。
	levelRow := container.NewHBox(
		levelLabel,
		layout.NewSpacer(),
		container.NewGridWrap(fyne.NewSize(100, 40), lp.levelSel),
		layout.NewSpacer(),
	)
	typeRow := container.NewHBox(
		typeLabel,
		layout.NewSpacer(),
		container.NewGridWrap(fyne.NewSize(100, 40), lp.typeSel),
		layout.NewSpacer(),
	)
	topBar := container.NewPadded(container.NewVBox(levelRow, typeRow))

	// 日志内容区域
	lp.logScroll = container.NewScroll(lp.logContent)

	// 创建面板容器
	lp.panelContainer = container.NewBorder(
		container.NewVBox(
			topBar,
			NewSeparator(),
		),
		nil,
		nil,
		nil,
		lp.logScroll,
	)

	return lp.panelContainer
}

// toggleCollapse 切换折叠/展开状态
func (lp *LogsPanel) toggleCollapse() {
	lp.isCollapsed = !lp.isCollapsed
	lp.updateCollapseState()
	lp.updateCollapseButtonText()

	// 保存状态到数据库（通过 ConfigService）
	if lp.appState != nil && lp.appState.ConfigService != nil {
		if err := lp.appState.ConfigService.SetLogsCollapsed(lp.isCollapsed); err != nil {
			if lp.appState.Logger != nil {
				lp.appState.Logger.Error("保存日志折叠状态失败: %v", err)
			}
		}
	}
}

// updateCollapseState 更新折叠状态显示
func (lp *LogsPanel) updateCollapseState() {
	if lp.logScroll == nil {
		return
	}

	if lp.isCollapsed {
		// 折叠：隐藏日志内容，只显示控制栏
		lp.logScroll.Hide()
	} else {
		// 展开：显示日志内容
		lp.logScroll.Show()
	}

	// 刷新容器
	if lp.panelContainer != nil {
		lp.panelContainer.Refresh()
	}
}

// updateCollapseButtonText 更新折叠按钮图标
func (lp *LogsPanel) updateCollapseButtonText() {
	if lp.collapseBtn == nil {
		return
	}

	var icon fyne.Resource
	if lp.isCollapsed {
		icon = theme.MenuExpandIcon()
	} else {
		icon = theme.MenuDropDownIcon()
	}
	lp.collapseBtn.SetIcon(icon)
}

// IsCollapsed 返回当前是否折叠
func (lp *LogsPanel) IsCollapsed() bool {
	return lp.isCollapsed
}

// AppendLog 追加一条日志到日志面板（线程安全）
// 注意：此方法主要用于兼容性，建议使用 AppendLogLine 确保格式一致
// 该方法可以从任何地方调用，会自动追加到日志缓冲区并更新显示
func (lp *LogsPanel) AppendLog(level, logType, message string) {
	if lp == nil {
		return
	}

	// 规范化：级别一律大写，类型仅允许 app/xray
	level = strings.ToUpper(level)
	switch strings.ToLower(logType) {
	case "xray":
		logType = "xray"
	default:
		logType = "app"
	}

	// 构建完整的日志行（与Logger.log()中的格式保持一致）
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logLine := fmt.Sprintf("%s [%s] [%s] %s", timestamp, level, logType, message)

	// 使用统一的AppendLogLine方法
	lp.AppendLogLine(logLine)
}

// AppendLogLine 追加一条完整格式的日志行到日志面板（线程安全）
// 此方法用于确保日志格式与文件中的格式一致
// 参数：
//   - logLine: 完整的日志行，格式为 "timestamp [LEVEL] [type] message"
func (lp *LogsPanel) AppendLogLine(logLine string) {
	if lp == nil {
		return
	}

	// 解析日志行
	entry := lp.parseLogLine(logLine)
	if entry == nil {
		return
	}

	// 线程安全地追加到缓冲区
	lp.bufferMutex.Lock()
	lp.logBuffer = append(lp.logBuffer, *entry)

	// 如果超过最大大小，删除最旧的日志
	if len(lp.logBuffer) > lp.maxBufferSize {
		lp.logBuffer = lp.logBuffer[len(lp.logBuffer)-lp.maxBufferSize:]
	}
	lp.bufferMutex.Unlock()

	// 更新显示
	lp.refreshDisplay()
}

// loadInitialLogs 加载初始日志（从文件加载历史日志）
func (lp *LogsPanel) loadInitialLogs() {
	if lp.appState == nil || lp.appState.Logger == nil {
		return
	}

	logFilePath := lp.appState.Logger.GetLogFilePath()
	if logFilePath == "" {
		return
	}

	// 尝试直接读取日志文件（如果存在）
	file, err := os.Open(logFilePath)
	if err != nil {
		// 文件不存在，跳过
		return
	}
	defer file.Close()

	// 读取所有内容
	scanner := bufio.NewScanner(file)
	logLines := make([]string, 0)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			logLines = append(logLines, line)
		}
	}

	// 解析日志行并添加到缓冲区
	lp.bufferMutex.Lock()
	lp.logBuffer = make([]LogEntry, 0, len(logLines))
	for _, line := range logLines {
		entry := lp.parseLogLine(line)
		if entry != nil {
			lp.logBuffer = append(lp.logBuffer, *entry)
		}
	}

	// 更新 lastReadPos 为文件末尾（避免重复读取）
	if fileInfo, err := os.Stat(logFilePath); err == nil {
		lp.lastReadPos = fileInfo.Size()
	}

	lp.bufferMutex.Unlock()

	// 刷新显示
	lp.refreshDisplay()
}

// parseLogLine 解析日志行，提取级别、类型和消息
// 支持两种格式：
// 1. 应用日志格式: timestamp [LEVEL] [type] message
// 2. xray 日志格式: timestamp [Level] tag: message 或 timestamp [Level] tag/subtag: message
func (lp *LogsPanel) parseLogLine(line string) *LogEntry {
	// 尝试解析应用日志格式: timestamp [LEVEL] [type] message
	levelStart := strings.Index(line, "[")
	if levelStart == -1 {
		return nil
	}
	levelEnd := strings.Index(line[levelStart:], "]")
	if levelEnd == -1 {
		return nil
	}
	levelEnd += levelStart

	// 提取时间戳和级别
	timestampStr := strings.TrimSpace(line[:levelStart])
	level := line[levelStart+1 : levelEnd]

	// 查找第二个 [ 和 ]（应用日志格式）
	typeStart := strings.Index(line[levelEnd+1:], "[")
	var logType string
	var message string
	var timestamp time.Time

	if typeStart != -1 && typeStart < 50 { // 第二个 [ 应该很快出现（应用日志格式）
		typeStart += levelEnd + 1
		typeEnd := strings.Index(line[typeStart:], "]")
		if typeEnd != -1 {
			typeEnd += typeStart
			// 应用日志格式
			logType = strings.ToLower(line[typeStart+1 : typeEnd])
			message = strings.TrimSpace(line[typeEnd+1:])
			var err error
			timestamp, err = time.Parse("2006-01-02 15:04:05", timestampStr)
			if err != nil {
				timestamp = time.Now()
			}
		} else {
			return nil
		}
	} else {
		// xray 日志格式: timestamp [Level] tag/subtag: message
		// 例如: 2025/12/15 16:53:13.879127 [Debug] app/log: Logger started
		rest := strings.TrimSpace(line[levelEnd+1:])
		colonIndex := strings.Index(rest, ":")
		if colonIndex > 0 {
			_ = strings.TrimSpace(rest[:colonIndex])
			message = strings.TrimSpace(rest[colonIndex+1:])

			// 从 tag 中提取类型（例如 "app/log" -> 归并为 xray）
			logType = "xray"

			// 解析 xray 时间戳格式: 2025/12/15 16:53:13.879127
			var err error
			timestamp, err = time.Parse("2006/01/02 15:04:05.000000", timestampStr)
			if err != nil {
				// 尝试不带微秒的格式
				timestamp, err = time.Parse("2006/01/02 15:04:05", timestampStr)
				if err != nil {
					timestamp = time.Now()
				}
			}
		} else {
			// 没有冒号，整个 rest 作为消息
			message = rest
			logType = "xray"
			timestamp = time.Now()
		}
	}

	// 标准化级别名称（全大写），来源小写
	level = strings.ToUpper(level)
	switch level {
	case "DEBUG", "INFO", "WARN", "ERROR", "FATAL":
	default:
		level = "INFO"
	}
	logType = strings.ToLower(logType)

	return &LogEntry{
		Timestamp: timestamp,
		Level:     level,
		Type:      logType,
		Message:   message,
		Line:      line,
	}
}

// refreshDisplay 根据当前过滤条件刷新显示（仅过滤显示，不影响日志输出级别）
func (lp *LogsPanel) refreshDisplay() {
	if lp.logContent == nil || lp.levelSel == nil || lp.typeSel == nil {
		return
	}

	lp.bufferMutex.Lock()
	defer lp.bufferMutex.Unlock()

	levelFilter := lp.levelSel.Selected
	typeFilter := lp.typeSel.Selected

	// 过滤日志
	var filteredEntries []LogEntry
	for _, entry := range lp.logBuffer {
		// 按级别过滤
		if levelFilter != "全部" && entry.Level != levelFilter {
			continue
		}

		// 按类型过滤
		if typeFilter != "全部" && entry.Type != typeFilter {
			continue
		}

		filteredEntries = append(filteredEntries, entry)
	}

	// 构建显示文本 - 优化字体和行高
	var segments []widget.RichTextSegment
	for _, entry := range filteredEntries {
		// 根据日志级别设置不同的颜色（在黑白主题中，使用不同的灰度）
		colorName := theme.ColorNameForeground
		// 可以根据级别调整样式，但保持黑白主题
		segments = append(segments, &widget.TextSegment{
			Text: entry.Line + "\n",
			Style: widget.RichTextStyle{
				ColorName: colorName,
				TextStyle: fyne.TextStyle{Monospace: true}, // 等宽字体
			},
		})
	}

	// 更新日志内容
	fyne.Do(func() {
		lp.logContent.Segments = segments
		lp.logContent.Refresh()
	})
}

// Refresh 刷新日志显示，重新应用当前过滤条件。
func (lp *LogsPanel) Refresh() {
	lp.refreshDisplay()
}

// StartLogFileWatcher 启动日志文件监控（公开方法，可在Logger初始化后调用）
func (lp *LogsPanel) StartLogFileWatcher() {
	if lp.appState == nil || lp.appState.Logger == nil {
		return
	}

	logFilePath := lp.appState.Logger.GetLogFilePath()
	if logFilePath == "" {
		return
	}

	// 如果监控器已存在，先关闭
	if lp.fileWatcher != nil {
		lp.fileWatcher.Close()
	}

	// 创建文件监控器
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return
	}
	lp.fileWatcher = watcher

	// 监控日志文件所在目录
	logDir := logFilePath
	if lastSlash := strings.LastIndex(logDir, "/"); lastSlash >= 0 {
		logDir = logDir[:lastSlash]
	}
	if logDir == "" {
		logDir = "."
	}

	// 添加目录到监控
	if err := watcher.Add(logDir); err != nil {
		watcher.Close()
		lp.fileWatcher = nil
		return
	}

	// 初始化 lastReadPos 为当前文件大小（避免重复读取已有内容）
	if fileInfo, err := os.Stat(logFilePath); err == nil {
		lp.lastReadPos = fileInfo.Size()
	}

	// 启动监控 goroutine
	go lp.watchLogFile()
}

// watchLogFile 监控日志文件变化
func (lp *LogsPanel) watchLogFile() {
	if lp.fileWatcher == nil {
		return
	}
	defer lp.fileWatcher.Close()

	logFilePath := lp.appState.Logger.GetLogFilePath()
	ticker := time.NewTicker(500 * time.Millisecond) // 每 500ms 检查一次文件变化
	defer ticker.Stop()

	for {
		select {
		case <-lp.ctx.Done():
			return
		case <-ticker.C:
			// 读取文件的新内容
			lp.readNewLogLines(logFilePath)
		case event, ok := <-lp.fileWatcher.Events:
			if !ok {
				return
			}
			// 检查是否是目标日志文件的变化
			if event.Op&fsnotify.Write == fsnotify.Write && event.Name == logFilePath {
				lp.readNewLogLines(logFilePath)
			}
		case err, ok := <-lp.fileWatcher.Errors:
			if !ok {
				return
			}
			// 忽略监控错误，继续运行
			_ = err
		}
	}
}

// readNewLogLines 读取日志文件的新行
// 注意：此方法主要用于读取直接从文件写入的日志（如xray日志）
// 通过Logger写入的日志会通过回调直接更新UI，避免重复处理
func (lp *LogsPanel) readNewLogLines(logFilePath string) {
	file, err := os.Open(logFilePath)
	if err != nil {
		return
	}
	defer file.Close()

	// 获取文件信息
	fileInfo, err := file.Stat()
	if err != nil {
		return
	}

	// 如果文件大小小于等于上次读取的位置，说明没有新内容
	if fileInfo.Size() <= lp.lastReadPos {
		// 如果文件被截断（比如归档），重置位置
		if fileInfo.Size() < lp.lastReadPos {
			lp.lastReadPos = 0
		}
		return
	}

	// 移动到上次读取的位置
	if _, err := file.Seek(lp.lastReadPos, 0); err != nil {
		return
	}

	// 读取新内容
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			// 使用统一的AppendLogLine方法，确保格式一致性
			lp.AppendLogLine(line)
		}
	}

	// 更新最后读取的位置
	lp.lastReadPos, _ = file.Seek(0, 1)
}

// Stop 停止日志面板的监控
func (lp *LogsPanel) Stop() {
	if lp.cancel != nil {
		lp.cancel()
	}
	if lp.fileWatcher != nil {
		lp.fileWatcher.Close()
	}
}
