package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"myproxy.com/p/internal/model"
)

// SettingsMenu 设置菜单项
type SettingsMenu int

const (
	SettingsMenuAppearance SettingsMenu = iota
	SettingsMenuDirectRoute
	SettingsMenuLog
	SettingsMenuAccessRecord
	SettingsMenuAbout
)

// 主题相关常量
const (
	// ThemeDark 深色主题值
	ThemeDark = "dark"
	// ThemeLight 浅色主题值
	ThemeLight = "light"
	// ThemeSystem 跟随系统主题值
	ThemeSystem = "system"
	// ThemeDisplayDark 深色主题显示文本
	ThemeDisplayDark = "深色"
	// ThemeDisplayLight 浅色主题显示文本
	ThemeDisplayLight = "浅色"
	// ThemeDisplaySystem 跟随系统主题显示文本
	ThemeDisplaySystem = "跟随系统"
)

func (m SettingsMenu) String() string {
	switch m {
	case SettingsMenuAppearance:
		return "外观"
	case SettingsMenuDirectRoute:
		return "代理配置"
	case SettingsMenuLog:
		return "日志"
	case SettingsMenuAccessRecord:
		return "访问记录"
	case SettingsMenuAbout:
		return "关于"
	default:
		return ""
	}
}

// fixedMenuContentLayout 固定左侧菜单宽度、右侧内容占满剩余空间的布局；分隔不随窗口拖拽变化。
type fixedMenuContentLayout struct {
	menuWidth float32
}

func (f fixedMenuContentLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) != 2 {
		return fyne.NewSize(0, 0)
	}
	menuMin := objects[0].MinSize()
	contentMin := objects[1].MinSize()
	w := f.menuWidth
	if w < menuMin.Width {
		w = menuMin.Width
	}
	return fyne.NewSize(w+contentMin.Width, max(menuMin.Height, contentMin.Height))
}

func (f fixedMenuContentLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) != 2 {
		return
	}
	menuMin := objects[0].MinSize()
	w := f.menuWidth
	if w < menuMin.Width {
		w = menuMin.Width
	}
	contentW := size.Width - w
	if contentW < 0 {
		contentW = 0
	}
	objects[0].Resize(fyne.NewSize(w, size.Height))
	objects[0].Move(fyne.NewPos(0, 0))
	objects[1].Resize(fyne.NewSize(contentW, size.Height))
	objects[1].Move(fyne.NewPos(w, 0))
}

// SettingsPage 管理应用设置的显示和操作。
// 左侧菜单栏：外观 | 直连路由 | 日志 | 关于；右侧为对应的内容区。
type SettingsPage struct {
	appState    *AppState
	content     fyne.CanvasObject
	menuButtons [5]*widget.Button
	contentCard *fyne.Container
	currentMenu SettingsMenu

	// 直连路由相关
	routesList    *widget.List
	routesData    []string
	routeAddEntry *widget.Entry
	routeUseProxy *widget.Check

	// 日志：在设置页「日志」菜单中复用，用于查看日志
	logsPanel *LogsPanel

	// 访问记录相关
	accessRecordsList *widget.List
	accessRecordsData []model.AccessRecord
}

// NewSettingsPage 创建设置页面实例。
func NewSettingsPage(appState *AppState) *SettingsPage {
	sp := &SettingsPage{
		appState:    appState,
		currentMenu: SettingsMenuAppearance,
	}
	return sp
}

