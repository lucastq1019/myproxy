package service

import (
	"myproxy.com/p/internal/systemproxy"
)

// SystemProxyService 系统代理服务，提供系统代理相关的业务逻辑。
type SystemProxyService struct {
	proxy *systemproxy.SystemProxy
}

// NewSystemProxyService 创建新的系统代理服务实例。
// 参数：
//   - proxyHost: 代理主机地址
//   - proxyPort: 代理端口
//
// 返回：初始化后的 SystemProxyService 实例
func NewSystemProxyService(proxyHost string, proxyPort int) *SystemProxyService {
	return &SystemProxyService{
		proxy: systemproxy.NewSystemProxy(proxyHost, proxyPort),
	}
}

// ClearSystemProxy 清除系统代理设置。
// 返回：错误（如果有）
func (sps *SystemProxyService) ClearSystemProxy() error {
	if sps.proxy == nil {
		return nil
	}
	return sps.proxy.ClearSystemProxy()
}

// SetSystemProxy 自动配置系统代理。
// 返回：错误（如果有）
func (sps *SystemProxyService) SetSystemProxy() error {
	if sps.proxy == nil {
		return nil
	}
	return sps.proxy.SetSystemProxy()
}

// SetTerminalProxy 设置终端代理（环境变量代理）。
// 参数：
//   - proxyType: 代理类型（https 或 socks5）
//
// 返回：错误（如果有）
func (sps *SystemProxyService) SetTerminalProxy(proxyType string) error {
	if sps.proxy == nil {
		return nil
	}
	return sps.proxy.SetTerminalProxy(proxyType)
}

// ClearTerminalProxy 清除终端代理设置。
// 返回：错误（如果有）
func (sps *SystemProxyService) ClearTerminalProxy() error {
	if sps.proxy == nil {
		return nil
	}
	return sps.proxy.ClearTerminalProxy()
}

// GetCurrentProxyMode 获取当前代理模式。
// 返回：当前代理模式（none/auto/terminal）
func (sps *SystemProxyService) GetCurrentProxyMode() systemproxy.ProxyMode {
	if sps.proxy == nil {
		return systemproxy.ProxyModeNone
	}
	return sps.proxy.GetCurrentProxyMode()
}

// UpdateProxy 更新代理地址和端口（用于动态更新）。
// 参数：
//   - host: 新的代理主机地址
//   - port: 新的代理端口
func (sps *SystemProxyService) UpdateProxy(host string, port int) {
	if sps.proxy == nil {
		sps.proxy = systemproxy.NewSystemProxy(host, port)
	} else {
		sps.proxy.UpdateProxy(host, port)
	}
}

// ApplyProxyMode 应用指定的代理模式。
// 参数：
//   - mode: 代理模式字符串（clear/auto/terminal）
//   - proxyType: 代理类型（https 或 socks5），仅用于terminal模式
//
// 返回：错误（如果有）
func (sps *SystemProxyService) ApplyProxyMode(mode string, proxyType string) error {
	if sps.proxy == nil {
		return nil
	}

	switch mode {
	case "clear", "none":
		return sps.ClearSystemProxy()
	case "auto":
		return sps.SetSystemProxy()
	case "terminal":
		return sps.SetTerminalProxy(proxyType)
	default:
		return sps.ClearSystemProxy() // 默认清除代理
	}
}
