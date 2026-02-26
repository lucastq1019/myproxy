package systemproxy

import (
	"fmt"
	"runtime"
)

// PlatformProxy 平台特定的代理操作接口
type PlatformProxy interface {
	// ClearSystemProxy 清除系统代理设置
	ClearSystemProxy() error
	// SetSystemProxy 设置系统代理
	SetSystemProxy(host string, port int) error
	// SetTerminalProxy 设置终端代理（环境变量）
	SetTerminalProxy(host string, port int, proxyType string) error
	// ClearTerminalProxy 清除终端代理
	ClearTerminalProxy() error
	// GetCurrentProxyMode 获取当前代理模式
	GetCurrentProxyMode() ProxyMode
}

// NewPlatformProxy 根据当前平台创建对应的代理管理器
func NewPlatformProxy(host string, port int) PlatformProxy {
	switch runtime.GOOS {
	case "darwin":
		return newDarwinProxy(host, port)
	case "linux":
		return newLinuxProxy(host, port)
	case "windows":
		return newWindowsProxy(host, port)
	default:
		return newUnsupportedProxy(runtime.GOOS)
	}
}

// UnsupportedProxy 不支持的操作系统实现
type UnsupportedProxy struct {
	os string
}

func newUnsupportedProxy(os string) *UnsupportedProxy {
	return &UnsupportedProxy{os: os}
}

func (p *UnsupportedProxy) ClearSystemProxy() error {
	return fmt.Errorf("不支持的操作系统: %s", p.os)
}

func (p *UnsupportedProxy) SetSystemProxy(host string, port int) error {
	return fmt.Errorf("不支持的操作系统: %s", p.os)
}

func (p *UnsupportedProxy) SetTerminalProxy(host string, port int, proxyType string) error {
	return fmt.Errorf("不支持的操作系统: %s", p.os)
}

func (p *UnsupportedProxy) ClearTerminalProxy() error {
	return fmt.Errorf("不支持的操作系统: %s", p.os)
}

func (p *UnsupportedProxy) GetCurrentProxyMode() ProxyMode {
	return ProxyModeNone
}
