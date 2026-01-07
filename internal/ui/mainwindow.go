package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"myproxy.com/p/internal/logging"
	"myproxy.com/p/internal/service"
	"myproxy.com/p/internal/store"
	"myproxy.com/p/internal/systemproxy"
)

// proxyModeButtonLayout 自定义布局，确保三个按钮平分宽度
type proxyModeButtonLayout struct{}

func (p *proxyModeButtonLayout) Layout(objects []fyne.CanvasObject, containerSize fyne.Size) {
	if len(objects) != 3 {
		return
	}

	// 三个按钮平分宽度，每个占 1/3
	// 使用较小的间距，Mac 简约风格
	spacing := float32(4) // 按钮之间的间距
	totalSpacing := spacing * 2 // 两个间距
	availableWidth := containerSize.Width - totalSpacing
	buttonWidth := availableWidth / 3

	for i, obj := range objects {
		if obj != nil {
			// 计算每个按钮的位置：前面按钮的宽度 + 间距
			x := float32(i) * (buttonWidth + spacing)
			obj.Resize(fyne.NewSize(buttonWidth, containerSize.Height))
			obj.Move(fyne.NewPos(x, 0))
		}
	}
}

func (p *proxyModeButtonLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) < 3 {
		return fyne.NewSize(0, 0)
	}

	// 最小宽度：三个按钮的最小宽度之和
	minWidth := float32(0)
	minHeight := float32(0)
	for _, obj := range objects {
		if obj != nil {
			size := obj.MinSize()
			minWidth += size.Width
			if size.Height > minHeight {
				minHeight = size.Height
			}
		}
	}
	// 加上按钮间距
	minWidth += 2 * 4 // 两个间距

	return fyne.NewSize(minWidth, minHeight)
}

// modeButtonLayout 自定义布局，确保模式按钮组占90%宽度
type modeButtonLayout struct{}

func (m *modeButtonLayout) Layout(objects []fyne.CanvasObject, containerSize fyne.Size) {
	if len(objects) != 2 {
		return
	}
	
	iconArea := objects[0]
	buttonArea := objects[1]
	
	// 图标区域：占10%宽度
	iconWidth := containerSize.Width * 0.1
	if iconArea != nil {
		iconArea.Resize(fyne.NewSize(iconWidth, containerSize.Height))
		iconArea.Move(fyne.NewPos(0, 0))
	}
	
	// 按钮组区域：占90%宽度，从10%位置开始
	buttonWidth := containerSize.Width * 0.9
	buttonX := containerSize.Width * 0.1
	if buttonArea != nil {
		buttonArea.Resize(fyne.NewSize(buttonWidth, containerSize.Height))
		buttonArea.Move(fyne.NewPos(buttonX, 0))
	}
}

func (m *modeButtonLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) < 2 {
		return fyne.NewSize(0, 0)
	}
	
	iconMin := objects[0].MinSize()
	buttonMin := objects[1].MinSize()
	
	// 最小宽度：图标区域最小宽度 + 按钮组区域最小宽度（按比例）
	totalWidth := fyne.Max(iconMin.Width/0.1, buttonMin.Width/0.9)
	return fyne.NewSize(totalWidth, fyne.Max(iconMin.Height, buttonMin.Height))
}

// nodeNameLayout 自定义布局，确保节点名称区域占90%宽度
type nodeNameLayout struct{}

func (n *nodeNameLayout) Layout(objects []fyne.CanvasObject, containerSize fyne.Size) {
	if len(objects) != 2 {
		return
	}
	
	iconArea := objects[0]
	nameArea := objects[1]
	
	// 图标区域：占10%宽度
	iconWidth := containerSize.Width * 0.1
	if iconArea != nil {
		iconArea.Resize(fyne.NewSize(iconWidth, containerSize.Height))
		iconArea.Move(fyne.NewPos(0, 0))
	}
	
	// 节点名称区域：占90%宽度，从10%位置开始
	nameWidth := containerSize.Width * 0.9
	nameX := containerSize.Width * 0.1
	if nameArea != nil {
		nameArea.Resize(fyne.NewSize(nameWidth, containerSize.Height))
		nameArea.Move(fyne.NewPos(nameX, 0))
	}
}

