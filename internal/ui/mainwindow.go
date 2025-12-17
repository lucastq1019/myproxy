package ui

import (
	"encoding/json"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"myproxy.com/p/internal/database"
)

// LayoutConfig 存储窗口布局的配置信息，包括各区域的分割比例。
// 这些配置会持久化到数据库中，以便在应用重启后恢复用户的布局偏好。
type LayoutConfig struct {
	SubscriptionOffset float64 `json:"subscriptionOffset"` // 订阅管理区域比例 (默认0.2 = 20%)
	ServerListOffset   float64 `json:"serverListOffset"`   // 服务器列表比例 (默认0.6667 = 66.7% of 75%)
	StatusOffset       float64 `json:"statusOffset"`       // 状态信息比例 (默认0.9375 = 93.75% of 80%, 即5% of total)
}

// DefaultLayoutConfig 返回默认的布局配置。
// 默认布局：订阅管理 20%，服务器列表 50%，日志 25%，状态信息 5%。
func DefaultLayoutConfig() *LayoutConfig {
	return &LayoutConfig{
		SubscriptionOffset: 0.2,    // 20%
		ServerListOffset:   0.6667, // 66.7% of 75% = 50% of total
		StatusOffset:       0.9375, // 93.75% of 80% = 75% of total, 剩余5%
	}
}

// MainWindow 管理主窗口的布局和各个面板组件。
// 它负责协调订阅管理、服务器列表、日志显示和状态信息四个主要区域的显示。
type MainWindow struct {
	appState          *AppState
	subscriptionPanel *SubscriptionPanel
	serverListPanel   *ServerListPanel
	logsPanel         *LogsPanel
	statusPanel       *StatusPanel
	mainSplit         *container.Split // 主分割容器（服务器列表和日志，保留用于日志面板独立窗口等场景）
	layoutConfig      *LayoutConfig    // 布局配置

	// 单窗口多页面：通过 SetContent() 在一个窗口内切换不同的 Container
	homePage     fyne.CanvasObject // 主界面（极简一键开关）
	nodePage     fyne.CanvasObject // 节点列表页面
	settingsPage fyne.CanvasObject // 设置页面
}

// NewMainWindow 创建并初始化主窗口。
// 该方法会加载布局配置、创建各个面板组件，并建立它们之间的关联。
// 参数：
//   - appState: 应用状态实例
//
// 返回：初始化后的主窗口实例
func NewMainWindow(appState *AppState) *MainWindow {
	mw := &MainWindow{
		appState: appState,
	}

	// 加载布局配置
	mw.loadLayoutConfig()

	// 创建各个面板
	mw.subscriptionPanel = NewSubscriptionPanel(appState)
	mw.serverListPanel = NewServerListPanel(appState)
	mw.logsPanel = NewLogsPanel(appState)
	mw.statusPanel = NewStatusPanel(appState)

	// 设置状态面板引用，以便服务器列表可以刷新状态
	mw.serverListPanel.SetStatusPanel(mw.statusPanel)

	// 设置主窗口和日志面板引用到 AppState，以便其他组件可以刷新日志面板
	appState.MainWindow = mw
	appState.LogsPanel = mw.logsPanel

	return mw
}

// loadLayoutConfig 从数据库加载布局配置
func (mw *MainWindow) loadLayoutConfig() {
	configJSON, err := database.GetLayoutConfig("layout_config")
	if err != nil || configJSON == "" {
		// 如果没有配置，使用默认配置并保存
		mw.layoutConfig = DefaultLayoutConfig()
		mw.saveLayoutConfig()
		return
	}

	// 解析配置
	var config LayoutConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		// 解析失败，使用默认配置
		mw.layoutConfig = DefaultLayoutConfig()
		mw.saveLayoutConfig()
		return
	}

	mw.layoutConfig = &config
}

