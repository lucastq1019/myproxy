package service

import (
	"fmt"

	"myproxy.com/p/internal/systemproxy"
	"myproxy.com/p/internal/xray"
)

// ProxyService 系统代理服务层，提供系统代理相关的业务逻辑。
type ProxyService struct {
	systemProxy  *systemproxy.SystemProxy
	xrayInstance *xray.XrayInstance
}

// NewProxyService 创建新的代理服务实例。
// 参数：
//   - xrayInstance: Xray 实例，用于获取代理端口
//
// 返回：初始化后的 ProxyService 实例
func NewProxyService(xrayInstance *xray.XrayInstance) *ProxyService {
	ps := &ProxyService{
		xrayInstance: xrayInstance,
	}
	ps.updateSystemProxyPort()
	return ps
}

// updateSystemProxyPort 更新系统代理管理器的端口。
func (ps *ProxyService) updateSystemProxyPort() {
	proxyPort := 10808
	if ps.xrayInstance != nil && ps.xrayInstance.IsRunning() {
		if port := ps.xrayInstance.GetPort(); port > 0 {
			proxyPort = port
		}
	}
	ps.systemProxy = systemproxy.NewSystemProxy("127.0.0.1", proxyPort)
}

// UpdateXrayInstance 更新 Xray 实例引用（当 Xray 实例变化时调用）。
// 参数：
//   - xrayInstance: Xray 实例
func (ps *ProxyService) UpdateXrayInstance(xrayInstance *xray.XrayInstance) {
	ps.xrayInstance = xrayInstance
	ps.updateSystemProxyPort()
}

// ApplySystemProxyModeResult 系统代理操作结果。
type ApplySystemProxyModeResult struct {
	LogMessage string // 日志消息
	Error      error  // 错误（如果有）
}

// ApplySystemProxyMode 应用系统代理模式。
// 参数：
//   - mode: 系统代理模式（clear, auto, terminal）
//
// 返回：操作结果（包含日志消息和错误）
func (ps *ProxyService) ApplySystemProxyMode(mode string) *ApplySystemProxyModeResult {
	ps.updateSystemProxyPort()

	var err error
	var logMessage string

	switch mode {
	case "clear":
		err = ps.systemProxy.ClearSystemProxy()
		terminalErr := ps.systemProxy.ClearTerminalProxy()
		if err == nil && terminalErr == nil {
			logMessage = "已清除系统代理设置和环境变量代理"
		} else if err != nil && terminalErr != nil {
			logMessage = fmt.Sprintf("清除系统代理失败: %v; 清除环境变量代理失败: %v", err, terminalErr)
			err = fmt.Errorf("代理服务: 清除失败: %v; %v", err, terminalErr)
		} else if err != nil {
			logMessage = fmt.Sprintf("清除系统代理失败: %v; 已清除环境变量代理", err)
		} else {
			logMessage = fmt.Sprintf("已清除系统代理设置; 清除环境变量代理失败: %v", terminalErr)
			err = terminalErr
		}

	case "auto":
		_ = ps.systemProxy.ClearSystemProxy()
		_ = ps.systemProxy.ClearTerminalProxy()
		err = ps.systemProxy.SetSystemProxy()
		if err == nil {
			proxyPort := 10808
			if ps.xrayInstance != nil && ps.xrayInstance.IsRunning() {
				if port := ps.xrayInstance.GetPort(); port > 0 {
					proxyPort = port
				}
			}
			logMessage = fmt.Sprintf("已自动配置系统代理: 127.0.0.1:%d", proxyPort)
		} else {
			logMessage = fmt.Sprintf("自动配置系统代理失败: %v", err)
		}

	case "terminal":
		_ = ps.systemProxy.ClearSystemProxy()
		_ = ps.systemProxy.ClearTerminalProxy()
		err = ps.systemProxy.SetTerminalProxy()
		if err == nil {
			proxyPort := 10808
			if ps.xrayInstance != nil && ps.xrayInstance.IsRunning() {
				if port := ps.xrayInstance.GetPort(); port > 0 {
					proxyPort = port
				}
			}
			logMessage = fmt.Sprintf("已设置环境变量代理: socks5://127.0.0.1:%d (已写入shell配置文件)", proxyPort)
		} else {
			logMessage = fmt.Sprintf("设置环境变量代理失败: %v", err)
		}

	default:
		return &ApplySystemProxyModeResult{
			LogMessage: fmt.Sprintf("未知的系统代理模式: %s", mode),
			Error:      fmt.Errorf("代理服务: 未知的系统代理模式: %s", mode),
		}
	}

	return &ApplySystemProxyModeResult{
		LogMessage: logMessage,
		Error:      err,
	}
}
