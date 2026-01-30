package service

import (
	"fmt"

	"myproxy.com/p/internal/store"
	"myproxy.com/p/internal/xray"
)

// XrayControlService 代理控制服务层，提供 xray 代理启动和停止的业务逻辑。
type XrayControlService struct {
	store        *store.Store
	config       *ConfigService
	logCallback  func(level, message string)
}

// NewXrayControlService 创建新的代理控制服务实例。
// 参数：
//   - store: Store 实例，用于数据访问
//   - config: ConfigService，用于读取直连路由等配置
//   - logCallback: 日志回调函数，用于记录日志
// 返回：初始化后的 XrayControlService 实例
func NewXrayControlService(store *store.Store, config *ConfigService, logCallback func(level, message string)) *XrayControlService {
	return &XrayControlService{
		store:       store,
		config:      config,
		logCallback: logCallback,
	}
}

// StartProxyResult 启动代理操作结果。
type StartProxyResult struct {
	XrayInstance *xray.XrayInstance // Xray 实例
	LogMessage   string             // 日志消息
	Error        error              // 错误（如果有）
}

// StartProxy 启动代理（使用当前选中的节点）。
// 根据架构规范，xray 是工具层，实例生命周期 = 代理运行生命周期。
// 启动代理时创建实例，切换节点时销毁旧实例并创建新实例，停止代理时销毁实例。
// 参数：
//   - oldInstance: 旧的 Xray 实例（如果存在，会先停止）
//   - logFilePath: 日志文件路径
// 返回：操作结果（包含 Xray 实例、日志消息和错误）
func (xcs *XrayControlService) StartProxy(oldInstance *xray.XrayInstance, logFilePath string) *StartProxyResult {
	if xcs.store == nil || xcs.store.Nodes == nil {
		return &StartProxyResult{
			LogMessage: "启动代理失败: Store 未初始化",
			Error:      fmt.Errorf("Xray控制服务: Store 未初始化"),
		}
	}

	// 从 Store 获取当前选中的节点
	selectedNode := xcs.store.Nodes.GetSelected()
	if selectedNode == nil {
		return &StartProxyResult{
			LogMessage: "启动代理失败: 未选中服务器",
			Error:      fmt.Errorf("Xray控制服务: 未选中服务器"),
		}
	}

	// 如果已有代理在运行，先停止并销毁实例
	if oldInstance != nil {
		if oldInstance.IsRunning() {
			_ = oldInstance.Stop()
		}
		// 注意：这里不销毁 oldInstance，由调用者负责
	}

	// 使用固定的10808端口监听本地SOCKS5
	proxyPort := 10808

	// 记录开始启动日志
	if xcs.logCallback != nil {
		xcs.logCallback("INFO", fmt.Sprintf("开始启动xray-core代理: %s", selectedNode.Name))
	}

	// 读取直连路由配置
	var routing *xray.RoutingOptions
	if xcs.config != nil {
		routes := xcs.config.GetDirectRoutes()
		useProxy := xcs.config.GetDirectRoutesUseProxy()
		if len(routes) > 0 {
			routing = &xray.RoutingOptions{
				DirectRoutes:         routes,
				DirectRoutesUseProxy: useProxy,
			}
		}
	}

	// 创建 xray 配置（含日志路径与路由选项）
	xrayConfigJSON, err := xray.CreateXrayConfig(proxyPort, selectedNode, logFilePath, routing)
	if err != nil {
		logMsg := fmt.Sprintf("创建xray配置失败: %v", err)
		if xcs.logCallback != nil {
			xcs.logCallback("ERROR", logMsg)
		}
		return &StartProxyResult{
			LogMessage: logMsg,
			Error:      fmt.Errorf("Xray控制服务: 创建xray配置失败: %w", err),
		}
	}

	// 记录配置创建成功日志
	if xcs.logCallback != nil {
		xcs.logCallback("DEBUG", fmt.Sprintf("xray配置已创建: %s", selectedNode.Name))
	}

	// 创建日志回调函数，将 xray 日志转发到应用日志系统
	logCallback := func(level, message string) {
		if xcs.logCallback != nil {
			xcs.logCallback(level, message)
		}
	}

	// 创建xray实例，并设置日志回调（每次配置变化都需要重新创建实例）
	xrayInstance, err := xray.NewXrayInstanceFromJSONWithCallback(xrayConfigJSON, logCallback)
	if err != nil {
		logMsg := fmt.Sprintf("创建xray实例失败: %v", err)
		if xcs.logCallback != nil {
			xcs.logCallback("ERROR", logMsg)
		}
		return &StartProxyResult{
			LogMessage: logMsg,
			Error:      fmt.Errorf("Xray控制服务: 创建xray实例失败: %w", err),
		}
	}

	// 启动xray实例
	err = xrayInstance.Start()
	if err != nil {
		logMsg := fmt.Sprintf("启动xray实例失败: %v", err)
		if xcs.logCallback != nil {
			xcs.logCallback("ERROR", logMsg)
		}
		return &StartProxyResult{
			XrayInstance: xrayInstance, // 即使启动失败，也返回实例（可能需要清理）
			LogMessage:   logMsg,
			Error:        fmt.Errorf("Xray控制服务: 启动xray实例失败: %w", err),
		}
	}

	// 启动成功，设置端口信息
	xrayInstance.SetPort(proxyPort)

	// 记录日志（统一日志记录）
	logMsg := fmt.Sprintf("xray-core代理已启动: %s (端口: %d)", selectedNode.Name, proxyPort)
	if xcs.logCallback != nil {
		xcs.logCallback("INFO", logMsg)
		xcs.logCallback("INFO", fmt.Sprintf("服务器信息: %s:%d, 协议: %s", selectedNode.Addr, selectedNode.Port, selectedNode.ProtocolType))
	}

	return &StartProxyResult{
		XrayInstance: xrayInstance,
		LogMessage:   logMsg,
		Error:        nil,
	}
}

