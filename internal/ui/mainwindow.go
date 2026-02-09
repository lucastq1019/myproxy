package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"myproxy.com/p/internal/logging"
	"myproxy.com/p/internal/service"
	"myproxy.com/p/internal/store"
	"myproxy.com/p/internal/systemproxy"
)

// proxyModeButtonLayout 自定义布局，确保两个按钮平分宽度
type proxyModeButtonLayout struct{}

func (p *proxyModeButtonLayout) Layout(objects []fyne.CanvasObject, containerSize fyne.Size) {
	if len(objects) != 2 {
		return
	}

	// 两个按钮平分宽度，每个占 1/2
	// 使用较小的间距，Mac 简约风格
	spacing := float32(4)       // 按钮之间的间距
	totalSpacing := spacing * 1 // 一个间距
	availableWidth := containerSize.Width - totalSpacing
	buttonWidth := availableWidth / 2

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
	if len(objects) < 2 {
		return fyne.NewSize(0, 0)
	}

	// 最小宽度：两个按钮的最小宽度之和
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
	minWidth += 1 * 4 // 一个间距

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
	PageTypeHome         PageType = iota // 主界面
	PageTypeNode                         // 节点列表页面
	PageTypeSettings                     // 设置页面
	PageTypeSubscription                 // 订阅管理页面
)

// PageStack 路由栈结构，用于管理页面导航历史
type PageStack struct {
	stack    []PageType // 页面栈
	maxDepth int        // 最大深度限制（0 表示无限制）
}

const (
	// DefaultMaxStackDepth 默认最大栈深度（防止异常情况导致栈无限增长）
	DefaultMaxStackDepth = 50
)

// NewPageStack 创建新的路由栈
func NewPageStack() *PageStack {
	return &PageStack{
		stack:    make([]PageType, 0),
		maxDepth: DefaultMaxStackDepth,
	}
}

// Push 将页面压入栈中
// 如果栈已满（达到最大深度），会移除最旧的页面（FIFO）
func (ps *PageStack) Push(pageType PageType) {
	// 如果设置了最大深度限制，且栈已满，移除最旧的页面
	if ps.maxDepth > 0 && len(ps.stack) >= ps.maxDepth {
		ps.stack = ps.stack[1:]
	}
	ps.stack = append(ps.stack, pageType)
}

// Pop 从栈中弹出页面
// 返回值：页面类型和是否成功弹出（栈为空时返回 false）
func (ps *PageStack) Pop() (PageType, bool) {
	if len(ps.stack) == 0 {
		return PageTypeHome, false
	}
	lastIndex := len(ps.stack) - 1
	pageType := ps.stack[lastIndex]
	ps.stack = ps.stack[:lastIndex]
	return pageType, true
}

// Peek 查看栈顶页面但不弹出
// 返回值：页面类型和是否存在（栈为空时返回 false）
func (ps *PageStack) Peek() (PageType, bool) {
	if len(ps.stack) == 0 {
		return PageTypeHome, false
	}
	return ps.stack[len(ps.stack)-1], true
}

// Size 返回栈中页面的数量
func (ps *PageStack) Size() int {
	return len(ps.stack)
}

// Clear 清空路由栈
func (ps *PageStack) Clear() {
	ps.stack = ps.stack[:0]
}

// IsEmpty 检查栈是否为空
func (ps *PageStack) IsEmpty() bool {
	return len(ps.stack) == 0
}