// Build 构建设置页面 UI。
func (sp *SettingsPage) Build() fyne.CanvasObject {
	backBtn := widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() {
		if sp.appState != nil && sp.appState.MainWindow != nil {
			sp.appState.MainWindow.Back()
		}
	})
	backBtn.Importance = widget.LowImportance

	titleLabel := NewTitleLabel("设置")
	headerBar := container.NewPadded(container.NewHBox(
		backBtn,
		layout.NewSpacer(),
		titleLabel,
		layout.NewSpacer(),
	))

	// 左侧菜单
	sp.menuButtons[0] = widget.NewButton("外观", func() { sp.switchMenu(SettingsMenuAppearance) })
	sp.menuButtons[1] = widget.NewButton("代理配置", func() { sp.switchMenu(SettingsMenuDirectRoute) })
	sp.menuButtons[2] = widget.NewButton("日志", func() { sp.switchMenu(SettingsMenuLog) })
	sp.menuButtons[3] = widget.NewButton("访问记录", func() { sp.switchMenu(SettingsMenuAccessRecord) })
	sp.menuButtons[4] = widget.NewButton("关于", func() { sp.switchMenu(SettingsMenuAbout) })

	for i := range sp.menuButtons {
		sp.menuButtons[i].Importance = widget.LowImportance
	}

	// 将logo和菜单按钮组合在一起
	menuContent := container.NewVBox(
		sp.menuButtons[0],
		sp.menuButtons[1],
		sp.menuButtons[2],
		sp.menuButtons[3],
		sp.menuButtons[4],
	)
	menuBox := container.NewPadded(menuContent)
	// 极简柔光：浅色模式下侧边栏背景 #F1F5F9，增加物理隔离感
	var sidebarBg fyne.CanvasObject
	if sp.appState != nil && sp.appState.App != nil {
		sidebarBg = canvas.NewRectangle(SidebarBackgroundColor(sp.appState.App))
	}
	leftColumn := menuBox
	if sidebarBg != nil {
		leftColumn = container.NewStack(sidebarBg, menuBox)
	}

	// 右侧内容区，使用 Scroll 包裹避免内容撑开窗口
	sp.contentCard = container.NewMax()
	sp.contentCard.Add(sp.buildAppearanceContent())
	contentArea := container.NewScroll(container.NewPadded(sp.contentCard))

	// 左右分栏：菜单固定宽度，完整展示菜单项；内容区占剩余空间（分隔不随窗口拖拽变化）
	mainContent := container.New(&fixedMenuContentLayout{menuWidth: 140}, leftColumn, contentArea)

	sp.content = container.NewBorder(
		headerBar,
		nil, nil, nil,
		mainContent,
	)

	sp.updateMenuState()
	return sp.content
}

// switchMenu 切换菜单并更新内容区。
func (sp *SettingsPage) switchMenu(menu SettingsMenu) {
	sp.currentMenu = menu
	sp.contentCard.RemoveAll()
	switch menu {
	case SettingsMenuAppearance:
		sp.contentCard.Add(sp.buildAppearanceContent())
	case SettingsMenuDirectRoute:
		sp.contentCard.Add(sp.buildDirectRouteContent())
	case SettingsMenuLog:
		sp.contentCard.Add(sp.buildLogContent())
	case SettingsMenuAccessRecord:
		sp.contentCard.Add(sp.buildAccessRecordContent())
	case SettingsMenuAbout:
		sp.contentCard.Add(sp.buildAboutContent())
	}
	sp.contentCard.Refresh()
	sp.updateMenuState()
}

// updateMenuState 更新菜单按钮选中样式。当前项使用 HighImportance（主色）便于区分。
func (sp *SettingsPage) updateMenuState() {
	for i := range sp.menuButtons {
		if SettingsMenu(i) == sp.currentMenu {
			sp.menuButtons[i].Importance = widget.HighImportance
		} else {
			sp.menuButtons[i].Importance = widget.LowImportance
		}
		sp.menuButtons[i].Refresh()
	}
}

