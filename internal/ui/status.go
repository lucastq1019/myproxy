package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"myproxy.com/p/internal/database"
	"myproxy.com/p/internal/logging"
	"myproxy.com/p/internal/systemproxy"
)

// 系统代理模式常量定义
const (
	// 完整模式名称
	SystemProxyModeClear      = "清除系统代理"
	SystemProxyModeAuto       = "自动配置系统代理"
	SystemProxyModeTerminal   = "环境变量代理"

	// 简短模式名称（用于UI显示）
	SystemProxyModeShortClear    = "清除"
	SystemProxyModeShortAuto     = "系统"
	SystemProxyModeShortTerminal = "终端"
)

// StatusPanel 显示代理状态、端口和当前服务器信息。
// 它使用 Fyne 的双向数据绑定机制，当应用状态更新时自动刷新显示。
type StatusPanel struct {
	appState         *AppState
	proxyStatusLabel *widget.Label
	portLabel        *widget.Label
	serverNameLabel  *widget.Label
	delayLabel       *widget.Label
	proxyModeSelect  *widget.Select
	systemProxy      *systemproxy.SystemProxy
	statusIcon       *widget.Icon // 状态图标
	portIcon         *widget.Icon // 端口图标
	serverIcon       *widget.Icon // 服务器图标
	proxyIcon        *widget.Icon // 代理图标

	// 主界面一键操作大按钮相关
	mainToggleButton *widget.Button      // 主开关按钮（连接/断开）
	onToggleProxy    func()              // 由外部注入的代理开关回调
}

// NewStatusPanel 创建并初始化状态信息面板。
// 该方法会创建绑定到应用状态的标签组件，实现自动更新。
// 参数：
//   - appState: 应用状态实例
//
// 返回：初始化后的状态面板实例
func NewStatusPanel(appState *AppState) *StatusPanel {
	sp := &StatusPanel{
		appState: appState,
	}

	// 检查绑定数据是否已初始化
	if appState == nil {
		// 如果 appState 为 nil，创建默认标签（不应该发生，但作为安全措施）
		sp.proxyStatusLabel = widget.NewLabel("代理状态: 未知")
		sp.portLabel = widget.NewLabel("动态端口: -")
		sp.serverNameLabel = widget.NewLabel("当前服务器: 无")
		sp.delayLabel = widget.NewLabel("延迟: -")
		return sp
	}

	// 使用绑定数据创建标签，实现自动更新
	// 代理状态标签 - 绑定到 ProxyStatusBinding
	if appState.ProxyStatusBinding != nil {
		sp.proxyStatusLabel = widget.NewLabelWithData(appState.ProxyStatusBinding)
	} else {
		sp.proxyStatusLabel = widget.NewLabel("代理状态: 未知")
	}
	sp.proxyStatusLabel.Wrapping = fyne.TextWrapOff

	// 端口标签 - 绑定到 PortBinding
	if appState.PortBinding != nil {
		sp.portLabel = widget.NewLabelWithData(appState.PortBinding)
	} else {
		sp.portLabel = widget.NewLabel("动态端口: -")
	}
	sp.portLabel.Wrapping = fyne.TextWrapOff

	// 服务器名称标签 - 绑定到 ServerNameBinding
	if appState.ServerNameBinding != nil {
		sp.serverNameLabel = widget.NewLabelWithData(appState.ServerNameBinding)
	} else {
		sp.serverNameLabel = widget.NewLabel("当前服务器: 无")
	}
	sp.serverNameLabel.Wrapping = fyne.TextWrapOff

	// 当前延迟标签（非绑定，使用 Refresh 时从 ServerManager 读取）
	sp.delayLabel = widget.NewLabel("延迟: -")
	sp.delayLabel.Wrapping = fyne.TextWrapOff

	// 创建系统代理管理器（默认使用 localhost:10080）
	sp.systemProxy = systemproxy.NewSystemProxy("127.0.0.1", 10080)

	// 创建图标
	sp.statusIcon = widget.NewIcon(theme.CancelIcon())
	sp.portIcon = widget.NewIcon(theme.InfoIcon())
	sp.serverIcon = widget.NewIcon(theme.ComputerIcon())
	sp.proxyIcon = widget.NewIcon(theme.SettingsIcon())

	// 创建主开关按钮（大按钮），具体文本在 Build/Refresh 中根据状态更新
	sp.mainToggleButton = widget.NewButton("主开关 (未连接)", func() {
		// 交由外部注入的回调处理实际的启动/停止逻辑
		if sp.onToggleProxy != nil {
			sp.onToggleProxy()
		}
	})
	// 使用较高的重要性，让按钮在主题下更突出
	sp.mainToggleButton.Importance = widget.HighImportance

	// 创建系统代理设置下拉框（只读，用于显示当前状态，不绑定 change 事件）
	// 选项使用简短文本显示，但在内部映射到完整功能
	sp.proxyModeSelect = widget.NewSelect(
		[]string{
			SystemProxyModeShortClear,
			SystemProxyModeShortAuto,
			SystemProxyModeShortTerminal,
		},
		nil, // 不绑定 change 事件，只在启动时恢复状态
	)
	sp.proxyModeSelect.PlaceHolder = "系统代理设置"

	// 恢复系统代理状态（在应用启动时）
	sp.restoreSystemProxyState()

	return sp
}