// StopProxyResult 停止代理操作结果。
type StopProxyResult struct {
	LogMessage string // 日志消息
	Error      error  // 错误（如果有）
}

// StopProxy 停止代理。
// 根据架构规范，xray 实例生命周期 = 代理运行生命周期，停止代理时销毁实例。
// 参数：
//   - instance: Xray 实例
// 返回：操作结果（包含日志消息和错误）
func (xcs *XrayControlService) StopProxy(instance *xray.XrayInstance) *StopProxyResult {
	if instance == nil {
		return &StopProxyResult{
			LogMessage: "代理未运行",
			Error:      nil,
		}
	}

	if !instance.IsRunning() {
		return &StopProxyResult{
			LogMessage: "代理未运行",
			Error:      nil,
		}
	}

	// 记录停止日志
	if xcs.logCallback != nil {
		xcs.logCallback("INFO", "正在停止xray-core代理...")
	}

	err := instance.Stop()
	if err != nil {
		logMsg := fmt.Sprintf("停止xray代理失败: %v", err)
		if xcs.logCallback != nil {
			xcs.logCallback("ERROR", logMsg)
		}
		return &StopProxyResult{
			LogMessage: logMsg,
			Error:      fmt.Errorf("Xray控制服务: 停止xray代理失败: %w", err),
		}
	}

	// 记录成功日志
	logMsg := "xray-core代理已停止"
	if xcs.logCallback != nil {
		xcs.logCallback("INFO", logMsg)
	}

	return &StopProxyResult{
		LogMessage: logMsg,
		Error:      nil,
	}
}

// IsRunning 检查代理是否正在运行。
// 参数：
//   - instance: Xray 实例
// 返回：是否正在运行
func (xcs *XrayControlService) IsRunning(instance *xray.XrayInstance) bool {
	if instance == nil {
		return false
	}
	return instance.IsRunning()
}