// buildThemePreview 构建主题预览区域
func buildThemePreview() fyne.CanvasObject {
	// 创建预览卡片
	previewCard := container.NewVBox(
		// 预览标题
		widget.NewLabelWithStyle("主题预览", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		// 预览元素：按钮
		widget.NewLabel("按钮预览"),
		container.NewHBox(
			widget.NewButton("普通按钮", nil),
			widget.NewButtonWithIcon("图标按钮", theme.InfoIcon(), nil),
		),
		// 预览元素：输入框
		widget.NewLabel("输入框预览"),
		func() *widget.Entry {
			entry := widget.NewEntry()
			entry.SetPlaceHolder("请输入内容...")
			return entry
		}(),
		// 预览元素：复选框
		widget.NewLabel("复选框预览"),
		widget.NewCheck("选项 1", nil),
		// 预览元素：标签
		widget.NewLabel("文本预览：这是一段示例文本"),
	)

	// 添加边框和内边距
	previewCard = container.NewPadded(previewCard)

	// 创建一个带有最小大小的容器
	minSizeContainer := container.NewMax(previewCard)
	minSizeContainer.Resize(fyne.NewSize(0, 200))

	return minSizeContainer
}

// buildAppearanceContent 构建设置「外观」内容区。
func (sp *SettingsPage) buildAppearanceContent() fyne.CanvasObject {
	themeOptions := []string{ThemeDisplayDark, ThemeDisplayLight, ThemeDisplaySystem}
	themeSelect := widget.NewSelect(themeOptions, func(s string) {
		sp.onThemeChanged(s)
	})

	// 根据当前配置设置选中项
	currentThemeDisplay := ThemeDisplayDark
	if sp.appState != nil {
		t := sp.appState.GetTheme()
		switch t {
		case ThemeLight:
			currentThemeDisplay = ThemeDisplayLight
		case ThemeSystem:
			currentThemeDisplay = ThemeDisplaySystem
		default:
			currentThemeDisplay = ThemeDisplayDark
		}
	}
	themeSelect.SetSelected(currentThemeDisplay)

	return container.NewVBox(
		widget.NewLabel("主题"),
		themeSelect,
		// 添加主题预览区域
		widget.NewSeparator(),
		buildThemePreview(),
	)
}

// buildDirectRouteContent 构建设置「直连路由」内容区。
func (sp *SettingsPage) buildDirectRouteContent() fyne.CanvasObject {
	sp.loadRoutes()

	sp.routeUseProxy = widget.NewCheck("不走直连", func(b bool) {
		if sp.appState != nil && sp.appState.ConfigService != nil {
			_ = sp.appState.ConfigService.SetDirectRoutesUseProxy(b)
		}
	})
	if sp.appState != nil && sp.appState.ConfigService != nil {
		sp.routeUseProxy.SetChecked(sp.appState.ConfigService.GetDirectRoutesUseProxy())
	}

	sp.routesList = widget.NewList(
		func() int { return len(sp.routesData) },
		func() fyne.CanvasObject {
			textBtn := widget.NewButton("", nil)
			delBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), nil)
			return container.NewHBox(textBtn, layout.NewSpacer(), delBtn)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			row := obj.(*fyne.Container)
			textBtn := row.Objects[0].(*widget.Button)
			delBtn := row.Objects[2].(*widget.Button)

			if id < 0 || id >= len(sp.routesData) {
				return
			}
			route := sp.routesData[id]
			textBtn.SetText(route)
			textBtn.OnTapped = func() { sp.showEditRouteDialog(id) }
			delBtn.OnTapped = func() { sp.deleteRoute(id) }
		},
	)

	sp.routeAddEntry = widget.NewEntry()
	sp.routeAddEntry.SetPlaceHolder("domain:xxx 或 IP/CIDR")
	addBtn := widget.NewButtonWithIcon("添加", theme.ContentAddIcon(), sp.addRoute)
	addBtn.Importance = widget.LowImportance

	addArea := container.NewBorder(nil, nil, nil, addBtn, sp.routeAddEntry)

	listScroll := container.NewScroll(sp.routesList)
	listScroll.SetMinSize(fyne.NewSize(0, 120))

	// 重置按钮：添加默认路由（如果不存在）
	resetBtn := widget.NewButtonWithIcon("重置", theme.ViewRefreshIcon(), func() {
		sp.resetToDefaultRoutes()
	})
	resetBtn.Importance = widget.LowImportance

	// 终端代理配置选项
	terminalProxyCheck := widget.NewCheck("终端代理", func(b bool) {
		if sp.appState != nil && sp.appState.ConfigService != nil {
			_ = sp.appState.ConfigService.SetTerminalProxyEnabled(b)
		}
	})
	if sp.appState != nil && sp.appState.ConfigService != nil {
		terminalProxyCheck.SetChecked(sp.appState.ConfigService.GetTerminalProxyEnabled())
	}

	// 代理类型选择
	proxyTypeOptions := []string{"socks5", "https"}
	proxyTypeSelect := widget.NewSelect(proxyTypeOptions, func(s string) {
		if sp.appState != nil && sp.appState.ConfigService != nil {
			_ = sp.appState.ConfigService.SetProxyType(s)
		}
	})
	if sp.appState != nil && sp.appState.ConfigService != nil {
		proxyTypeSelect.SetSelected(sp.appState.ConfigService.GetProxyType())
	}
	proxyTypeLabel := widget.NewLabel("代理类型")

	// 代理配置区域：包含"终端代理"标题、"不走直连"、"重置"按钮
	proxyConfigArea := container.NewVBox(
		terminalProxyCheck,
		container.NewVBox(
			proxyTypeLabel,
			proxyTypeSelect,
		),
		widget.NewSeparator(),
		container.NewHBox(sp.routeUseProxy, resetBtn, layout.NewSpacer()),
	)

	routesLabel := widget.NewLabel("路由列表")

	// 使用 Border 布局：顶部固定代理配置区域，中间路由列表占满剩余空间，底部固定添加路由区域
	return container.NewBorder(
		container.NewVBox(proxyConfigArea, routesLabel), // 顶部：代理配置区域 + "路由列表"标签
		addArea, // 底部：添加路由输入框
		nil, nil,
		listScroll, // 中间：路由列表占满剩余空间
	)
}

