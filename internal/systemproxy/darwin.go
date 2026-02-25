package systemproxy

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// DarwinProxy macOS 平台的代理实现
type DarwinProxy struct {
	proxyHost string
	proxyPort int
}

func newDarwinProxy(host string, port int) *DarwinProxy {
	return &DarwinProxy{
		proxyHost: host,
		proxyPort: port,
	}
}

// ClearSystemProxy 清除 macOS 系统代理设置
func (p *DarwinProxy) ClearSystemProxy() error {
	services, err := p.getNetworkServices()
	if err != nil {
		return fmt.Errorf("获取网络服务失败: %v", err)
	}

	for _, service := range services {
		// 清除 HTTP 代理
		cmd := exec.Command("networksetup", "-setwebproxystate", service, "off")
		_ = cmd.Run()

		// 清除 HTTPS 代理
		cmd = exec.Command("networksetup", "-setsecurewebproxystate", service, "off")
		_ = cmd.Run()

		// 清除 SOCKS 代理
		cmd = exec.Command("networksetup", "-setsocksfirewallproxystate", service, "off")
		_ = cmd.Run()
	}
	return nil
}

// SetSystemProxy 设置 macOS 系统代理
func (p *DarwinProxy) SetSystemProxy(host string, port int) error {
	services, err := p.getNetworkServices()
	if err != nil {
		return fmt.Errorf("获取网络服务失败: %v", err)
	}

	portStr := fmt.Sprintf("%d", port)
	for _, service := range services {
		// 设置 HTTP 代理
		cmd := exec.Command("networksetup", "-setwebproxy", service, host, portStr)
		if err := cmd.Run(); err != nil {
			continue
		}

		// 设置 HTTPS 代理
		cmd = exec.Command("networksetup", "-setsecurewebproxy", service, host, portStr)
		_ = cmd.Run()

		// 设置 SOCKS 代理
		cmd = exec.Command("networksetup", "-setsocksfirewallproxy", service, host, portStr)
		_ = cmd.Run()
	}
	return nil
}

// SetTerminalProxy 设置终端代理（使用外部shell文件方案）
func (p *DarwinProxy) SetTerminalProxy(host string, port int, proxyType string) error {
	if proxyType == "" {
		proxyType = "socks5"
	}
	proxyURL := fmt.Sprintf("%s://%s:%d", proxyType, host, port)

	// 1. 设置当前进程环境变量（立即生效）
	os.Setenv("HTTP_PROXY", proxyURL)
	os.Setenv("HTTPS_PROXY", proxyURL)
	os.Setenv("http_proxy", proxyURL)
	os.Setenv("https_proxy", proxyURL)
	os.Setenv("ALL_PROXY", proxyURL)
	os.Setenv("all_proxy", proxyURL)

	// 2. 使用外部shell文件方案（推荐）
	return p.setupExternalShellFile(proxyURL)
}

// ClearTerminalProxy 清除终端代理
func (p *DarwinProxy) ClearTerminalProxy() error {
	// 清除当前进程环境变量
	os.Unsetenv("HTTP_PROXY")
	os.Unsetenv("HTTPS_PROXY")
	os.Unsetenv("http_proxy")
	os.Unsetenv("https_proxy")
	os.Unsetenv("ALL_PROXY")
	os.Unsetenv("all_proxy")

	// 清除外部shell文件
	return p.removeExternalShellFile()
}

// GetCurrentProxyMode 获取当前代理模式
func (p *DarwinProxy) GetCurrentProxyMode() ProxyMode {
	if os.Getenv("HTTP_PROXY") != "" || os.Getenv("http_proxy") != "" {
		return ProxyModeTerminal
	}
	return ProxyModeNone
}

// getNetworkServices 获取 macOS 网络服务列表
func (p *DarwinProxy) getNetworkServices() ([]string, error) {
	cmd := exec.Command("networksetup", "-listallnetworkservices")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	var services []string
	for i, line := range lines {
		if i == 0 {
			continue // 跳过第一行标题
		}
		line = strings.TrimSpace(line)
		if line != "" {
			services = append(services, line)
		}
	}

	if len(services) == 0 {
		return []string{"Wi-Fi", "Ethernet"}, nil // 默认服务
	}

	return services, nil
}

