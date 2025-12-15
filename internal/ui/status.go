package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// StatusPanel 显示代理状态、端口和当前服务器信息。
// 它使用 Fyne 的双向数据绑定机制，当应用状态更新时自动刷新显示。
type StatusPanel struct {
	appState         *AppState
	proxyStatusLabel *widget.Label
	portLabel        *widget.Label
	serverNameLabel  *widget.Label
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

	return sp
}

// Build 构建并返回状态信息面板的 UI 组件。
// 返回：包含代理状态、端口和服务器名称的水平布局容器
func (sp *StatusPanel) Build() fyne.CanvasObject {
	// 使用水平布局显示所有信息，所有元素横向排列
	// 使用 HBox 布局，元素从左到右排列，保持最小尺寸
	statusArea := container.NewHBox(
		sp.proxyStatusLabel,
		widget.NewSeparator(), // 分隔符
		sp.portLabel,
		widget.NewSeparator(), // 分隔符
		sp.serverNameLabel,
	)

	// 使用 Border 布局，顶部添加分隔线，确保区域可见
	// Border 布局：top=分隔线，center=状态信息内容（水平布局）
	result := container.NewBorder(
		widget.NewSeparator(), // 顶部：分隔线
		nil,                   // 底部：无
		nil,                   // 左侧：无
		nil,                   // 右侧：无
		statusArea,            // 中间：状态信息内容（HBox 水平布局）
	)

	return result
}

// Refresh 刷新状态信息显示。
// 注意：由于使用了双向数据绑定，通常只需要更新绑定数据即可，UI 会自动更新。
// 此方法保留用于兼容性，实际更新通过 AppState.UpdateProxyStatus() 完成。
func (sp *StatusPanel) Refresh() {
	// 使用双向绑定后，只需要更新绑定数据，UI 会自动更新
	if sp.appState != nil {
		sp.appState.UpdateProxyStatus()
	}
}
