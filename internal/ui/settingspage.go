package ui

import (
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// SettingsMenu 设置菜单项
type SettingsMenu int

const (
	SettingsMenuAppearance SettingsMenu = iota
	SettingsMenuDirectRoute
	SettingsMenuLog
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
		return "直连路由"
	case SettingsMenuLog:
		return "日志"
	case SettingsMenuAbout:
		return "关于"
	default:
		return ""
	}
}

// SettingsPage 管理应用设置的显示和操作。
// 左侧菜单栏：外观 | 直连路由 | 日志 | 关于；右侧为对应的内容区。
type SettingsPage struct {
	appState    *AppState
	content     fyne.CanvasObject
	menuButtons [4]*widget.Button
	contentCard *fyne.Container
	currentMenu SettingsMenu

	// 直连路由相关
	routesList    *widget.List
	routesData    []string
	routeAddEntry *widget.Entry
	routeUseProxy *widget.Check
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
		NewSpacer(SpacingLarge),
		titleLabel,
		layout.NewSpacer(),
	))

	// 左侧菜单
	sp.menuButtons[0] = widget.NewButton("外观", func() { sp.switchMenu(SettingsMenuAppearance) })
	sp.menuButtons[1] = widget.NewButton("直连路由", func() { sp.switchMenu(SettingsMenuDirectRoute) })
	sp.menuButtons[2] = widget.NewButton("日志", func() { sp.switchMenu(SettingsMenuLog) })
	sp.menuButtons[3] = widget.NewButton("关于", func() { sp.switchMenu(SettingsMenuAbout) })

	for i := range sp.menuButtons {
		sp.menuButtons[i].Importance = widget.LowImportance
	}

	menuBox := container.NewVBox(sp.menuButtons[0], sp.menuButtons[1], sp.menuButtons[2], sp.menuButtons[3])
	menuBox = container.NewPadded(menuBox)

	// 右侧内容区，使用 Scroll 包裹避免内容撑开窗口
	sp.contentCard = container.NewMax()
	sp.contentCard.Add(sp.buildAppearanceContent())
	contentArea := container.NewScroll(container.NewPadded(sp.contentCard))

	// 左右分栏
	mainContent := container.NewHSplit(menuBox, contentArea)
	mainContent.SetOffset(0.15)

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
	case SettingsMenuAbout:
		sp.contentCard.Add(sp.buildAboutContent())
	}
	sp.contentCard.Refresh()
	sp.updateMenuState()
}

// updateMenuState 更新菜单按钮选中样式。
func (sp *SettingsPage) updateMenuState() {
	for i := range sp.menuButtons {
		if SettingsMenu(i) == sp.currentMenu {
			sp.menuButtons[i].Importance = widget.MediumImportance
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
	if sp.appState != nil && sp.appState.ConfigService != nil {
		t := sp.appState.ConfigService.GetTheme()
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
		widget.NewLabelWithStyle("外观", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		container.NewVBox(
			widget.NewLabel("主题"),
			themeSelect,
			widget.NewLabel(ThemeDisplayDark+" / "+ThemeDisplayLight+" / "+ThemeDisplaySystem),
		),

		// 添加主题预览区域
		widget.NewSeparator(),
		buildThemePreview(),
	)
}

// buildDirectRouteContent 构建设置「直连路由」内容区。
func (sp *SettingsPage) buildDirectRouteContent() fyne.CanvasObject {
	sp.loadRoutes()

	sp.routeUseProxy = widget.NewCheck("直连列表中的地址也走代理", func(b bool) {
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

	return container.NewVBox(
		widget.NewLabelWithStyle("直连路由", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		sp.routeUseProxy,
		widget.NewLabel("勾选=走代理，不勾=走直连"),
		widget.NewLabel("路由列表"),
		listScroll,
		widget.NewLabel("添加新路由"),
		addArea,
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

// buildLogContent 构建设置「日志」内容区。
func (sp *SettingsPage) buildLogContent() fyne.CanvasObject {
	opts := []string{"debug", "info", "warn", "error"}
	logSelect := widget.NewSelect(opts, func(s string) {
		sp.onLogLevelChanged(s)
	})
	currentLevel := "info"
	if sp.appState != nil && sp.appState.Logger != nil {
		currentLevel = sp.appState.Logger.GetLogLevel()
	}
	logSelect.SetSelected(currentLevel)

	return container.NewVBox(
		widget.NewLabelWithStyle("日志", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		container.NewVBox(
			widget.NewLabel("级别"),
			logSelect,
			widget.NewLabel("debug / info / warn / error"),
		),
	)
}

// buildAboutContent 构建设置「关于」内容区。
func (sp *SettingsPage) buildAboutContent() fyne.CanvasObject {
	return container.NewVBox(
		widget.NewLabelWithStyle("关于", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		widget.NewLabel("myproxy  版本 1.0.0"),
		widget.NewLabel("轻量级代理管理工具，基于 Xray-core 与 Fyne"),
		widget.NewLabel("邮箱: lucastq1019@gmail.com"),
	)
}

// getSystemThemeVariant 获取系统主题变体
func getSystemThemeVariant() fyne.ThemeVariant {
	// 在 Fyne 中，默认的系统主题检测会通过 Settings().ThemeVariant() 提供
	// 这里我们直接返回，让 Fyne 处理系统主题检测
	return theme.VariantDark // 默认返回深色，实际会被系统主题覆盖
}

// onThemeChanged 主题变更回调。
func (sp *SettingsPage) onThemeChanged(selectedDisplay string) {
	if sp.appState == nil || sp.appState.App == nil {
		return
	}

	// 将显示文本转换为主题值
	newTheme := ThemeDark
	if selectedDisplay == ThemeDisplayLight {
		newTheme = ThemeLight
	} else if selectedDisplay == ThemeDisplaySystem {
		newTheme = ThemeSystem
	}

	// 保存主题配置
	if sp.appState.ConfigService != nil {
		_ = sp.appState.ConfigService.SetTheme(newTheme)
	}

	// 应用主题到 Fyne
	variant := theme.VariantDark
	if newTheme == ThemeLight {
		variant = theme.VariantLight
	} else if newTheme == ThemeSystem {
		// 跟随系统主题
		variant = sp.appState.App.Settings().ThemeVariant()
	}
	sp.appState.App.Settings().SetTheme(NewMonochromeTheme(variant))

	// 平滑主题切换动画
	// 使用 fyne.Do 确保在 UI 线程执行
	w := sp.appState.Window
	if w != nil {
		// 首先刷新当前设置页面，让主题选择器立即更新
		if sp.content != nil {
			sp.content.Refresh()
		}

		// 延迟一点时间，然后刷新整个窗口，确保主题变更完全生效
		go func() {
			// 短暂延迟，让主题设置有时间生效
			<-time.After(50 * time.Millisecond)

			fyne.Do(func() {
				content := w.Canvas().Content()
				if content != nil {
					// 重新设置内容以触发完整的主题刷新
					w.SetContent(content)
					// 显式刷新画布
					if c := w.Canvas(); c != nil {
						c.Refresh(content)
					}
				}
			})
		}()
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