// Build 构建并返回状态信息面板的 UI 组件。
// 返回：包含代理状态、端口和服务器名称的水平布局容器
func (sp *StatusPanel) Build() fyne.CanvasObject {
	// 更新状态图标
	sp.updateStatusIcon()
	// 更新主按钮和延迟标签内容
	sp.updateMainToggleButton()
	sp.updateDelayLabel()

	// 顶部：当前连接状态（简洁文案）
	statusHeader := container.NewCenter(container.NewHBox(
		sp.statusIcon,
		NewSpacer(SpacingSmall),
		sp.proxyStatusLabel,
	))

	// 中部：巨大的主开关按钮（居中）
	mainControlArea := container.NewCenter(sp.mainToggleButton)

	// 下方：当前节点 + 模式 + 延迟（单行简洁信息）
	nodeAndMode := container.NewHBox(
		sp.serverIcon,
		NewSpacer(SpacingSmall),
		sp.serverNameLabel,
		NewSpacer(SpacingMedium),
		widget.NewLabel("模式:"),
		NewSpacer(SpacingSmall),
		sp.proxyModeSelect,
		NewSpacer(SpacingMedium),
		sp.delayLabel,
		layout.NewSpacer(),
	)
	nodeAndMode = container.NewPadded(nodeAndMode)

	// 底部：实时流量占位（未来可替换为小曲线图）
	trafficPlaceholder := widget.NewLabel("实时流量图（预留）")
	trafficArea := container.NewCenter(trafficPlaceholder)

	// 整体垂直排版，类似 UI.md 草图
	content := container.NewVBox(
		container.NewPadded(statusHeader),
		container.NewPadded(mainControlArea),
		nodeAndMode,
		container.NewPadded(trafficArea),
	)

	// 让内容在窗口中垂直居中一些，不要顶到上边缘
	return container.NewBorder(
		NewSpacer(SpacingLarge), // 顶部预留少量空白
		NewSpacer(SpacingLarge), // 底部预留少量空白
		nil,
		nil,
		container.NewCenter(content),
	)
}

// updateStatusIcon 更新状态图标
func (sp *StatusPanel) updateStatusIcon() {
	if sp.statusIcon == nil {
		return
	}
	
	isRunning := false
	if sp.appState != nil && sp.appState.XrayInstance != nil {
		isRunning = sp.appState.XrayInstance.IsRunning()
	}
	
	if isRunning {
		sp.statusIcon.SetResource(theme.ConfirmIcon())
	} else {
		sp.statusIcon.SetResource(theme.CancelIcon())
	}
}

// Refresh 刷新状态信息显示。
// 注意：由于使用了双向数据绑定，通常只需要更新绑定数据即可，UI 会自动更新。
// 此方法保留用于兼容性，实际更新通过 AppState.UpdateProxyStatus() 完成。
func (sp *StatusPanel) Refresh() {
	// 使用双向绑定后，只需要更新绑定数据，UI 会自动更新
	if sp.appState != nil {
		sp.appState.UpdateProxyStatus()
	}
	// 更新状态图标、延迟标签和主按钮
	sp.updateStatusIcon()
	sp.updateDelayLabel()
	sp.updateMainToggleButton()
	// 更新系统代理管理器的端口
	sp.updateSystemProxyPort()
}