// saveLayoutConfig 保存布局配置到数据库
func (mw *MainWindow) saveLayoutConfig() {
	if mw.layoutConfig == nil {
		mw.layoutConfig = DefaultLayoutConfig()
	}

	configJSON, err := json.Marshal(mw.layoutConfig)
	if err != nil {
		return
	}

	database.SetLayoutConfig("layout_config", string(configJSON))
}

// Build 构建并返回主窗口的 UI 组件树。
// 该方法使用自定义 Border 布局，支持百分比控制各区域的大小。
// 返回：主窗口的根容器组件
func (mw *MainWindow) Build() fyne.CanvasObject {
	// 新主界面：遵循 UI 设计规范，采用“单窗口 + 多页面”设计。
	// 通过 Window.SetContent() 在 homePage / nodePage / settingsPage 之间切换。

	// 初始化各页面（home/node/settings）
	mw.initPages()

	// 默认返回 homePage 作为初始内容
	if mw.homePage != nil {
		return mw.homePage
	}
	return container.NewWithoutLayout()
}

// Refresh 刷新主窗口的所有面板，包括服务器列表、日志显示和订阅管理。
// 该方法会更新数据绑定，使 UI 自动反映最新的应用状态。
// 注意：此方法包含安全检查，防止在窗口移动/缩放时出现空指针错误。
func (mw *MainWindow) Refresh() {
	// 安全检查：确保所有面板都已初始化
	if mw.serverListPanel != nil {
		mw.serverListPanel.Refresh()
	}
	if mw.logsPanel != nil {
		mw.logsPanel.Refresh() // 刷新日志面板，显示最新日志
	}
	if mw.subscriptionPanel != nil {
		mw.subscriptionPanel.refreshSubscriptionList()
	}
	// 使用双向绑定，只需更新绑定数据，UI 会自动更新
	if mw.appState != nil {
		mw.appState.UpdateProxyStatus()
		mw.appState.UpdateSubscriptionLabels() // 更新订阅标签绑定
	}
}

// SaveLayoutConfig 保存当前的布局配置到数据库。
// 该方法会在窗口关闭时自动调用，以保存用户的布局偏好。
func (mw *MainWindow) SaveLayoutConfig() {
	if mw.mainSplit != nil {
		mw.layoutConfig.ServerListOffset = mw.mainSplit.Offset
	}
	// 布局比例由 customLayout 控制，配置保存到数据库
	mw.saveLayoutConfig()
}

// GetLayoutConfig 返回当前的布局配置。
// 返回：布局配置实例，如果未初始化则返回默认配置
func (mw *MainWindow) GetLayoutConfig() *LayoutConfig {
	return mw.layoutConfig
}

// UpdateLogsCollapseState 更新日志折叠状态并调整布局
func (mw *MainWindow) UpdateLogsCollapseState(isCollapsed bool) {
	if mw.mainSplit == nil {
		return
	}
	
	if isCollapsed {
		// 折叠：将偏移设置为接近 1.0，使日志区域几乎不可见
		mw.mainSplit.Offset = 0.99
	} else {
		// 展开：恢复保存的分割位置
		if mw.layoutConfig != nil && mw.layoutConfig.ServerListOffset > 0 {
			mw.mainSplit.Offset = mw.layoutConfig.ServerListOffset
		} else {
			mw.mainSplit.Offset = 0.6667
		}
	}
	
	// 刷新分割容器
	mw.mainSplit.Refresh()
}

// initPages 初始化单窗口的三个页面：home / node / settings
func (mw *MainWindow) initPages() {
	// 主界面（homePage）：极简状态 + 一键主开关
	mw.homePage = mw.buildHomePage()

	// 节点列表页面（nodePage）：顶部返回 + 标题，下方为服务器列表
	mw.nodePage = mw.buildNodePage()

	// 设置页面（settingsPage）：顶部返回 + 标题，下方预留设置内容
	mw.settingsPage = mw.buildSettingsPage()
}

