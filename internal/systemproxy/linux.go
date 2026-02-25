package systemproxy

import (
	"fmt"
	"os"
)

// LinuxProxy Linux 平台的代理实现
type LinuxProxy struct {
	proxyHost string
	proxyPort int
}

func newLinuxProxy(host string, port int) *LinuxProxy {
	return &LinuxProxy{
		proxyHost: host,
		proxyPort: port,
	}
}

func (p *LinuxProxy) ClearSystemProxy() error {
	// TODO: 实现 Linux 系统代理清除
	// 可以通过 gsettings (GNOME) 或其他方式
	return fmt.Errorf("linux 系统代理清除功能暂未实现")
}

func (p *LinuxProxy) SetSystemProxy(host string, port int) error {
	// TODO: 实现 Linux 系统代理设置
	return fmt.Errorf("linux 系统代理设置功能暂未实现")
}

func (p *LinuxProxy) SetTerminalProxy(host string, port int, proxyType string) error {
	if proxyType == "" {
		proxyType = "socks5"
	}
	proxyURL := fmt.Sprintf("%s://%s:%d", proxyType, host, port)

	// 设置当前进程环境变量
	os.Setenv("HTTP_PROXY", proxyURL)
	os.Setenv("HTTPS_PROXY", proxyURL)
	os.Setenv("http_proxy", proxyURL)
	os.Setenv("https_proxy", proxyURL)
	os.Setenv("ALL_PROXY", proxyURL)
	os.Setenv("all_proxy", proxyURL)

	// Linux 也可以使用外部shell文件方案（类似 macOS）
	// TODO: 实现 Linux 的外部shell文件方案
	return nil
}

func (p *LinuxProxy) ClearTerminalProxy() error {
	os.Unsetenv("HTTP_PROXY")
	os.Unsetenv("HTTPS_PROXY")
	os.Unsetenv("http_proxy")
	os.Unsetenv("https_proxy")
	os.Unsetenv("ALL_PROXY")
	os.Unsetenv("all_proxy")
	return nil
}

func (p *LinuxProxy) GetCurrentProxyMode() ProxyMode {
	if os.Getenv("HTTP_PROXY") != "" || os.Getenv("http_proxy") != "" {
		return ProxyModeTerminal
	}
	return ProxyModeNone
}