func (n *nodeNameLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) < 2 {
		return fyne.NewSize(0, 0)
	}
	
	iconMin := objects[0].MinSize()
	nameMin := objects[1].MinSize()
	
	// 最小宽度：图标区域最小宽度 + 节点名称区域最小宽度（按比例）
	// 如果图标区域最小宽度为 w，则总宽度至少为 w / 0.1
	// 如果节点名称区域最小宽度为 w，则总宽度至少为 w / 0.9
	totalWidth := fyne.Max(iconMin.Width/0.1, nameMin.Width/0.9)
	return fyne.NewSize(totalWidth, fyne.Max(iconMin.Height, nameMin.Height))
}

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
// 注意：此类型已迁移到 store 包，这里保留作为类型别名以便兼容。
type LayoutConfig = store.LayoutConfig

// DefaultLayoutConfig 返回默认的布局配置。
// 注意：此函数已迁移到 store 包，这里保留作为便捷函数。
func DefaultLayoutConfig() *LayoutConfig {
	return store.DefaultLayoutConfig()
}

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

// MainWindow 管理主窗口的布局和各个面板组件。
// 它负责协调订阅管理、服务器列表、日志显示和状态信息四个主要区域的显示。
type MainWindow struct {
	appState          *AppState
	logsPanel         *LogsPanel
	mainSplit         *container.Split // 主分割容器（服务器列表和日志，保留用于日志面板独立窗口等场景）
	pageStack         *PageStack      // 路由栈，用于管理页面导航历史
	currentPage       PageType        // 当前页面类型

	// 单窗口多页面：通过 SetContent() 在一个窗口内切换不同的 Container
	homePage         fyne.CanvasObject // 主界面（极简一键开关）

	nodePage         fyne.CanvasObject // 节点列表页面
	nodePageInstance *NodePage // 节点列表页面实例
	
	settingsPage     fyne.CanvasObject // 设置页面

	subscriptionPage fyne.CanvasObject // 订阅管理页面
	subscriptionPageInstance *SubscriptionPage // 订阅管理页面实例

	// 主界面状态UI组件（使用双向绑定）
	mainToggleButton *widget.Button      // 主开关按钮（连接/断开）
	proxyStatusLabel *widget.Label        // 代理状态标签（绑定到 ProxyStatusBinding）
	portLabel        *widget.Label        // 端口标签（绑定到 PortBinding）
	serverNameLabel  *widget.Label        // 服务器名称标签（绑定到 ServerNameBinding）
	delayLabel       *widget.Label        // 延迟标签
	proxyModeButtons [3]*widget.Button    // 系统代理模式按钮组（清除、系统、终端）
	systemProxy      *systemproxy.SystemProxy // 系统代理管理器
	
	// 状态标志
	systemProxyRestored bool // 标记系统代理状态是否已恢复（避免重复恢复）
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

	// 布局配置由 Store 管理，无需在这里加载

	// 创建各个面板
	mw.logsPanel = NewLogsPanel(appState)

	// 创建系统代理管理器（默认使用 localhost:10080）
	mw.systemProxy = systemproxy.NewSystemProxy("127.0.0.1", 10080)

	// 设置主窗口和日志面板引用到 AppState，以便其他组件可以刷新日志面板
	appState.MainWindow = mw
	appState.LogsPanel = mw.logsPanel

	// 注意：系统代理状态的恢复将在 buildHomePage() 中完成
	// 因为需要先创建 proxyModeRadio 组件

	return mw
}

// loadLayoutConfig 从 Store 加载布局配置（Store 已经管理，这里只是确保数据最新）
func (mw *MainWindow) loadLayoutConfig() {
	if mw.appState != nil && mw.appState.Store != nil && mw.appState.Store.Layout != nil {
		_ = mw.appState.Store.Layout.Load()
	}
}

