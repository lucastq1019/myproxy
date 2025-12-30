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

// PageType 页面类型枚举
type PageType int

const (
	PageTypeHome PageType = iota // 主界面
	PageTypeNode                 // 节点列表页面
	PageTypeSettings             // 设置页面
	PageTypeSubscription         // 订阅管理页面
)

// PageStack 路由栈结构，用于管理页面导航历史
type PageStack struct {
	stack []PageType // 页面栈
}

// NewPageStack 创建新的路由栈
func NewPageStack() *PageStack {
	return &PageStack{
		stack: make([]PageType, 0),
	}
}

// Push 将页面压入栈中
func (ps *PageStack) Push(pageType PageType) {
	ps.stack = append(ps.stack, pageType)
}

// Pop 从栈中弹出页面，如果栈为空返回 false
func (ps *PageStack) Pop() (PageType, bool) {
	if len(ps.stack) == 0 {
		return PageTypeHome, false
	}
	lastIndex := len(ps.stack) - 1
	pageType := ps.stack[lastIndex]
	ps.stack = ps.stack[:lastIndex]
	return pageType, true
}

// Clear 清空路由栈
func (ps *PageStack) Clear() {
	ps.stack = ps.stack[:0]
}

// IsEmpty 检查栈是否为空
func (ps *PageStack) IsEmpty() bool {
	return len(ps.stack) == 0
}

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
	logsPanel         *LogsPanel
	statusPanel       *StatusPanel
	mainSplit         *container.Split // 主分割容器（服务器列表和日志，保留用于日志面板独立窗口等场景）
	layoutConfig      *LayoutConfig    // 布局配置
	pageStack         *PageStack      // 路由栈，用于管理页面导航历史
	currentPage       PageType        // 当前页面类型

	// 单窗口多页面：通过 SetContent() 在一个窗口内切换不同的 Container
	homePage         fyne.CanvasObject // 主界面（极简一键开关）

	nodePage         fyne.CanvasObject // 节点列表页面
	nodePageInstance *NodePage // 节点列表页面实例
	
	settingsPage     fyne.CanvasObject // 设置页面


	subscriptionPage fyne.CanvasObject // 订阅管理页面
	subscriptionPageInstance *SubscriptionPage // 订阅管理页面实例
}