// loadRoutes 从 ConfigService 加载直连路由到 routesData。
func (sp *SettingsPage) loadRoutes() {
	sp.routesData = nil
	if sp.appState != nil && sp.appState.ConfigService != nil {
		sp.routesData = sp.appState.ConfigService.GetDirectRoutes()
	}
	if sp.routesData == nil {
		sp.routesData = []string{}
	}
}

// resetToDefaultRoutes 重置直连路由：如果当前列表中没有默认路由则添加（使用map提高效率）
func (sp *SettingsPage) resetToDefaultRoutes() {
	if sp.appState == nil || sp.appState.ConfigService == nil {
		return
	}

	// 从 ConfigService 获取默认路由
	defaultRoutes := sp.appState.ConfigService.GetDefaultDirectRoutes()
	if len(defaultRoutes) == 0 {
		return
	}

	// 使用map提高查找效率
	existingRoutes := make(map[string]bool)
	for _, route := range sp.routesData {
		existingRoutes[route] = true
	}

	// 检查默认路由，如果不存在则添加
	added := false
	for _, defaultRoute := range defaultRoutes {
		if !existingRoutes[defaultRoute] {
			sp.routesData = append(sp.routesData, defaultRoute)
			added = true
		}
	}

	// 如果有新增，保存并刷新列表
	if added {
		sp.saveRoutes()
		if sp.routesList != nil {
			sp.routesList.Refresh()
		}
	}
}

// saveRoutes 将 routesData 保存到 ConfigService。
func (sp *SettingsPage) saveRoutes() {
	if sp.appState == nil || sp.appState.ConfigService == nil {
		return
	}
	_ = sp.appState.ConfigService.SetDirectRoutes(sp.routesData)
}

// addRoute 添加一条新路由。
func (sp *SettingsPage) addRoute() {
	text := strings.TrimSpace(sp.routeAddEntry.Text)
	if text == "" {
		return
	}
	routes := parseSingleRoute(text)
	if len(routes) == 0 {
		return
	}
	for _, r := range routes {
		// 去重
		found := false
		for _, existing := range sp.routesData {
			if existing == r {
				found = true
				break
			}
		}
		if !found {
			sp.routesData = append(sp.routesData, r)
		}
	}
	sp.routeAddEntry.SetText("")
	sp.saveRoutes()
	if sp.routesList != nil {
		sp.routesList.Refresh()
	}
}

// deleteRoute 删除指定索引的路由。
func (sp *SettingsPage) deleteRoute(id widget.ListItemID) {
	if id < 0 || id >= len(sp.routesData) {
		return
	}
	sp.routesData = append(sp.routesData[:id], sp.routesData[id+1:]...)
	sp.saveRoutes()
	if sp.routesList != nil {
		sp.routesList.Refresh()
	}
}