// buildHomePage 构建主界面 Container（homePage）
func (mw *MainWindow) buildHomePage() fyne.CanvasObject {
	if mw.statusPanel == nil {
		return container.NewWithoutLayout()
	}

	statusArea := mw.statusPanel.Build()
	if statusArea == nil {
		statusArea = container.NewWithoutLayout()
	}

	// 顶部标题栏：左侧应用名称，右侧为“节点”和“设置”入口
	titleLabel := NewTitleLabel("SOCKS5 代理客户端")
	headerButtons := container.NewHBox(
		NewStyledButton("节点", theme.NavigateNextIcon(), func() {
			mw.ShowNodePage()
		}),
		NewSpacer(SpacingSmall),
		NewStyledButton("设置", theme.SettingsIcon(), func() {
			mw.ShowSettingsPage()
		}),
	)
	headerBar := container.NewPadded(container.NewHBox(
		titleLabel,
		layout.NewSpacer(),
		headerButtons,
	))

	// 中部内容：状态面板（内部负责实现“一键主开关 + 状态 + 节点 + 模式 + 流量图占位”）
	centerContent := container.NewCenter(statusArea)

	return container.NewBorder(
		headerBar,
		nil,
		nil,
		nil,
		centerContent,
	)
}

// buildNodePage 构建节点列表页面 Container（nodePage）
func (mw *MainWindow) buildNodePage() fyne.CanvasObject {
	if mw.serverListPanel == nil {
		return container.NewWithoutLayout()
	}

	// 顶部栏：返回主界面 + 标题
	backBtn := NewStyledButton("← 返回", nil, func() {
		mw.ShowHomePage()
	})
	titleLabel := NewTitleLabel("节点列表")
	headerBar := container.NewPadded(container.NewHBox(
		backBtn,
		NewSpacer(SpacingLarge),
		titleLabel,
		layout.NewSpacer(),
	))

	listContent := mw.serverListPanel.Build()
	if listContent == nil {
		listContent = container.NewWithoutLayout()
	}

	return container.NewBorder(
		headerBar,
		nil,
		nil,
		nil,
		listContent,
	)
}

// buildSettingsPage 构建设置页面 Container（settingsPage）
func (mw *MainWindow) buildSettingsPage() fyne.CanvasObject {
	// 顶部栏：返回主界面 + 标题
	backBtn := NewStyledButton("← 返回", nil, func() {
		mw.ShowHomePage()
	})
	titleLabel := NewTitleLabel("设置")
	headerBar := container.NewPadded(container.NewHBox(
		backBtn,
		NewSpacer(SpacingLarge),
		titleLabel,
		layout.NewSpacer(),
	))

	// 这里暂时使用占位内容，后续可以替换为真正的设置视图
	placeholder := widget.NewLabel("设置界面开发中（Settings View Placeholder）")
	center := container.NewCenter(placeholder)

	return container.NewBorder(
		headerBar,
		nil,
		nil,
		nil,
		center,
	)
}

// ShowHomePage 切换到主界面（homePage）
func (mw *MainWindow) ShowHomePage() {
	if mw == nil || mw.appState == nil || mw.appState.Window == nil {
		return
	}
	if mw.homePage == nil {
		mw.homePage = mw.buildHomePage()
	}
	mw.appState.Window.SetContent(mw.homePage)
}

// ShowNodePage 切换到节点列表页面（nodePage）
func (mw *MainWindow) ShowNodePage() {
	if mw == nil || mw.appState == nil || mw.appState.Window == nil {
		return
	}
	if mw.nodePage == nil {
		mw.nodePage = mw.buildNodePage()
	}
	mw.appState.Window.SetContent(mw.nodePage)
}

// ShowSettingsPage 切换到设置页面（settingsPage）
func (mw *MainWindow) ShowSettingsPage() {
	if mw == nil || mw.appState == nil || mw.appState.Window == nil {
		return
	}
	if mw.settingsPage == nil {
		mw.settingsPage = mw.buildSettingsPage()
	}
	mw.appState.Window.SetContent(mw.settingsPage)
}