// setupExternalShellFile 使用外部shell文件方案设置代理
// 方案：在 ~/.myproxy_proxy.sh 中定义代理环境变量，然后在 shell 配置文件中 source 它
func (p *DarwinProxy) setupExternalShellFile(proxyURL string) error {
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		return fmt.Errorf("无法获取用户主目录")
	}

	// 1. 创建外部代理配置文件
	proxyFile := fmt.Sprintf("%s/.myproxy_proxy.sh", homeDir)
	configContent := fmt.Sprintf(`# Proxy settings (set by myproxy)
# This file is managed by myproxy. Do not edit manually.

export HTTP_PROXY=%s
export HTTPS_PROXY=%s
export http_proxy=%s
export https_proxy=%s
export ALL_PROXY=%s
export all_proxy=%s
`, proxyURL, proxyURL, proxyURL, proxyURL, proxyURL, proxyURL)

	if err := os.WriteFile(proxyFile, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("写入代理配置文件失败: %v", err)
	}

	// 2. 在 shell 配置文件中添加 source 语句（如果不存在）
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/zsh"
	}

	var configFile string
	if strings.Contains(shell, "zsh") {
		configFile = fmt.Sprintf("%s/.zshrc", homeDir)
	} else if strings.Contains(shell, "bash") {
		configFile = fmt.Sprintf("%s/.bashrc", homeDir)
	} else {
		return fmt.Errorf("不支持的 shell: %s", shell)
	}

	// 读取现有配置
	content, err := os.ReadFile(configFile)
	if err != nil {
		content = []byte{}
	}

	contentStr := string(content)
	sourceLine := fmt.Sprintf("source %s", proxyFile)

	// 检查是否已经存在 source 语句
	if strings.Contains(contentStr, sourceLine) {
		return nil // 已经配置过了
	}

	// 检查是否已经存在 myproxy 相关的 source（可能路径不同）
	if strings.Contains(contentStr, ".myproxy_proxy.sh") {
		// 已经存在，但可能路径不同，先移除旧的
		contentStr = p.removeOldSourceLine(contentStr)
	}

	// 追加 source 语句
	newContent := contentStr
	if len(newContent) > 0 && !strings.HasSuffix(newContent, "\n") {
		newContent += "\n"
	}
	newContent += fmt.Sprintf("# Source myproxy proxy settings\n%s\n", sourceLine)

	return os.WriteFile(configFile, []byte(newContent), 0644)
}

// removeExternalShellFile 移除外部shell文件配置
func (p *DarwinProxy) removeExternalShellFile() error {
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		return nil
	}

	// 1. 删除外部代理配置文件
	proxyFile := fmt.Sprintf("%s/.myproxy_proxy.sh", homeDir)
	_ = os.Remove(proxyFile) // 忽略错误，文件可能不存在

	// 2. 从 shell 配置文件中移除 source 语句
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/zsh"
	}

	var configFile string
	if strings.Contains(shell, "zsh") {
		configFile = fmt.Sprintf("%s/.zshrc", homeDir)
	} else if strings.Contains(shell, "bash") {
		configFile = fmt.Sprintf("%s/.bashrc", homeDir)
	} else {
		return nil
	}

	content, err := os.ReadFile(configFile)
	if err != nil {
		return nil // 文件不存在，无需清除
	}

	contentStr := string(content)
	newContent := p.removeOldSourceLine(contentStr)

	// 如果内容有变化，写回文件
	if newContent != contentStr {
		return os.WriteFile(configFile, []byte(newContent), 0644)
	}

	return nil
}

// removeOldSourceLine 从配置文件中移除旧的 source 语句
func (p *DarwinProxy) removeOldSourceLine(content string) string {
	lines := strings.Split(content, "\n")
	var newLines []string
	skipNext := false

	for i, line := range lines {
		// 跳过包含 .myproxy_proxy.sh 的 source 行
		if strings.Contains(line, ".myproxy_proxy.sh") {
			// 检查是否是注释行
			if strings.HasPrefix(strings.TrimSpace(line), "#") {
				// 如果是注释，检查下一行是否是 source
				if i+1 < len(lines) && strings.Contains(lines[i+1], "source") && strings.Contains(lines[i+1], ".myproxy_proxy.sh") {
					skipNext = true
					continue
				}
			} else if strings.Contains(line, "source") {
				// 直接是 source 行，跳过
				continue
			}
		}

		// 如果上一行是注释且这一行是 source，跳过
		if skipNext && strings.Contains(line, "source") {
			skipNext = false
			continue
		}
		skipNext = false

		newLines = append(newLines, line)
	}

	return strings.Join(newLines, "\n")
}