// showEditRouteDialog 弹出编辑路由对话框。
func (sp *SettingsPage) showEditRouteDialog(id widget.ListItemID) {
	if sp.appState == nil || sp.appState.Window == nil || id < 0 || id >= len(sp.routesData) {
		return
	}
	entry := widget.NewEntry()
	entry.SetText(sp.routesData[id])

	d := dialog.NewForm("编辑路由", "确定", "取消", []*widget.FormItem{
		{Text: "路由", Widget: entry},
	}, func(ok bool) {
		if !ok {
			return
		}
		text := strings.TrimSpace(entry.Text)
		if text == "" {
			return
		}
		routes := parseSingleRoute(text)
		if len(routes) > 0 {
			sp.routesData[id] = routes[0]
			sp.saveRoutes()
			if sp.routesList != nil {
				sp.routesList.Refresh()
			}
		}
	}, sp.appState.Window)
	d.Resize(fyne.NewSize(320, 0))
	d.Show()
}

// parseSingleRoute 解析单条路由输入，返回规范化后的列表。
func parseSingleRoute(input string) []string {
	// 复用 ConfigService 的解析逻辑：通过换行分割，空行忽略
	lines := strings.Split(input, "\n")
	var out []string
	for _, line := range lines {
		s := strings.TrimSpace(line)
		if s == "" {
			continue
		}
		if strings.HasPrefix(s, "domain:") || strings.HasPrefix(s, "geosite:") ||
			strings.HasPrefix(s, "regexp:") || strings.HasPrefix(s, "full:") {
			out = append(out, s)
		} else if strings.Contains(s, ".") && !isLikelyIPOrCIDR(s) {
			out = append(out, "domain:"+s)
		} else {
			out = append(out, s)
		}
	}
	return out
}

func isLikelyIPOrCIDR(s string) bool {
	if strings.Contains(s, "/") {
		return true
	}
	for _, r := range s {
		if (r >= '0' && r <= '9') || r == '.' {
			continue
		}
		return false
	}
	return true
}

// buildLogContent 构建设置「日志」内容区，嵌入完整日志面板用于查看日志。
func (sp *SettingsPage) buildLogContent() fyne.CanvasObject {
	if sp.appState != nil && sp.appState.LogsPanel != nil {
		return sp.appState.LogsPanel.Build()
	}
	if sp.logsPanel == nil {
		sp.logsPanel = NewLogsPanel(sp.appState)
	}
	return sp.logsPanel.Build()
}

