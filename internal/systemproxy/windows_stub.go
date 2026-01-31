//go:build !windows
// +build !windows

package systemproxy

import "fmt"

// WindowsProxy Windows 平台的代理实现（非 Windows 平台 stub）
type WindowsProxy struct {
	proxyHost string
	proxyPort int
}

func newWindowsProxy(host string, port int) *WindowsProxy {
	return &WindowsProxy{
		proxyHost: host,
		proxyPort: port,
	}
}

func (p *WindowsProxy) ClearSystemProxy() error {
	return fmt.Errorf("windows 系统代理功能仅在 Windows 平台可用")
}

func (p *WindowsProxy) SetSystemProxy(host string, port int) error {
	return fmt.Errorf("windows 系统代理功能仅在 Windows 平台可用")
}

func (p *WindowsProxy) SetTerminalProxy(host string, port int) error {
	return fmt.Errorf("windows 终端代理功能仅在 Windows 平台可用")
}

func (p *WindowsProxy) ClearTerminalProxy() error {
	return fmt.Errorf("windows 终端代理功能仅在 Windows 平台可用")
}

func (p *WindowsProxy) GetCurrentProxyMode() ProxyMode {
	return ProxyModeNone
}