// updateSystemProxyPort 更新系统代理管理器的端口
func (sp *StatusPanel) updateSystemProxyPort() {
	if sp.appState == nil || sp.appState.Config == nil {
		return
	}

	// 从配置或 xray 实例获取端口
	proxyPort := 10080 // 默认端口
	if sp.appState.XrayInstance != nil && sp.appState.XrayInstance.IsRunning() {
		if port := sp.appState.XrayInstance.GetPort(); port > 0 {
			proxyPort = port
		}
	} else if sp.appState.Config.AutoProxyPort > 0 {
		proxyPort = sp.appState.Config.AutoProxyPort
	}

	// 更新系统代理管理器
	sp.systemProxy = systemproxy.NewSystemProxy("127.0.0.1", proxyPort)
}

// updateDelayLabel 根据当前选中服务器更新延迟显示
func (sp *StatusPanel) updateDelayLabel() {
	if sp.delayLabel == nil || sp.appState == nil || sp.appState.ServerManager == nil {
		return
	}

	delayText := "延迟: -"
	if sp.appState.SelectedServerID != "" {
		if srv, err := sp.appState.ServerManager.GetServer(sp.appState.SelectedServerID); err == nil && srv != nil {
			if srv.Delay > 0 {
				delayText = fmt.Sprintf("延迟: %d ms", srv.Delay)
			} else if srv.Delay < 0 {
				delayText = "延迟: 测试失败"
			} else {
				delayText = "延迟: 未测"
			}
		}
	}
	sp.delayLabel.SetText(delayText)
}

// updateMainToggleButton 根据代理运行状态更新主开关按钮的文案
// 这里只负责 UI 文案，真正的启动/停止逻辑由 onToggleProxy 回调处理
func (sp *StatusPanel) updateMainToggleButton() {
	if sp.mainToggleButton == nil {
		return
	}

	isRunning := false
	if sp.appState != nil && sp.appState.XrayInstance != nil {
		isRunning = sp.appState.XrayInstance.IsRunning()
	}

	if isRunning {
		sp.mainToggleButton.SetText("● 已连接（点击断开）")
		sp.mainToggleButton.Importance = widget.HighImportance
	} else {
		sp.mainToggleButton.SetText("○ 未连接（点击连接）")
		sp.mainToggleButton.Importance = widget.MediumImportance
	}
}

// SetToggleHandler 设置主界面一键操作按钮的回调，由外部（如 MainWindow）注入。
// 回调内部可以根据当前状态决定是启动还是停止代理。
func (sp *StatusPanel) SetToggleHandler(handler func()) {
	sp.onToggleProxy = handler
}

// getFullModeName 将简短文本映射到完整的功能名称
func (sp *StatusPanel) getFullModeName(shortText string) string {
	switch shortText {
	case SystemProxyModeShortClear:
		return SystemProxyModeClear
	case SystemProxyModeShortAuto:
		return SystemProxyModeAuto
	case SystemProxyModeShortTerminal:
		return SystemProxyModeTerminal
	default:
		return shortText
	}
}

// getShortModeName 将完整的功能名称映射到简短文本
func (sp *StatusPanel) getShortModeName(fullName string) string {
	switch fullName {
	case SystemProxyModeClear:
		return SystemProxyModeShortClear
	case SystemProxyModeAuto:
		return SystemProxyModeShortAuto
	case SystemProxyModeTerminal:
		return SystemProxyModeShortTerminal
	default:
		return ""
	}
}