// SetMaxDepth 设置最大深度限制（0 表示无限制）
func (ps *PageStack) SetMaxDepth(depth int) {
	ps.maxDepth = depth
	// 如果当前栈超过新限制，移除最旧的页面
	if depth > 0 && len(ps.stack) > depth {
		ps.stack = ps.stack[len(ps.stack)-depth:]
	}
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

// SystemProxyMode 系统代理模式类型
type SystemProxyMode int

const (
	// SystemProxyModeClear 清除系统代理
	SystemProxyModeClear SystemProxyMode = iota
	// SystemProxyModeAuto 自动配置系统代理
	SystemProxyModeAuto
	// SystemProxyModeTerminal 环境变量代理
	SystemProxyModeTerminal
)

// String 返回完整模式名称（用于存储和日志）
func (m SystemProxyMode) String() string {
	switch m {
	case SystemProxyModeClear:
		return "清除系统代理"
	case SystemProxyModeAuto:
		return "自动配置系统代理"
	case SystemProxyModeTerminal:
		return "环境变量代理"
	default:
		return ""
	}
}

// ShortString 返回简短模式名称（用于UI显示）
func (m SystemProxyMode) ShortString() string {
	switch m {
	case SystemProxyModeClear:
		return "清除"
	case SystemProxyModeAuto:
		return "系统"
	case SystemProxyModeTerminal:
		return "终端"
	default:
		return ""
	}
}

// ParseSystemProxyMode 从完整模式名称解析 SystemProxyMode
func ParseSystemProxyMode(fullModeName string) SystemProxyMode {
	switch fullModeName {
	case "清除系统代理":
		return SystemProxyModeClear
	case "自动配置系统代理":
		return SystemProxyModeAuto
	case "环境变量代理":
		return SystemProxyModeTerminal
	default:
		return SystemProxyModeClear // 默认返回清除模式
	}
}

// ParseSystemProxyModeFromShort 从简短模式名称解析 SystemProxyMode
func ParseSystemProxyModeFromShort(shortModeName string) SystemProxyMode {
	switch shortModeName {
	case "清除":
		return SystemProxyModeClear
	case "系统":
		return SystemProxyModeAuto
	case "终端":
		return SystemProxyModeTerminal
	default:
		return SystemProxyModeClear // 默认返回清除模式
	}
}

// MainWindow 管理主窗口的布局和各个面板组件。
// 它负责协调订阅管理、服务器列表、日志显示和状态信息四个主要区域的显示。
type MainWindow struct {
	appState    *AppState
	logsPanel   *LogsPanel
	pageStack   *PageStack // 路由栈，用于管理页面导航历史
	currentPage PageType   // 当前页面类型

	// 单窗口多页面：通过 SetContent() 在一个窗口内切换不同的 Container
	homePage fyne.CanvasObject // 主界面（极简一键开关）

	nodePage         fyne.CanvasObject // 节点列表页面
	nodePageInstance *NodePage         // 节点列表页面实例

	settingsPage         fyne.CanvasObject // 设置页面
	settingsPageInstance *SettingsPage     // 设置页面实例

	subscriptionPage         fyne.CanvasObject // 订阅管理页面
	subscriptionPageInstance *SubscriptionPage // 订阅管理页面实例

	homeLogoIcon *widget.Icon // 主页logo图标，用于主题变化时更新

	// 主界面状态UI组件（使用双向绑定）
	mainToggleButton *CircularButton          // 主开关按钮（连接/断开，圆形，替代了状态显示）
	serverNameLabel  *widget.Label            // 服务器名称标签（绑定到 ServerNameBinding）
	proxyModeButtons [2]*widget.Button        // 系统代理模式按钮组（清除、系统）
	systemProxy      *systemproxy.SystemProxy // 系统代理管理器
	trafficChart     *TrafficChart            // 实时流量图组件

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
		appState:    appState,
		pageStack:   NewPageStack(),
		currentPage: PageTypeHome,
	}

	// 布局配置由 Store 管理，无需在这里加载

	// 创建各个面板
	mw.logsPanel = NewLogsPanel(appState)

	// 创建系统代理管理器（默认使用 localhost:10808）
	mw.systemProxy = systemproxy.NewSystemProxy("127.0.0.1", 10808)

	// 设置日志回调函数到 AppState，以便其他组件可以记录日志
	appState.LogCallback = func(level, logType, message string) {
		if mw.logsPanel != nil {
			mw.logsPanel.AppendLog(level, logType, message)
		}
	}

	// 注意：系统代理状态的恢复将在 buildHomePage() 中完成
	// 因为需要先创建 proxyModeRadio 组件

	return mw
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
		mw.appState.UpdateProxyStatus() // 更新绑定数据（serverNameLabel 会自动更新）
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
	_ = mw.appState.Store.Layout.Save(config)
}