// buildAccessRecordContent 构建设置「访问记录」内容区，展示访问的网站及累计访问次数。
func (sp *SettingsPage) buildAccessRecordContent() fyne.CanvasObject {
	sp.loadAccessRecords()

	sp.accessRecordsList = widget.NewList(
		func() int { return len(sp.accessRecordsData) },
		func() fyne.CanvasObject {
			addrLabel := widget.NewLabel("")
			addrLabel.Wrapping = fyne.TextWrapWord // 宽度过宽时自动换行
			countLabel := widget.NewLabel("")
			return container.NewVBox(
				addrLabel,
				container.NewHBox(layout.NewSpacer(), countLabel),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(sp.accessRecordsData) {
				return
			}
			r := sp.accessRecordsData[id]
			displayAddr := r.Address
			if displayAddr == "" {
				displayAddr = r.Domain
			}
			countText := fmt.Sprintf("访问 %d 次", r.AccessCount)
			labels := collectLabelsFromObject(obj)
			if len(labels) >= 2 {
				labels[0].SetText(displayAddr)
				labels[1].SetText(countText)
			}
		},
	)

	clearBtn := widget.NewButtonWithIcon("清空记录", theme.DeleteIcon(), func() {
		if sp.appState == nil || sp.appState.Window == nil {
			return
		}
		dialog.ShowConfirm("清空访问记录", "确定要清空所有访问记录吗？此操作不可恢复。", func(ok bool) {
			if !ok {
				return
			}
			if sp.appState != nil && sp.appState.Store != nil && sp.appState.Store.AccessRecords != nil {
				_ = sp.appState.Store.AccessRecords.ClearAll()
				_ = sp.appState.Store.AccessRecords.Load()
				sp.loadAccessRecords()
				if sp.accessRecordsList != nil {
					sp.accessRecordsList.Refresh()
				}
			}
		}, sp.appState.Window)
	})
	clearBtn.Importance = widget.LowImportance

	refreshBtn := widget.NewButtonWithIcon("刷新", theme.ViewRefreshIcon(), func() {
		sp.loadAccessRecords()
		if sp.accessRecordsList != nil {
			sp.accessRecordsList.Refresh()
		}
	})
	refreshBtn.Importance = widget.LowImportance

	topBar := container.NewHBox(
		widget.NewLabel("访问的地址（host:port，按最近访问时间排序）"),
		layout.NewSpacer(),
		refreshBtn,
		clearBtn,
	)

	listScroll := container.NewScroll(sp.accessRecordsList)
	listScroll.SetMinSize(fyne.NewSize(0, 200))

	return container.NewBorder(
		container.NewVBox(topBar, NewSeparator()),
		nil, nil, nil,
		listScroll,
	)
}

// loadAccessRecords 从 Store 加载访问记录。
func (sp *SettingsPage) loadAccessRecords() {
	sp.accessRecordsData = nil
	if sp.appState != nil && sp.appState.Store != nil && sp.appState.Store.AccessRecords != nil {
		sp.accessRecordsData = sp.appState.Store.AccessRecords.GetAll()
	}
	if sp.accessRecordsData == nil {
		sp.accessRecordsData = []model.AccessRecord{}
	}
}

// collectLabelsFromObject 递归收集 CanvasObject 树中的 *widget.Label，保持遍历顺序。
func collectLabelsFromObject(obj fyne.CanvasObject) []*widget.Label {
	var labels []*widget.Label
	if c, ok := obj.(*fyne.Container); ok {
		for _, o := range c.Objects {
			if l, ok := o.(*widget.Label); ok {
				labels = append(labels, l)
			} else {
				labels = append(labels, collectLabelsFromObject(o)...)
			}
		}
	}
	return labels
}

// buildAboutContent 构建设置「关于」内容区。
func (sp *SettingsPage) buildAboutContent() fyne.CanvasObject {
	titleLabel := widget.NewLabelWithStyle("关于", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	versionLabel := widget.NewLabel("myproxy  版本 1.0.0")
	versionLabel.Wrapping = fyne.TextWrapWord // 启用自动换行，适配窄屏显示

	descLabel := widget.NewLabel("轻量级代理管理工具，基于 Xray-core 与 Fyne")
	descLabel.Wrapping = fyne.TextWrapWord // 启用自动换行，适配窄屏显示

	emailLabel := widget.NewLabel("邮箱: lucastq1019@gmail.com")
	emailLabel.Wrapping = fyne.TextWrapWord // 启用自动换行，适配窄屏显示

	return container.NewVBox(
		titleLabel,
		widget.NewSeparator(),
		versionLabel,
		descLabel,
		emailLabel,
	)
}

// onThemeChanged 主题变更回调。
func (sp *SettingsPage) onThemeChanged(selectedDisplay string) {
	if sp.appState == nil || sp.appState.App == nil {
		return
	}

	// 将显示文本转换为主题值
	newTheme := ThemeDark
	switch selectedDisplay {
	case ThemeDisplayLight:
		newTheme = ThemeLight
	case ThemeDisplaySystem:
		newTheme = ThemeSystem
	}

	// 保存并应用主题配置
	if sp.appState != nil {
		_ = sp.appState.SetTheme(newTheme)
	}

	// 重建当前页面使主题色生效（设置页侧栏/背景等会重新取色）
	if sp.appState != nil && sp.appState.MainWindow != nil {
		sp.appState.MainWindow.RebuildCurrentPageForTheme()
	}
}

// onLogLevelChanged 日志级别变更回调。
func (sp *SettingsPage) onLogLevelChanged(level string) {
	if sp.appState == nil {
		return
	}
	if sp.appState.Logger != nil {
		sp.appState.Logger.SetLogLevel(level)
	}
	if sp.appState.ConfigService != nil {
		_ = sp.appState.ConfigService.Set("logLevel", level)
	}
}