// applySystemProxyMode 应用系统代理模式（在启动时调用）
func (sp *StatusPanel) applySystemProxyMode(fullModeName string) error {
	if sp.appState == nil {
		return fmt.Errorf("appState 未初始化")
	}

	// 更新系统代理端口
	sp.updateSystemProxyPort()

	var err error
	var logMessage string

	switch fullModeName {
	case SystemProxyModeClear:
		// 清除系统代理
		err = sp.systemProxy.ClearSystemProxy()
		// 同时清除环境变量代理，避免污染环境
		terminalErr := sp.systemProxy.ClearTerminalProxy()
		if err == nil && terminalErr == nil {
			logMessage = "已清除系统代理设置和环境变量代理"
		} else if err != nil && terminalErr != nil {
			logMessage = fmt.Sprintf("清除系统代理失败: %v; 清除环境变量代理失败: %v", err, terminalErr)
			err = fmt.Errorf("清除失败: %v; %v", err, terminalErr)
		} else if err != nil {
			logMessage = fmt.Sprintf("清除系统代理失败: %v; 已清除环境变量代理", err)
		} else {
			logMessage = fmt.Sprintf("已清除系统代理设置; 清除环境变量代理失败: %v", terminalErr)
			err = terminalErr
		}

	case SystemProxyModeAuto:
		// 先清除之前的代理设置，再设置新的
		_ = sp.systemProxy.ClearSystemProxy()
		_ = sp.systemProxy.ClearTerminalProxy()
		// 然后设置系统代理
		err = sp.systemProxy.SetSystemProxy()
		if err == nil {
			proxyPort := 10080
			if sp.appState.XrayInstance != nil && sp.appState.XrayInstance.IsRunning() {
				if port := sp.appState.XrayInstance.GetPort(); port > 0 {
					proxyPort = port
				}
			} else if sp.appState.Config != nil && sp.appState.Config.AutoProxyPort > 0 {
				proxyPort = sp.appState.Config.AutoProxyPort
			}
			logMessage = fmt.Sprintf("已自动配置系统代理: 127.0.0.1:%d", proxyPort)
		} else {
			logMessage = fmt.Sprintf("自动配置系统代理失败: %v", err)
		}

	case SystemProxyModeTerminal:
		// 先清除之前的代理设置
		_ = sp.systemProxy.ClearSystemProxy()
		_ = sp.systemProxy.ClearTerminalProxy()
		// 然后设置环境变量代理
		err = sp.systemProxy.SetTerminalProxy()
		if err == nil {
			proxyPort := 10080
			if sp.appState.XrayInstance != nil && sp.appState.XrayInstance.IsRunning() {
				if port := sp.appState.XrayInstance.GetPort(); port > 0 {
					proxyPort = port
				}
			} else if sp.appState.Config != nil && sp.appState.Config.AutoProxyPort > 0 {
				proxyPort = sp.appState.Config.AutoProxyPort
			}
			logMessage = fmt.Sprintf("已设置环境变量代理: socks5://127.0.0.1:%d (已写入shell配置文件)", proxyPort)
		} else {
			logMessage = fmt.Sprintf("设置环境变量代理失败: %v", err)
		}
	}

	// 输出日志到日志区域
	if err == nil {
		sp.appState.AppendLog("INFO", "app", logMessage)
		if sp.appState.Logger != nil {
			sp.appState.Logger.InfoWithType(logging.LogTypeApp, logMessage)
		}
	} else {
		sp.appState.AppendLog("ERROR", "app", logMessage)
		if sp.appState.Logger != nil {
			sp.appState.Logger.Error(logMessage)
		}
	}

	return err
}

// saveSystemProxyState 保存系统代理状态到数据库
func (sp *StatusPanel) saveSystemProxyState(mode string) {
	if err := database.SetAppConfig("systemProxyMode", mode); err != nil {
		if sp.appState != nil && sp.appState.Logger != nil {
			sp.appState.Logger.Error("保存系统代理状态失败: %v", err)
		}
	}
}

// restoreSystemProxyState 从数据库恢复系统代理状态（在应用启动时调用）
func (sp *StatusPanel) restoreSystemProxyState() {
	// 从数据库读取保存的系统代理模式
	mode, err := database.GetAppConfig("systemProxyMode")
	if err != nil || mode == "" {
		// 如果没有保存的状态，不执行任何操作
		return
	}

	// 应用系统代理模式
	restoreErr := sp.applySystemProxyMode(mode)

	// 更新下拉框显示文本（使用简短文本）
	if restoreErr == nil {
		shortText := sp.getShortModeName(mode)
		if shortText != "" {
			sp.proxyModeSelect.SetSelected(shortText)
		}
	}
}