// saveLayoutConfig 保存布局配置到 Store
func (mw *MainWindow) saveLayoutConfig() {
	if mw.appState == nil || mw.appState.Store == nil || mw.appState.Store.Layout == nil {
		return
	}
	config := mw.GetLayoutConfig()
	_ = mw.appState.Store.Layout.Save(config)
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
	if mw.logsPanel != nil {
		mw.logsPanel.Refresh() // 刷新日志面板，显示最新日志
	}
	// 使用双向绑定，只需更新绑定数据，UI 会自动更新
	if mw.appState != nil {
		mw.appState.UpdateProxyStatus() // 更新绑定数据（proxyStatusLabel, portLabel, serverNameLabel 会自动更新）
		if mw.mainToggleButton != nil {
			mw.updateMainToggleButton() 
		}
		// 订阅标签绑定由 Store 自动管理，无需手动更新
	}
}

// SaveLayoutConfig 保存当前的布局配置到 Store。
// 该方法会在窗口关闭时自动调用，以保存用户的布局偏好。
func (mw *MainWindow) SaveLayoutConfig() {
	if mw.appState == nil || mw.appState.Store == nil || mw.appState.Store.Layout == nil {
		return
	}
	
	config := mw.GetLayoutConfig()
	if mw.mainSplit != nil {
		config.ServerListOffset = mw.mainSplit.Offset
	}
	_ = mw.appState.Store.Layout.Save(config)
}