// NewMainWindow 创建并初始化主窗口。
// 该方法会加载布局配置、创建各个面板组件，并建立它们之间的关联。
// 参数：
//   - appState: 应用状态实例
//
// 返回：初始化后的主窗口实例
func NewMainWindow(appState *AppState) *MainWindow {
	mw := &MainWindow{
		appState:   appState,
		pageStack:  NewPageStack(),
		currentPage: PageTypeHome,
	}

	// 加载布局配置
	mw.loadLayoutConfig()

	// 创建各个面板
	// mw.serverListPanel = NewServerListPanel(appState)
	mw.logsPanel = NewLogsPanel(appState)
	mw.statusPanel = NewStatusPanel(appState)

	// 设置状态面板引用，以便服务器列表可以刷新状态
	// mw.serverListPanel.SetStatusPanel(mw.statusPanel)

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
	// if mw.serverListPanel != nil {
	// 	mw.serverListPanel.Refresh()
	// }
	if mw.logsPanel != nil {
		mw.logsPanel.Refresh() // 刷新日志面板，显示最新日志
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

// initPages 初始化单窗口的四个页面：home / node / settings / subscription
func (mw *MainWindow) initPages() {
	// 主界面（homePage）：极简状态 + 一键主开关
	mw.homePage = mw.buildHomePage()

	// 设置页面（settingsPage）：顶部返回 + 标题，下方预留设置内容
	mw.settingsPage = mw.buildSettingsPage()

	// 节点列表页面（nodePage）：服务器列表和管理功能
	mw.nodePageInstance = NewNodePage(mw.appState)
	mw.nodePage = mw.nodePageInstance.Build()

	// 订阅管理页面（subscriptionPage）：订阅列表和管理功能
	mw.subscriptionPageInstance = NewSubscriptionPage(mw.appState)
	mw.subscriptionPage = mw.subscriptionPageInstance.Build()
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
	// 顶部标题栏：右侧仅保留设置入口（符合 UI.md 设计：设置入口据右侧）
	headerButtons := container.NewHBox(
		layout.NewSpacer(),
		NewStyledButton("设置", theme.SettingsIcon(), func() {
			mw.ShowSettingsPage()
		}),
	)
	headerBar := container.NewPadded(headerButtons)

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


// buildSettingsPage 构建设置页面 Container（settingsPage）
func (mw *MainWindow) buildSettingsPage() fyne.CanvasObject {
	// 顶部栏：返回上一个页面 + 标题
	backBtn := widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() {
		mw.Back()
	})
	backBtn.Importance = widget.LowImportance
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

// showPage 通用的页面切换方法，会将当前页面压入栈，然后切换到新页面
func (mw *MainWindow) showPage(pageType PageType, pageContent fyne.CanvasObject, pushCurrent bool) {
	if mw == nil || mw.appState == nil || mw.appState.Window == nil {
		return
	}
	
	// 如果需要压入当前页面（通常从其他页面跳转时需要）
	if pushCurrent && mw.currentPage != pageType {
		mw.pageStack.Push(mw.currentPage)
	}
	
	// 更新当前页面类型
	mw.currentPage = pageType
	
	// 设置内容
	mw.appState.Window.SetContent(pageContent)
	
	// 从数据库读取窗口大小并应用（在SetContent之后，避免内容的最小尺寸要求导致窗口变大）
	defaultSize := fyne.NewSize(420, 520)
	windowSize := LoadWindowSize(defaultSize)
	mw.appState.Window.Resize(windowSize)
	// 保存当前窗口大小到数据库（确保保存的是设置后的尺寸）
	SaveWindowSize(windowSize)
}

// Back 返回到上一个页面（从路由栈中弹出）
func (mw *MainWindow) Back() {
	if mw == nil || mw.appState == nil || mw.appState.Window == nil {
		return
	}
	
	// 从栈中弹出上一个页面
	prevPageType, ok := mw.pageStack.Pop()
	if !ok {
		// 如果栈为空，默认返回主界面（不压栈）
		mw.navigateToPage(PageTypeHome, false)
		return
	}
	
	// 切换到上一个页面（不压栈，因为这是返回操作）
	mw.navigateToPage(prevPageType, false)
}

// navigateToPage 导航到指定页面（内部方法，不压栈）
func (mw *MainWindow) navigateToPage(pageType PageType, pushCurrent bool) {
	var pageContent fyne.CanvasObject
	
	switch pageType {
	case PageTypeHome:
		if mw.homePage == nil {
			mw.homePage = mw.buildHomePage()
		}
		pageContent = mw.homePage
	case PageTypeNode:
		if mw.nodePage == nil {
			mw.nodePage = mw.nodePageInstance.Build()
		}
		// 刷新服务器列表
		// if mw.serverListPanel != nil {
		// 	mw.serverListPanel.Refresh()
		// }
		pageContent = mw.nodePage
	case PageTypeSettings:
		if mw.settingsPage == nil {
			mw.settingsPage = mw.buildSettingsPage()
		}
		pageContent = mw.settingsPage
	case PageTypeSubscription:
		if mw.subscriptionPage == nil {
			mw.subscriptionPageInstance = NewSubscriptionPage(mw.appState)
			mw.subscriptionPage = mw.subscriptionPageInstance.Build()
		}
		// 刷新订阅列表
		if mw.subscriptionPageInstance != nil {
			mw.subscriptionPageInstance.Refresh()
		}
		pageContent = mw.subscriptionPage
	default:
		// 未知页面类型，返回主界面
		if mw.homePage == nil {
			mw.homePage = mw.buildHomePage()
		}
		pageContent = mw.homePage
		pageType = PageTypeHome
	}
	
	mw.showPage(pageType, pageContent, pushCurrent)
}

// ShowHomePage 切换到主界面（homePage）
func (mw *MainWindow) ShowHomePage() {
	mw.navigateToPage(PageTypeHome, true)
}

// ShowNodePage 切换到节点列表页面（nodePage）
func (mw *MainWindow) ShowNodePage() {
	mw.navigateToPage(PageTypeNode, true)
}

// ShowSettingsPage 切换到设置页面（settingsPage）
func (mw *MainWindow) ShowSettingsPage() {
	mw.navigateToPage(PageTypeSettings, true)
}

// ShowSubscriptionPage 切换到订阅管理页面（subscriptionPage）
func (mw *MainWindow) ShowSubscriptionPage() {
	mw.navigateToPage(PageTypeSubscription, true)
}