// Cleanup 清理资源（在窗口关闭时调用）
func (mw *MainWindow) Cleanup() {
	// 停止流量图更新
	if mw.trafficChart != nil {
		mw.trafficChart.Stop()
	}
}

// GetLayoutConfig 返回当前的布局配置。
// 返回：布局配置实例，如果未初始化则返回默认配置
func (mw *MainWindow) GetLayoutConfig() *LayoutConfig {
	if mw.appState != nil && mw.appState.Store != nil && mw.appState.Store.Layout != nil {
		return mw.appState.Store.Layout.Get()
	}
	return DefaultLayoutConfig()
}

// initPages 初始化单窗口的四个页面：home / node / settings / subscription
func (mw *MainWindow) initPages() {
	// 主界面（homePage）：极简状态 + 一键主开关
	mw.homePage = mw.buildHomePage()

	// 设置页面（settingsPage）：顶部返回 + 标题，下方预留设置内容
	mw.settingsPageInstance = NewSettingsPage(mw.appState)
	mw.settingsPage = mw.settingsPageInstance.Build()

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
	// 创建主开关按钮（圆形，带链接图标）
	if mw.mainToggleButton == nil {
		// 计算按钮尺寸（窗口大小的1/10）
		buttonSize := mw.calculateButtonSize()

		// 创建圆形按钮（使用连接/断开图标，根据状态变化）
		if mw.appState != nil && mw.appState.XrayInstance != nil && mw.appState.XrayInstance.IsRunning() {
			mw.mainToggleButton = NewCircularButton(theme.CancelIcon(), mw.onToggleProxy, buttonSize, mw.appState)
		} else {
			mw.mainToggleButton = NewCircularButton(theme.ConfirmIcon(), mw.onToggleProxy, buttonSize, mw.appState)
		}
		mw.updateMainToggleButton()
	}

	// 创建系统代理模式按钮组（两个按钮平分宽度）
	if mw.proxyModeButtons[0] == nil {
		// 创建两个按钮，使用不同的图标增强视觉识别
		mw.proxyModeButtons[0] = widget.NewButtonWithIcon(SystemProxyModeClear.ShortString(), theme.DeleteIcon(), func() {
			mw.onProxyModeButtonClicked(SystemProxyModeClear)
		})
		mw.proxyModeButtons[1] = widget.NewButtonWithIcon(SystemProxyModeAuto.ShortString(), theme.ComputerIcon(), func() {
			mw.onProxyModeButtonClicked(SystemProxyModeAuto)
		})

		// 设置按钮初始重要性（所有按钮初始为 LowImportance，选中状态由 updateProxyModeButtonsState 管理）
		for i := range mw.proxyModeButtons {
			mw.proxyModeButtons[i].Importance = widget.LowImportance
		}

		// 从 Store 恢复系统代理模式选择
		if mw.appState != nil && mw.appState.ConfigService != nil {
			savedModeStr := mw.appState.ConfigService.GetSystemProxyMode()
			if savedModeStr != "" {
				savedMode := ParseSystemProxyMode(savedModeStr)
				// 如果保存的是终端模式，转换为清除模式（因为终端模式已移除）
				if savedMode == SystemProxyModeTerminal {
					savedMode = SystemProxyModeClear
				}
				mw.updateProxyModeButtonsState(savedMode)
			}
		}
	}

	// 恢复系统代理状态（仅在首次创建时，避免重复应用）
	// 注意：按钮状态已在创建按钮时恢复，这里只应用实际的系统代理设置
	if !mw.systemProxyRestored {
		if mw.appState != nil && mw.appState.ConfigService != nil {
			savedModeStr := mw.appState.ConfigService.GetSystemProxyMode()
			if savedModeStr != "" {
				savedMode := ParseSystemProxyMode(savedModeStr)
				// 应用系统代理设置（不保存到 Store，因为这是从 Store 恢复的）
				_ = mw.applySystemProxyModeWithoutSave(savedMode)
			}
		}
		mw.systemProxyRestored = true
	}

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

	// 节点信息区域：占满宽度，留一些边距，添加分隔线提升视觉效果
	nodeInfoArea := container.NewStack(
		nodeInfoButton,
		container.NewPadded(container.NewBorder(
			widget.NewSeparator(),
			widget.NewSeparator(),
			nil,
			nil,
			nodeInfoContent,
		)),
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

	// 底部：实时流量图
	if mw.trafficChart == nil {
		mw.trafficChart = NewTrafficChart(mw.appState)
	}
	trafficArea := container.NewPadded(mw.trafficChart)

	// 整体垂直排版（减少顶部留白，整体往上移动）
	content := container.NewVBox(
		mainControlArea,
		nodeAndMode,
		NewSpacer(SpacingSmall), // 减少间距（从 Large 改为 Medium）
		trafficArea,
	)

	// 顶部标题栏：左侧logo，右侧设置入口
	logoResource := createHomeLogo(mw.appState)
	mw.homeLogoIcon = widget.NewIcon(logoResource)
	if mw.homeLogoIcon != nil {
		mw.homeLogoIcon.Resize(fyne.NewSize(32, 32))
	}

	headerButtons := container.NewHBox(
		mw.homeLogoIcon,
		layout.NewSpacer(),
		NewButtonWithIcon("设置", theme.SettingsIcon(), func() {
			mw.ShowSettingsPage()
		}),
	)
	headerBar := container.NewPadded(headerButtons)

	return container.NewBorder(
		headerBar,
		nil, // 底部预留少量空白
		nil,
		nil,
		container.NewCenter(content),
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
			mw.settingsPageInstance = NewSettingsPage(mw.appState)
			mw.settingsPage = mw.settingsPageInstance.Build()
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
		mw.stopProxy()
	} else {
		// 启动代理（使用当前选中的服务器）
		mw.startProxy()
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

// startProxy 启动代理（使用当前选中的节点）
// 使用 XrayControlService 来处理代理启动逻辑
func (mw *MainWindow) startProxy() {
	if mw.appState == nil {
		mw.logAndShowError("启动代理失败", fmt.Errorf("AppState 未初始化"))
		return
	}

	if mw.appState.XrayControlService == nil {
		mw.logAndShowError("启动代理失败", fmt.Errorf("XrayControlService 未初始化"))
		return
	}

	// 使用统一的日志文件路径（与应用日志使用同一个文件）
	unifiedLogPath := ""
	if mw.appState.Logger != nil {
		unifiedLogPath = mw.appState.Logger.GetLogFilePath()
	}

	// 调用 service 启动代理
	result := mw.appState.XrayControlService.StartProxy(mw.appState.XrayInstance, unifiedLogPath)

	if result.Error != nil {
		mw.logAndShowError("启动代理失败", result.Error)
		if mw.appState != nil {
			mw.appState.UpdateProxyStatus()
		}
		return
	}

	// 启动成功，更新 AppState 中的 XrayInstance
	mw.appState.XrayInstance = result.XrayInstance

	// 更新 ProxyService 的 xray 实例引用
	if mw.appState.ProxyService != nil {
		mw.appState.ProxyService.UpdateXrayInstance(result.XrayInstance)
	} else {
		// 延迟初始化 ProxyService
		mw.appState.ProxyService = service.NewProxyService(result.XrayInstance)
	}

	// 记录日志（统一日志记录）
	if mw.appState.Logger != nil && result.XrayInstance != nil {
		selectedNode := mw.appState.Store.Nodes.GetSelected()
		if selectedNode != nil {
			mw.appState.Logger.InfoWithType(logging.LogTypeProxy, "xray-core代理已启动: %s (端口: %d)", selectedNode.Name, result.XrayInstance.GetPort())
		}
	}

	// 更新状态绑定（使用双向绑定，UI 会自动更新）
	if mw.appState != nil {
		mw.appState.UpdateProxyStatus()
	}

	// 刷新节点页面（如果已创建）
	if mw.nodePageInstance != nil {
		mw.nodePageInstance.Refresh()
	}

	// 显示成功对话框
	if mw.appState.Window != nil && result.XrayInstance != nil {
		selectedNode := mw.appState.Store.Nodes.GetSelected()
		if selectedNode != nil {
			message := fmt.Sprintf("代理已启动\n节点: %s\n端口: %d", selectedNode.Name, result.XrayInstance.GetPort())
			dialog.ShowInformation("代理启动成功", message, mw.appState.Window)
		}
	}
}

// StopProxy 停止代理（公共方法，供外部调用）
// 调用内部的 stopProxy 方法来停止 Xray 实例
func (mw *MainWindow) StopProxy() {
	mw.stopProxy()
}

// stopProxy 停止代理
// 使用 XrayControlService 来处理代理停止逻辑
func (mw *MainWindow) stopProxy() {
	if mw.appState == nil {
		mw.logAndShowError("停止代理失败", fmt.Errorf("AppState 未初始化"))
		return
	}

	if mw.appState.XrayControlService == nil {
		mw.logAndShowError("停止代理失败", fmt.Errorf("XrayControlService 未初始化"))
		return
	}

	// 调用 service 停止代理
	result := mw.appState.XrayControlService.StopProxy(mw.appState.XrayInstance)

	if result.Error != nil {
		mw.logAndShowError("停止代理失败", result.Error)
		return
	}

	// 停止成功，销毁实例（生命周期 = 代理运行生命周期）
	mw.appState.XrayInstance = nil

	// 记录日志（统一日志记录）
	if mw.appState.Logger != nil {
		mw.appState.Logger.InfoWithType(logging.LogTypeProxy, "xray-core代理已停止")
	}

	// 更新状态绑定
	if mw.appState != nil {
		mw.appState.UpdateProxyStatus()
	}

	// 刷新节点页面（如果已创建）
	if mw.nodePageInstance != nil {
		mw.nodePageInstance.Refresh()
	}

	// 显示成功对话框
	if mw.appState.Window != nil {
		if result.LogMessage == "代理未运行" {
			dialog.ShowInformation("提示", "代理未运行", mw.appState.Window)
		} else {
			dialog.ShowInformation("代理停止成功", "代理已停止", mw.appState.Window)
		}
	}
}

// logAndShowError 记录日志并显示错误（统一错误处理）
func (mw *MainWindow) logAndShowError(message string, err error) {
	if mw.appState != nil && mw.appState.Logger != nil {
		mw.appState.Logger.Error("%s: %v", message, err)
	}
	if mw.appState != nil && mw.appState.Window != nil {
		errorMsg := fmt.Errorf("%s: %w", message, err)
		dialog.ShowError(errorMsg, mw.appState.Window)
	}
	if mw.appState != nil {
		mw.appState.AppendLog("ERROR", "app", fmt.Sprintf("%s: %v", message, err))
	}
}

// 注意：updateStatusIcon 已移除，因为圆形按钮已经替代了状态图标显示

// calculateButtonSize 计算按钮尺寸（窗口大小的1/6，扩大主按钮）
func (mw *MainWindow) calculateButtonSize() float32 {
	if mw.appState == nil || mw.appState.Window == nil {
		// 默认尺寸
		return 100
	}

	// 获取窗口尺寸
	windowSize := mw.appState.Window.Canvas().Size()
	if windowSize.Width == 0 && windowSize.Height == 0 {
		// 如果窗口尺寸未初始化，使用默认尺寸
		return 100
	}

	// 取窗口宽度和高度的较小值，然后除以6（从10改为6，扩大按钮）
	minDimension := windowSize.Width
	if windowSize.Height < windowSize.Width {
		minDimension = windowSize.Height
	}

	buttonSize := minDimension / 6

	// 设置最小和最大尺寸限制（提高最小和最大尺寸）
	if buttonSize < 80 {
		buttonSize = 80
	} else if buttonSize > 180 {
		buttonSize = 180
	}

	return buttonSize
}

// updateMainToggleButton 根据代理运行状态更新主开关按钮的样式
func (mw *MainWindow) updateMainToggleButton() {
	if mw.mainToggleButton == nil {
		return
	}

	isRunning := false
	if mw.appState != nil && mw.appState.XrayInstance != nil {
		isRunning = mw.appState.XrayInstance.IsRunning()
	}

	// 更新按钮图标：运行中使用 CancelIcon，未运行时使用 ConfirmIcon
	if isRunning {
		mw.mainToggleButton.SetIcon(theme.CancelIcon())
	} else {
		mw.mainToggleButton.SetIcon(theme.ConfirmIcon())
	}

	// 更新按钮尺寸（响应窗口大小变化）
	buttonSize := mw.calculateButtonSize()
	mw.mainToggleButton.SetSize(buttonSize)
}

// applySystemProxyModeCore 应用系统代理模式的核心逻辑（可复用）
// 参数：
//   - mode: 系统代理模式
//   - saveToStore: 是否保存到 Store
//
// 返回值：错误信息
func (mw *MainWindow) applySystemProxyModeCore(mode SystemProxyMode, saveToStore bool) error {
	if mw.appState == nil {
		return fmt.Errorf("appState 未初始化")
	}

	// 获取当前代理端口
	proxyPort := 10808
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

	switch mode {
	case SystemProxyModeClear:
		err = mw.systemProxy.ClearSystemProxy()
		// 根据设置页面的配置决定是否清除终端代理
		shouldClearTerminal := false
		if mw.appState != nil && mw.appState.ConfigService != nil {
			shouldClearTerminal = mw.appState.ConfigService.GetTerminalProxyEnabled()
		}
		if shouldClearTerminal {
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
		} else {
			if err == nil {
				logMessage = "已清除系统代理设置"
			} else {
				logMessage = fmt.Sprintf("清除系统代理失败: %v", err)
			}
		}

	case SystemProxyModeAuto:
		_ = mw.systemProxy.ClearSystemProxy()
		// 根据设置页面的配置决定是否设置终端代理
		shouldSetTerminal := false
		if mw.appState != nil && mw.appState.ConfigService != nil {
			shouldSetTerminal = mw.appState.ConfigService.GetTerminalProxyEnabled()
		}
		err = mw.systemProxy.SetSystemProxy()
		if err == nil {
			logMessage = fmt.Sprintf("已自动配置系统代理: 127.0.0.1:%d", proxyPort)
			if shouldSetTerminal {
				terminalErr := mw.systemProxy.SetTerminalProxy()
				if terminalErr == nil {
					logMessage += "；已设置环境变量代理"
				} else {
					logMessage += fmt.Sprintf("；设置环境变量代理失败: %v", terminalErr)
				}
			}
		} else {
			logMessage = fmt.Sprintf("自动配置系统代理失败: %v", err)
		}

	default:
		logMessage = fmt.Sprintf("未知的系统代理模式: %s", mode.String())
		err = fmt.Errorf("未知的系统代理模式: %s", mode.String())
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

	// 保存状态到 Store（如果需要）
	if saveToStore {
		mw.saveSystemProxyState(mode)
	}

	return err
}

// onProxyModeButtonClicked 系统代理模式按钮点击处理
// 直接调用 systemproxy 方法设置系统代理，不启动代理
func (mw *MainWindow) onProxyModeButtonClicked(mode SystemProxyMode) {
	if mw.appState == nil {
		return
	}

	// 使用统一的 SetSystemProxyMode 方法，确保托盘菜单也能同步更新
	_ = mw.SetSystemProxyMode(mode)
}

// SetSystemProxyMode 设置系统代理模式（公共方法，供托盘等外部调用）
// 参数：
//   - mode: 系统代理模式
func (mw *MainWindow) SetSystemProxyMode(mode SystemProxyMode) error {
	if mw.appState == nil {
		return fmt.Errorf("appState 未初始化")
	}

	// 更新按钮选中状态（如果按钮已创建）
	mw.updateProxyModeButtonsState(mode)

	// 应用系统代理模式（保存到 Store）
	err := mw.applySystemProxyModeCore(mode, true)

	// 刷新托盘菜单（如果存在）
	// 注意：TrayManager 不是 AppState 的字段，而是在 SetupTray 中创建的临时对象
	// 这里我们不直接引用 TrayManager，因为它的生命周期由 SetupTray 管理
	// 托盘菜单的刷新会在模式变化时自动处理

	return err
}

// GetCurrentSystemProxyMode 获取当前系统代理模式
// 返回值：当前模式，如果未设置则返回 SystemProxyModeClear
func (mw *MainWindow) GetCurrentSystemProxyMode() SystemProxyMode {
	if mw.appState == nil || mw.appState.ConfigService == nil {
		return SystemProxyModeClear
	}
	modeStr := mw.appState.ConfigService.GetSystemProxyMode()
	if modeStr == "" {
		return SystemProxyModeClear
	}
	return ParseSystemProxyMode(modeStr)
}

// updateProxyModeButtonsState 更新按钮选中状态
// 选中按钮使用 MediumImportance（适中的视觉区分，颜色已通过主题加深20%），未选中按钮使用 LowImportance（Mac 简约风格）
func (mw *MainWindow) updateProxyModeButtonsState(mode SystemProxyMode) {
	if mw.proxyModeButtons[0] == nil {
		return
	}

	// 重置所有按钮为未选中状态（LowImportance）
	for i := range mw.proxyModeButtons {
		mw.proxyModeButtons[i].Importance = widget.LowImportance
	}

	// 设置选中按钮为中等重要性（颜色已通过主题加深20%）
	switch mode {
	case SystemProxyModeClear:
		mw.proxyModeButtons[0].Importance = widget.MediumImportance
	case SystemProxyModeAuto:
		mw.proxyModeButtons[1].Importance = widget.MediumImportance
	}

	// 刷新按钮显示
	for i := range mw.proxyModeButtons {
		mw.proxyModeButtons[i].Refresh()
	}
}

// applySystemProxyMode 应用系统代理模式（通过 ProxyService，已废弃，保留用于兼容性）
// 参数：
//   - mode: 系统代理模式
func (mw *MainWindow) applySystemProxyMode(mode SystemProxyMode) error {
	// 直接使用核心方法
	return mw.applySystemProxyModeCore(mode, true)
}

// updateSystemProxyPort 更新系统代理管理器的端口
func (mw *MainWindow) updateSystemProxyPort() {
	if mw.appState == nil {
		return
	}

	proxyPort := 10808
	if mw.appState.XrayInstance != nil && mw.appState.XrayInstance.IsRunning() {
		if port := mw.appState.XrayInstance.GetPort(); port > 0 {
			proxyPort = port
		}
	}

	mw.systemProxy = systemproxy.NewSystemProxy("127.0.0.1", proxyPort)
}

// saveSystemProxyState 保存系统代理状态到数据库
func (mw *MainWindow) saveSystemProxyState(mode SystemProxyMode) {
	if mw.appState == nil || mw.appState.ConfigService == nil {
		return
	}
	// 保存完整模式名称字符串到 Store
	if err := mw.appState.ConfigService.SetSystemProxyMode(mode.String()); err != nil {
		if mw.appState.Logger != nil {
			mw.appState.Logger.Error("保存系统代理状态失败: %v", err)
		}
	}
}

// applySystemProxyModeWithoutSave 应用系统代理模式但不保存到 Store（用于恢复时避免重复保存）
// 直接调用 systemproxy 方法，不通过 ProxyService
func (mw *MainWindow) applySystemProxyModeWithoutSave(mode SystemProxyMode) error {
	// 使用核心方法，但不保存到 Store
	return mw.applySystemProxyModeCore(mode, false)
}