// GetLayoutConfig 返回当前的布局配置。
// 返回：布局配置实例，如果未初始化则返回默认配置
func (mw *MainWindow) GetLayoutConfig() *LayoutConfig {
	if mw.appState != nil && mw.appState.Store != nil && mw.appState.Store.Layout != nil {
		return mw.appState.Store.Layout.Get()
	}
	return DefaultLayoutConfig()
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
		config := mw.GetLayoutConfig()
		if config != nil && config.ServerListOffset > 0 {
			mw.mainSplit.Offset = config.ServerListOffset
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
// 使用双向绑定直接构建状态UI，不再依赖 StatusPanel
func (mw *MainWindow) buildHomePage() fyne.CanvasObject {
	if mw.appState == nil {
		return container.NewWithoutLayout()
	}

	// 创建状态标签（使用双向绑定）
	if mw.proxyStatusLabel == nil {
		if mw.appState.ProxyStatusBinding != nil {
			mw.proxyStatusLabel = widget.NewLabelWithData(mw.appState.ProxyStatusBinding)
		} else {
			mw.proxyStatusLabel = widget.NewLabel("代理状态: 未知")
		}
		mw.proxyStatusLabel.Wrapping = fyne.TextWrapOff
	}

	if mw.portLabel == nil {
		if mw.appState.PortBinding != nil {
			mw.portLabel = widget.NewLabelWithData(mw.appState.PortBinding)
		} else {
			mw.portLabel = widget.NewLabel("监听端口: -")
		}
		mw.portLabel.Wrapping = fyne.TextWrapOff
	}

	if mw.serverNameLabel == nil {
		if mw.appState.ServerNameBinding != nil {
			mw.serverNameLabel = widget.NewLabelWithData(mw.appState.ServerNameBinding)
		} else {
			mw.serverNameLabel = widget.NewLabel("当前服务器: 无")
		}
		// 横向显示，不换行，只有在超过90%空间时才显示省略号
		mw.serverNameLabel.Wrapping = fyne.TextWrapOff
		mw.serverNameLabel.Truncation = fyne.TextTruncateEllipsis
	}

	if mw.delayLabel == nil {
		mw.delayLabel = widget.NewLabel("-")
		mw.delayLabel.Wrapping = fyne.TextWrapOff
	}

	// 创建状态图标
	statusIcon := widget.NewIcon(theme.CancelIcon())
	mw.updateStatusIcon(statusIcon)

	// 创建主开关按钮
	if mw.mainToggleButton == nil {
		mw.mainToggleButton = widget.NewButton("", mw.onToggleProxy)
		mw.mainToggleButton.Importance = widget.HighImportance
		mw.mainToggleButton.Resize(fyne.NewSize(120, 120))
		mw.updateMainToggleButton()
	}

	// 创建系统代理模式按钮组（三个按钮平分宽度）
	if mw.proxyModeButtons[0] == nil {
		// 创建三个按钮
		mw.proxyModeButtons[0] = widget.NewButton(SystemProxyModeShortClear, func() {
			mw.onProxyModeButtonClicked(SystemProxyModeClear)
		})
		mw.proxyModeButtons[1] = widget.NewButton(SystemProxyModeShortAuto, func() {
			mw.onProxyModeButtonClicked(SystemProxyModeAuto)
		})
		mw.proxyModeButtons[2] = widget.NewButton(SystemProxyModeShortTerminal, func() {
			mw.onProxyModeButtonClicked(SystemProxyModeTerminal)
		})
		
		// 设置按钮初始重要性（所有按钮初始为 LowImportance，选中状态由 updateProxyModeButtonsState 管理）
		for i := range mw.proxyModeButtons {
			mw.proxyModeButtons[i].Importance = widget.LowImportance
		}
		
		// 从 Store 恢复系统代理模式选择
		if mw.appState != nil && mw.appState.ConfigService != nil {
			savedMode := mw.appState.ConfigService.GetSystemProxyMode()
			if savedMode != "" {
				mw.updateProxyModeButtonsState(savedMode)
			}
		}
	}
	
	// 恢复系统代理状态（仅在首次创建时，避免重复应用）
	// 注意：按钮状态已在创建按钮时恢复，这里只应用实际的系统代理设置
	if !mw.systemProxyRestored {
		if mw.appState != nil && mw.appState.ConfigService != nil {
			savedMode := mw.appState.ConfigService.GetSystemProxyMode()
			if savedMode != "" {
				// 应用系统代理设置（不保存到 Store，因为这是从 Store 恢复的）
				_ = mw.applySystemProxyModeWithoutSave(savedMode)
			}
		}
		mw.systemProxyRestored = true
	}

	// 顶部：当前连接状态（简洁文案，居中显示）
	statusHeader := container.NewCenter(container.NewHBox(
		statusIcon,
		NewSpacer(SpacingSmall),
		mw.proxyStatusLabel,
	))
	statusHeader = container.NewPadded(statusHeader)

	// 中部：巨大的主开关按钮（居中，更大的尺寸）
	mainControlArea := container.NewCenter(container.NewPadded(mw.mainToggleButton))

	// 下方：当前节点信息（可点击，跳转到节点选择页面）
	nodeInfoButton := widget.NewButton("", func() {
		mw.ShowNodePage()
	})
	nodeInfoButton.Importance = widget.LowImportance
	
	// 节点信息内容：仅保留一个图标和节点名称（不显示延迟）
	// 使用自定义布局确保：图标区域占10%，节点名称区域占90%
	iconWithSpacer := container.NewHBox(
		widget.NewIcon(theme.ComputerIcon()),
		NewSpacer(SpacingSmall),
	)
	
	// 节点名称区域：占90%宽度，确保占满
	nodeNameArea := container.NewWithoutLayout(mw.serverNameLabel)
	
	// 使用自定义布局精确控制：图标10%，节点名称90%
	nodeInfoContent := container.NewWithoutLayout(iconWithSpacer, nodeNameArea)
	nodeInfoContent.Layout = &nodeNameLayout{}
	
	// 节点信息区域：占满宽度，留一些边距
	nodeInfoArea := container.NewStack(
		nodeInfoButton,
		container.NewPadded(nodeInfoContent),
	)

	// 模式选择：使用图标和三个按钮，按钮组占90%宽度，Mac 简约风格
	// 图标区域：占10%宽度
	modeIcon := widget.NewIcon(theme.SettingsIcon())
	iconArea := container.NewHBox(
		modeIcon,
		NewSpacer(SpacingSmall),
	)
	
	// 按钮组区域：占90%宽度
	buttonGroup := container.NewWithoutLayout(
		mw.proxyModeButtons[0],
		mw.proxyModeButtons[1],
		mw.proxyModeButtons[2],
	)
	buttonGroup.Layout = &proxyModeButtonLayout{}
	
	// 使用自定义布局：图标10%，按钮组90%
	modeInfo := container.NewWithoutLayout(iconArea, buttonGroup)
	modeInfo.Layout = &modeButtonLayout{}
	modeInfo = container.NewPadded(modeInfo)

	// 节点和模式信息垂直排列，占满宽度（留一些边距）
	nodeAndMode := container.NewVBox(
		nodeInfoArea,
		modeInfo,
	)
	nodeAndMode = container.NewPadded(nodeAndMode)

	// 底部：实时流量占位（未来可替换为小曲线图）
	trafficPlaceholder := widget.NewLabel("实时流量图（预留）")
	trafficPlaceholder.Alignment = fyne.TextAlignCenter
	trafficArea := container.NewCenter(container.NewPadded(trafficPlaceholder))

	// 整体垂直排版
	content := container.NewVBox(
		statusHeader,
		NewSpacer(SpacingLarge),
		mainControlArea,
		NewSpacer(SpacingLarge),
		nodeAndMode,
		NewSpacer(SpacingMedium),
		trafficArea,
	)

	// 顶部标题栏：右侧仅保留设置入口
	headerButtons := container.NewHBox(
		layout.NewSpacer(),
		NewStyledButton("设置", theme.SettingsIcon(), func() {
			mw.ShowSettingsPage()
		}),
	)
	headerBar := container.NewPadded(headerButtons)

	return container.NewBorder(
		headerBar,
		NewSpacer(SpacingLarge), // 底部预留少量空白
		nil,
		nil,
		container.NewCenter(content),
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
	
	// 从 Store 读取窗口大小并应用（在SetContent之后，避免内容的最小尺寸要求导致窗口变大）
	defaultSize := fyne.NewSize(420, 520)
	windowSize := LoadWindowSize(mw.appState, defaultSize)
	mw.appState.Window.Resize(windowSize)
	// 保存当前窗口大小到 Store（确保保存的是设置后的尺寸）
	SaveWindowSize(mw.appState, windowSize)
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
		// 返回主界面时更新节点信息显示
		// 使用双向绑定，只需更新绑定数据，UI 会自动更新
		if mw.appState != nil {
			mw.appState.UpdateProxyStatus() // 更新绑定数据（serverNameLabel 会自动更新）
		}
		pageContent = mw.homePage
	case PageTypeNode:
		if mw.nodePage == nil {
			mw.nodePage = mw.nodePageInstance.Build()
		}
		// 刷新服务器列表并滚动到选中位置
		if mw.nodePageInstance != nil {
			mw.nodePageInstance.Refresh()
			// 延迟执行滚动，确保列表已渲染
			fyne.Do(func() {
				mw.nodePageInstance.scrollToSelected()
			})
		}
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

// onToggleProxy 主开关按钮回调：启动/停止代理
func (mw *MainWindow) onToggleProxy() {
	if mw.appState == nil {
		return
	}

	// 检查代理是否正在运行
	isRunning := false
	if mw.appState.XrayInstance != nil {
		isRunning = mw.appState.XrayInstance.IsRunning()
	}

	if isRunning {
		// 停止代理
		if mw.nodePageInstance != nil {
			mw.nodePageInstance.StopProxy()
		}
	} else {
		// 启动代理（使用当前选中的服务器）
		if mw.nodePageInstance != nil {
			mw.nodePageInstance.StartProxyForSelected()
		}
	}

	// 更新状态
	mw.refreshHomePageStatus()
}

// refreshHomePageStatus 刷新主界面状态显示
func (mw *MainWindow) refreshHomePageStatus() {
	if mw.appState != nil {
		mw.appState.UpdateProxyStatus()
	}
	// 注意：不再显示延迟，已从节点信息区域移除
	if mw.mainToggleButton != nil {
		mw.updateMainToggleButton()
	}
}

// updateStatusIcon 更新状态图标
func (mw *MainWindow) updateStatusIcon(icon *widget.Icon) {
	if icon == nil {
		return
	}
	
	isRunning := false
	if mw.appState != nil && mw.appState.XrayInstance != nil {
		isRunning = mw.appState.XrayInstance.IsRunning()
	}
	
	if isRunning {
		icon.SetResource(theme.ConfirmIcon())
	} else {
		icon.SetResource(theme.CancelIcon())
	}
}

// updateMainToggleButton 根据代理运行状态更新主开关按钮的文案和样式
func (mw *MainWindow) updateMainToggleButton() {
	if mw.mainToggleButton == nil {
		return
	}

	isRunning := false
	if mw.appState != nil && mw.appState.XrayInstance != nil {
		isRunning = mw.appState.XrayInstance.IsRunning()
	}

	if isRunning {
		mw.mainToggleButton.SetText("🟢 ON")
		mw.mainToggleButton.Importance = widget.HighImportance
	} else {
		mw.mainToggleButton.SetText("⚪ OFF")
		mw.mainToggleButton.Importance = widget.MediumImportance
	}
}

// updateDelayLabel 根据当前选中服务器更新延迟显示
func (mw *MainWindow) updateDelayLabel() {
	if mw.delayLabel == nil || mw.appState == nil {
		return
	}

	delayText := "-"
	if mw.appState.Store != nil && mw.appState.Store.Nodes != nil {
		selectedNode := mw.appState.Store.Nodes.GetSelected()
		if selectedNode != nil {
			if selectedNode.Delay > 0 {
				var colorIndicator string
				if selectedNode.Delay < 100 {
					colorIndicator = "🟢"
				} else if selectedNode.Delay <= 200 {
					colorIndicator = "🟡"
				} else {
					colorIndicator = "🔴"
				}
				delayText = fmt.Sprintf("%s %dms", colorIndicator, selectedNode.Delay)
			} else if selectedNode.Delay < 0 {
				delayText = "🔴 超时"
			} else {
				delayText = "⚪ N/A"
			}
		}
	}
	mw.delayLabel.SetText(delayText)
}

// onProxyModeButtonClicked 系统代理模式按钮点击处理
// 直接调用 systemproxy 方法设置系统代理，不启动代理
func (mw *MainWindow) onProxyModeButtonClicked(fullModeName string) {
	if mw.appState == nil {
		return
	}

	// 更新按钮选中状态
	mw.updateProxyModeButtonsState(fullModeName)

	// 直接调用 systemproxy 方法设置系统代理（不启动代理）
	proxyPort := 10080
	if mw.appState.XrayInstance != nil && mw.appState.XrayInstance.IsRunning() {
		if port := mw.appState.XrayInstance.GetPort(); port > 0 {
			proxyPort = port
		}
	}

	// 确保 SystemProxy 实例已创建
	if mw.systemProxy == nil {
		mw.systemProxy = systemproxy.NewSystemProxy("127.0.0.1", proxyPort)
	} else {
		mw.systemProxy.UpdateProxy("127.0.0.1", proxyPort)
	}

	var err error
	var logMessage string

	switch fullModeName {
	case SystemProxyModeClear:
		err = mw.systemProxy.ClearSystemProxy()
		terminalErr := mw.systemProxy.ClearTerminalProxy()
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
		_ = mw.systemProxy.ClearSystemProxy()
		_ = mw.systemProxy.ClearTerminalProxy()
		err = mw.systemProxy.SetSystemProxy()
		if err == nil {
			logMessage = fmt.Sprintf("已自动配置系统代理: 127.0.0.1:%d", proxyPort)
		} else {
			logMessage = fmt.Sprintf("自动配置系统代理失败: %v", err)
		}

	case SystemProxyModeTerminal:
		_ = mw.systemProxy.ClearSystemProxy()
		_ = mw.systemProxy.ClearTerminalProxy()
		err = mw.systemProxy.SetTerminalProxy()
		if err == nil {
			logMessage = fmt.Sprintf("已设置环境变量代理: socks5://127.0.0.1:%d (已写入shell配置文件)", proxyPort)
		} else {
			logMessage = fmt.Sprintf("设置环境变量代理失败: %v", err)
		}

	default:
		logMessage = fmt.Sprintf("未知的系统代理模式: %s", fullModeName)
		err = fmt.Errorf("未知的系统代理模式: %s", fullModeName)
	}

	// 输出日志
	if err == nil {
		mw.appState.AppendLog("INFO", "app", logMessage)
		if mw.appState.Logger != nil {
			mw.appState.Logger.InfoWithType(logging.LogTypeApp, "%s", logMessage)
		}
	} else {
		mw.appState.AppendLog("ERROR", "app", logMessage)
		if mw.appState.Logger != nil {
			mw.appState.Logger.Error("%s", logMessage)
		}
	}

	// 保存状态到 Store
	mw.saveSystemProxyState(fullModeName)
}

// updateProxyModeButtonsState 更新按钮选中状态
// 选中按钮使用 MediumImportance（适中的视觉区分，颜色已通过主题加深20%），未选中按钮使用 LowImportance（Mac 简约风格）
func (mw *MainWindow) updateProxyModeButtonsState(fullModeName string) {
	if mw.proxyModeButtons[0] == nil {
		return
	}

	// 重置所有按钮为未选中状态（LowImportance）
	for i := range mw.proxyModeButtons {
		mw.proxyModeButtons[i].Importance = widget.LowImportance
	}

	// 设置选中按钮为中等重要性（颜色已通过主题加深20%）
	switch fullModeName {
	case SystemProxyModeClear:
		mw.proxyModeButtons[0].Importance = widget.MediumImportance
	case SystemProxyModeAuto:
		mw.proxyModeButtons[1].Importance = widget.MediumImportance
	case SystemProxyModeTerminal:
		mw.proxyModeButtons[2].Importance = widget.MediumImportance
	}

	// 刷新按钮显示
	for i := range mw.proxyModeButtons {
		mw.proxyModeButtons[i].Refresh()
	}
}

// getFullModeName 将简短文本映射到完整的功能名称
func (mw *MainWindow) getFullModeName(shortText string) string {
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
func (mw *MainWindow) getShortModeName(fullName string) string {
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

// applySystemProxyMode 应用系统代理模式
// 参数：
//   - fullModeName: 完整模式名称（如"清除系统代理"、"自动配置系统代理"、"环境变量代理"）
func (mw *MainWindow) applySystemProxyMode(fullModeName string) error {
	if mw.appState == nil {
		return fmt.Errorf("appState 未初始化")
	}

	// 确保 ProxyService 已初始化或更新
	if mw.appState.ProxyService == nil {
		// 如果 XrayInstance 已创建，初始化 ProxyService
		if mw.appState.XrayInstance != nil {
			mw.appState.ProxyService = service.NewProxyService(mw.appState.XrayInstance)
		} else {
			// 即使 XrayInstance 未创建，也创建 ProxyService（使用 nil）
			mw.appState.ProxyService = service.NewProxyService(nil)
		}
	} else {
		// 如果 ProxyService 已存在，更新 XrayInstance 引用
		if mw.appState.XrayInstance != nil {
			mw.appState.ProxyService.UpdateXrayInstance(mw.appState.XrayInstance)
		}
	}

	// 将完整模式名称映射到简短模式名称（用于 ProxyService）
	var mode string
	switch fullModeName {
	case SystemProxyModeClear:
		mode = "clear"
	case SystemProxyModeAuto:
		mode = "auto"
	case SystemProxyModeTerminal:
		mode = "terminal"
	default:
		return fmt.Errorf("未知的系统代理模式: %s", fullModeName)
	}

	// 调用 ProxyService 应用系统代理模式
	result := mw.appState.ProxyService.ApplySystemProxyMode(mode)

	// 输出日志
	if result.Error == nil {
		mw.appState.AppendLog("INFO", "app", result.LogMessage)
		if mw.appState.Logger != nil {
			mw.appState.Logger.InfoWithType(logging.LogTypeApp, "%s", result.LogMessage)
		}
	} else {
		mw.appState.AppendLog("ERROR", "app", result.LogMessage)
		if mw.appState.Logger != nil {
			mw.appState.Logger.Error("%s", result.LogMessage)
		}
	}

	// 保存完整模式名称到 Store（供全局使用和恢复）
	mw.saveSystemProxyState(fullModeName)

	return result.Error
}

// updateSystemProxyPort 更新系统代理管理器的端口
func (mw *MainWindow) updateSystemProxyPort() {
	if mw.appState == nil {
		return
	}

	proxyPort := 10080
	if mw.appState.XrayInstance != nil && mw.appState.XrayInstance.IsRunning() {
		if port := mw.appState.XrayInstance.GetPort(); port > 0 {
			proxyPort = port
		}
	}

	mw.systemProxy = systemproxy.NewSystemProxy("127.0.0.1", proxyPort)
}

// saveSystemProxyState 保存系统代理状态到数据库
func (mw *MainWindow) saveSystemProxyState(mode string) {
	if mw.appState == nil || mw.appState.ConfigService == nil {
		return
	}
	if err := mw.appState.ConfigService.SetSystemProxyMode(mode); err != nil {
		if mw.appState.Logger != nil {
			mw.appState.Logger.Error("保存系统代理状态失败: %v", err)
		}
	}
}

// applySystemProxyModeWithoutSave 应用系统代理模式但不保存到 Store（用于恢复时避免重复保存）
// 直接调用 systemproxy 方法，不通过 ProxyService
func (mw *MainWindow) applySystemProxyModeWithoutSave(fullModeName string) error {
	if mw.appState == nil {
		return fmt.Errorf("appState 未初始化")
	}

	// 直接调用 systemproxy 方法设置系统代理（不启动代理）
	proxyPort := 10080
	if mw.appState.XrayInstance != nil && mw.appState.XrayInstance.IsRunning() {
		if port := mw.appState.XrayInstance.GetPort(); port > 0 {
			proxyPort = port
		}
	}

	// 确保 SystemProxy 实例已创建
	if mw.systemProxy == nil {
		mw.systemProxy = systemproxy.NewSystemProxy("127.0.0.1", proxyPort)
	} else {
		mw.systemProxy.UpdateProxy("127.0.0.1", proxyPort)
	}

	var err error
	var logMessage string

	switch fullModeName {
	case SystemProxyModeClear:
		err = mw.systemProxy.ClearSystemProxy()
		terminalErr := mw.systemProxy.ClearTerminalProxy()
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
		_ = mw.systemProxy.ClearSystemProxy()
		_ = mw.systemProxy.ClearTerminalProxy()
		err = mw.systemProxy.SetSystemProxy()
		if err == nil {
			logMessage = fmt.Sprintf("已自动配置系统代理: 127.0.0.1:%d", proxyPort)
		} else {
			logMessage = fmt.Sprintf("自动配置系统代理失败: %v", err)
		}

	case SystemProxyModeTerminal:
		_ = mw.systemProxy.ClearSystemProxy()
		_ = mw.systemProxy.ClearTerminalProxy()
		err = mw.systemProxy.SetTerminalProxy()
		if err == nil {
			logMessage = fmt.Sprintf("已设置环境变量代理: socks5://127.0.0.1:%d (已写入shell配置文件)", proxyPort)
		} else {
			logMessage = fmt.Sprintf("设置环境变量代理失败: %v", err)
		}

	default:
		logMessage = fmt.Sprintf("未知的系统代理模式: %s", fullModeName)
		err = fmt.Errorf("未知的系统代理模式: %s", fullModeName)
	}

	// 输出日志（恢复时使用 INFO 级别，成功时静默，失败时记录）
	if err != nil {
		mw.appState.AppendLog("WARN", "app", fmt.Sprintf("恢复系统代理设置失败: %s", logMessage))
		if mw.appState.Logger != nil {
			mw.appState.Logger.Error("%s", logMessage)
		}
	}
	// 成功时静默，避免干扰用户

	// 注意：不保存到 Store，因为这是从 Store 恢复的，避免重复保存

	return err
}

