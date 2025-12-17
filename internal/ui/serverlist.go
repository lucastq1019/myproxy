package ui

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"myproxy.com/p/internal/config"
	"myproxy.com/p/internal/database"
	"myproxy.com/p/internal/logging"
	"myproxy.com/p/internal/xray"
)

// ServerListPanel 管理服务器列表的显示和操作。
// 它支持服务器选择、延迟测试、代理启动/停止等功能，并提供右键菜单操作。
type ServerListPanel struct {
	appState           *AppState
	serverList         *widget.List
	subscriptionSelect *widget.Select // 订阅选择下拉菜单
	onServerSelect     func(server config.Server)
	statusPanel        *StatusPanel // 状态面板引用（用于刷新和一键操作）

	// 搜索与过滤相关
	searchEntry *widget.Entry // 节点搜索输入框
	searchText  string        // 当前搜索关键字（小写）
}

// NewServerListPanel 创建并初始化服务器列表面板。
// 该方法会创建服务器列表组件并设置选中事件处理。
// 参数：
//   - appState: 应用状态实例
//
// 返回：初始化后的服务器列表面板实例
func NewServerListPanel(appState *AppState) *ServerListPanel {
	slp := &ServerListPanel{
		appState: appState,
	}

	// 服务器列表
	slp.serverList = widget.NewList(
		slp.getServerCount,
		slp.createServerItem,
		slp.updateServerItem,
	)

	// 设置选中事件
	slp.serverList.OnSelected = slp.onSelected

	return slp
}

// SetOnServerSelect 设置服务器选中时的回调函数。
// 参数：
//   - callback: 当用户选中服务器时调用的回调函数
func (slp *ServerListPanel) SetOnServerSelect(callback func(server config.Server)) {
	slp.onServerSelect = callback
}

// SetStatusPanel 设置状态面板的引用，以便在服务器操作后更新状态显示。
// 参数：
//   - statusPanel: 状态面板实例
func (slp *ServerListPanel) SetStatusPanel(statusPanel *StatusPanel) {
	slp.statusPanel = statusPanel
	// 将一键操作主开关与现有启动/停止逻辑绑定
	if slp.statusPanel != nil {
		slp.statusPanel.SetToggleHandler(func() {
			// 如果当前已有代理在运行，则走“停止”逻辑；否则启动当前选中服务器
			if slp.appState != nil && slp.appState.XrayInstance != nil && slp.appState.XrayInstance.IsRunning() {
				slp.StopProxy()
			} else {
				slp.StartProxyForSelected()
			}
		})
	}
}

// Build 构建并返回服务器列表面板的 UI 组件。
// 返回：包含操作按钮和服务器列表的容器组件
func (slp *ServerListPanel) Build() fyne.CanvasObject {
	// 操作按钮 - 添加图标
	testAllBtn := NewStyledButton("🔃 一键测速", theme.ViewRefreshIcon(), slp.onTestAll)
	startProxyBtn := NewStyledButton("启动代理", theme.ConfirmIcon(), slp.onStartProxyFromSelected)
	stopProxyBtn := NewStyledButton("停止代理", theme.CancelIcon(), slp.onStopProxy)

	// 全局搜索栏：支持按名称、地址、协议实时搜索
	slp.searchEntry = widget.NewEntry()
	slp.searchEntry.SetPlaceHolder("🔍 搜索节点（名称 / 地址 / 协议）")
	slp.searchEntry.OnChanged = func(value string) {
		// 记录小写关键字，便于不区分大小写匹配
		slp.searchText = strings.ToLower(strings.TrimSpace(value))
		slp.Refresh()
	}

	// 订阅选择下拉菜单 - 使用样式化的下拉框
	slp.subscriptionSelect = NewStyledSelect([]string{"加载中..."}, nil)
	slp.updateSubscriptionSelect(slp.subscriptionSelect)

	// 服务器列表标题（使用标题样式）
	titleLabel := NewTitleLabel("节点选择")

	// 订阅标签（使用副标题样式）
	subscriptionLabel := NewSubtitleLabel("订阅：")

	// 服务器列表标题和按钮 - 优化布局和间距，贴近 UI 草图：
	// 第一行：搜索栏 + 一键测速（核心高频操作）
	// 第二行：订阅筛选 + 启动/停止代理按钮
	headerArea := container.NewVBox(
		// 第一行：搜索 + 一键测速
		container.NewPadded(container.NewHBox(
			slp.searchEntry,
			NewSpacer(SpacingLarge),
			testAllBtn,
		)),
		// 第二行：标题 + 订阅筛选 + 启停代理
		container.NewPadded(container.NewHBox(
			titleLabel,
			NewSpacer(SpacingLarge),
			subscriptionLabel,
			slp.subscriptionSelect,
			layout.NewSpacer(),
			startProxyBtn,
			NewSpacer(SpacingSmall),
			stopProxyBtn,
		)),
	)

	// 创建列标题行，与列表项对齐
	columnHeaders := slp.createColumnHeaders()

	// 分组标题：收藏与全部节点（当前仅展示分组标题，收藏功能可在未来扩展）
	favoritesHeader := NewSubtitleLabel("⭐ 我的收藏 (Favorites)")
	allNodesHeader := NewSubtitleLabel("🌍 所有节点 (All Nodes)")

	// 服务器列表滚动区域（不再展示右侧详情）
	serverScroll := container.NewScroll(slp.serverList)

	// 列表上方插入分组标题（目前所有节点都显示在“所有节点”下方）
	listWithGroups := container.NewVBox(
		// TODO: 未来在这里插入真正的“收藏”节点列表
		favoritesHeader,
		NewSeparator(),
		allNodesHeader,
		NewSeparator(),
		columnHeaders,
		NewSeparator(),
		serverScroll,
	)

	// 返回包含标题和列表的容器
	return container.NewBorder(
		headerArea,
		nil,
		nil,
		nil,
		listWithGroups,
	)
}

// createColumnHeaders 创建列标题行，与列表项对齐
func (slp *ServerListPanel) createColumnHeaders() fyne.CanvasObject {
	// 创建列标题标签：地区 / 节点名称 / 端口 / 延迟
	regionHeader := NewSubtitleLabel("地区")
	regionHeader.Alignment = fyne.TextAlignCenter

	nameHeader := NewSubtitleLabel("节点名称")
	nameHeader.Alignment = fyne.TextAlignLeading

	portHeader := NewSubtitleLabel("端口")
	portHeader.Alignment = fyne.TextAlignCenter

	delayHeader := NewSubtitleLabel("延迟")
	delayHeader.Alignment = fyne.TextAlignCenter

	// 创建图标占位（与列表项对齐）
	iconPlaceholder := widget.NewIcon(theme.ComputerIcon())

	// 地区列容器
	regionContainer := container.NewGridWrap(
		fyne.NewSize(80, 28),
		container.NewPadded(container.NewStack(regionHeader)),
	)

	// 名称列容器（包含图标）
	nameContainer := container.NewGridWrap(
		fyne.NewSize(220, 28),
		container.NewHBox(
			iconPlaceholder,
			NewSpacer(SpacingSmall),
			container.NewStack(nameHeader),
		),
	)

	// 端口列容器
	portContainer := container.NewGridWrap(
		fyne.NewSize(80, 28),
		container.NewPadded(container.NewStack(portHeader)),
	)

	// 延迟列容器
	delayContainer := container.NewGridWrap(
		fyne.NewSize(90, 28),
		container.NewPadded(container.NewStack(delayHeader)),
	)

	// 使用网格布局组织各列容器，与列表项对齐
	gridContainer := container.NewGridWithColumns(4,
		regionContainer,
		nameContainer,
		portContainer,
		delayContainer,
	)

	// 添加内边距，与列表项保持一致
	headerContainer := container.NewPadded(gridContainer)

	return headerContainer
}

// updateSubscriptionSelect 更新订阅选择下拉菜单
func (slp *ServerListPanel) updateSubscriptionSelect(selectWidget *widget.Select) {
	// 获取所有订阅
	subscriptions, err := database.GetAllSubscriptions()
	if err != nil {
		selectWidget.Options = []string{"全部"}
		selectWidget.Refresh()
		return
	}

	// 创建选项列表，第一个选项为"全部"
	options := []string{"全部"}
	optionToID := map[string]int64{"全部": 0}

	// 添加所有订阅
	for _, sub := range subscriptions {
		option := sub.Label
		options = append(options, option)
		optionToID[option] = sub.ID
	}

	// 设置选项
	selectWidget.Options = options

	// 设置当前选中项
	currentSubscriptionID := slp.appState.ServerManager.GetSelectedSubscriptionID()
	if currentSubscriptionID == 0 {
		selectWidget.SetSelected("全部")
	} else {
		for option, id := range optionToID {
			if id == currentSubscriptionID {
				selectWidget.SetSelected(option)
				break
			}
		}
	}

	// 设置选择事件处理函数
	selectWidget.OnChanged = func(selected string) {
		// 获取选中的订阅ID
		subscriptionID := optionToID[selected]

		// 设置选中的订阅
		slp.appState.ServerManager.SetSelectedSubscriptionID(subscriptionID)

		// 刷新服务器列表
		slp.Refresh()

		// 更新状态面板
		if slp.statusPanel != nil {
			slp.statusPanel.Refresh()
		}
	}

	selectWidget.Refresh()
}

// Refresh 刷新服务器列表的显示，使 UI 反映最新的服务器数据。
func (slp *ServerListPanel) Refresh() {
	fyne.Do(func() {
		if slp.serverList != nil {
			slp.serverList.Refresh()
		}
	})
}

// getServerCount 获取服务器数量
func (slp *ServerListPanel) getServerCount() int {
	if slp.appState == nil || slp.appState.ServerManager == nil {
		return 0
	}
	return len(slp.getFilteredServers())
}

// getFilteredServers 根据当前搜索关键字返回过滤后的服务器列表。
// 支持按名称、地址、协议类型进行不区分大小写的匹配。
func (slp *ServerListPanel) getFilteredServers() []config.Server {
	if slp.appState == nil || slp.appState.ServerManager == nil {
		return []config.Server{}
	}

	servers := slp.appState.ServerManager.ListServers()
	// 如果没有搜索关键字，直接返回完整列表
	if slp.searchText == "" {
		return servers
	}

	filtered := make([]config.Server, 0, len(servers))
	for _, s := range servers {
		name := strings.ToLower(s.Name)
		addr := strings.ToLower(s.Addr)
		protocol := strings.ToLower(s.ProtocolType)

		if strings.Contains(name, slp.searchText) ||
			strings.Contains(addr, slp.searchText) ||
			strings.Contains(protocol, slp.searchText) {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

// createServerItem 创建服务器列表项
func (slp *ServerListPanel) createServerItem() fyne.CanvasObject {
	return NewServerListItem()
}

// updateServerItem 更新服务器列表项
func (slp *ServerListPanel) updateServerItem(id widget.ListItemID, obj fyne.CanvasObject) {
	servers := slp.getFilteredServers()
	if id < 0 || id >= len(servers) {
		return
	}

	srv := servers[id]
	item := obj.(*ServerListItem)

	// 设置面板引用和ID
	item.panel = slp
	item.id = id
	item.isEven = (id % 2) == 0 // 设置是否为偶数行
	item.isSelected = srv.Selected // 设置是否选中

	// 使用新的Update方法更新多列信息
	item.Update(srv)
}

// onSelected 服务器选中事件
func (slp *ServerListPanel) onSelected(id widget.ListItemID) {
	servers := slp.getFilteredServers()
	if id < 0 || id >= len(servers) {
		return
	}

	srv := servers[id]
	slp.appState.SelectedServerID = srv.ID

	// 更新状态绑定（使用双向绑定，UI 会自动更新）
	if slp.appState != nil {
		slp.appState.UpdateProxyStatus()
	}

	// 调用回调
	if slp.onServerSelect != nil {
		slp.onServerSelect(srv)
	}
}

// onRightClick 右键菜单
func (slp *ServerListPanel) onRightClick(id widget.ListItemID, ev *fyne.PointEvent) {
	servers := slp.getFilteredServers()
	if id < 0 || id >= len(servers) {
		return
	}

	srv := servers[id]
	slp.appState.SelectedServerID = srv.ID

	// 创建右键菜单
	menu := fyne.NewMenu("",
		fyne.NewMenuItem("测速", func() {
			slp.onTestSpeed(id)
		}),
		fyne.NewMenuItem("启动代理", func() {
			slp.onStartProxy(id)
		}),
		fyne.NewMenuItem("停止代理", func() {
			slp.onStopProxy()
		}),
	)

	// 显示菜单
	popup := widget.NewPopUpMenu(menu, slp.appState.Window.Canvas())
	popup.ShowAtPosition(ev.AbsolutePosition)
}

// onTestSpeed 测速
func (slp *ServerListPanel) onTestSpeed(id widget.ListItemID) {
	servers := slp.getFilteredServers()
	if id < 0 || id >= len(servers) {
		return
	}

	srv := servers[id]

	// 在goroutine中执行测速
	go func() {
		// 记录开始测速日志
		if slp.appState != nil {
			slp.appState.AppendLog("INFO", "ping", fmt.Sprintf("开始测试服务器延迟: %s (%s:%d)", srv.Name, srv.Addr, srv.Port))
		}

		delay, err := slp.appState.PingManager.TestServerDelay(srv)
		if err != nil {
			// 记录失败日志
			if slp.appState != nil {
				slp.appState.AppendLog("ERROR", "ping", fmt.Sprintf("服务器 %s 测速失败: %v", srv.Name, err))
			}
			fyne.Do(func() {
				slp.appState.Window.SetTitle(fmt.Sprintf("测速失败: %v", err))
			})
			return
		}

		// 更新服务器延迟
		slp.appState.ServerManager.UpdateServerDelay(srv.ID, delay)

		// 记录成功日志
		if slp.appState != nil {
			slp.appState.AppendLog("INFO", "ping", fmt.Sprintf("服务器 %s 测速完成: %d ms", srv.Name, delay))
		}

		// 更新UI（需要在主线程中执行）
		fyne.Do(func() {
			slp.Refresh()
			slp.onSelected(id) // 刷新详情
			// 更新状态绑定（使用双向绑定，UI 会自动更新）
			if slp.appState != nil {
				slp.appState.UpdateProxyStatus()
			}
			slp.appState.Window.SetTitle(fmt.Sprintf("测速完成: %d ms", delay))
		})
	}()
}

// onStartProxyFromSelected 从当前选中的服务器启动代理
func (slp *ServerListPanel) onStartProxyFromSelected() {
	if slp.appState.SelectedServerID == "" {
		slp.appState.Window.SetTitle("请先选择一个服务器")
		return
	}

	servers := slp.appState.ServerManager.ListServers()
	var srv *config.Server
	for i := range servers {
		if servers[i].ID == slp.appState.SelectedServerID {
			srv = &servers[i]
			break
		}
	}

	if srv == nil {
		slp.appState.Window.SetTitle("选中的服务器不存在")
		return
	}

	// 如果已有代理在运行，先停止
	if slp.appState.XrayInstance != nil {
		slp.appState.XrayInstance.Stop()
		slp.appState.XrayInstance = nil
	}

	// 把当前的设置为选中
	slp.appState.ServerManager.SelectServer(srv.ID)
	slp.appState.SelectedServerID = srv.ID

	// 启动代理
	slp.startProxyWithServer(srv)
}

// onStartProxy 启动代理（右键菜单使用）
func (slp *ServerListPanel) onStartProxy(id widget.ListItemID) {
	servers := slp.appState.ServerManager.ListServers()
	if id < 0 || id >= len(servers) {
		return
	}

	srv := servers[id]
	slp.appState.ServerManager.SelectServer(srv.ID)
	slp.appState.SelectedServerID = srv.ID

	// 如果已有代理在运行，先停止
	if slp.appState.XrayInstance != nil {
		slp.appState.XrayInstance.Stop()
		slp.appState.XrayInstance = nil
	}

	// 启动代理
	slp.startProxyWithServer(&srv)
}

// startProxyWithServer 使用指定的服务器启动代理
func (slp *ServerListPanel) startProxyWithServer(srv *config.Server) {
	// 使用固定的10080端口监听本地SOCKS5
	proxyPort := 10080

	// 记录开始启动日志
	if slp.appState != nil {
		slp.appState.AppendLog("INFO", "xray", fmt.Sprintf("开始启动xray-core代理: %s", srv.Name))
	}

	// 使用统一的日志文件路径（与应用日志使用同一个文件）
	unifiedLogPath := slp.appState.Logger.GetLogFilePath()

	// 创建xray配置，设置日志文件路径为统一日志文件
	xrayConfigJSON, err := xray.CreateXrayConfig(proxyPort, srv, unifiedLogPath)
	if err != nil {
		slp.logAndShowError("创建xray配置失败", err)
		slp.appState.Config.AutoProxyEnabled = false
		slp.appState.XrayInstance = nil
		slp.appState.UpdateProxyStatus()
		slp.saveConfigToDB()
		return
	}

	// 记录配置创建成功日志
	if slp.appState != nil {
		slp.appState.AppendLog("DEBUG", "xray", fmt.Sprintf("xray配置已创建: %s", srv.Name))
	}

	// 创建日志回调函数，将 xray 日志转发到应用日志系统
	logCallback := func(level, message string) {
		if slp.appState != nil {
			slp.appState.AppendLog(level, "xray", message)
		}
	}

	// 创建xray实例，并设置日志回调
	xrayInstance, err := xray.NewXrayInstanceFromJSONWithCallback(xrayConfigJSON, logCallback)
	if err != nil {
		slp.logAndShowError("创建xray实例失败", err)
		slp.appState.Config.AutoProxyEnabled = false
		slp.appState.XrayInstance = nil
		slp.appState.UpdateProxyStatus()
		slp.saveConfigToDB()
		return
	}

	// 启动xray实例
	err = xrayInstance.Start()
	if err != nil {
		slp.logAndShowError("启动xray实例失败", err)
		slp.appState.Config.AutoProxyEnabled = false
		slp.appState.XrayInstance = nil
		slp.appState.UpdateProxyStatus()
		slp.saveConfigToDB()
		return
	}

	// 启动成功，设置端口信息
	xrayInstance.SetPort(proxyPort)
	slp.appState.XrayInstance = xrayInstance
	slp.appState.Config.AutoProxyEnabled = true
	slp.appState.Config.AutoProxyPort = proxyPort

	// 记录日志（统一日志记录）
	if slp.appState.Logger != nil {
		slp.appState.Logger.InfoWithType(logging.LogTypeProxy, "xray-core代理已启动: %s (端口: %d)", srv.Name, proxyPort)
	}

	// 追加日志到日志面板
	if slp.appState != nil {
		slp.appState.AppendLog("INFO", "xray", fmt.Sprintf("xray-core代理已启动: %s (端口: %d)", srv.Name, proxyPort))
		slp.appState.AppendLog("INFO", "xray", fmt.Sprintf("服务器信息: %s:%d, 协议: %s", srv.Addr, srv.Port, srv.ProtocolType))
	}

	slp.Refresh()
	// 更新状态绑定（使用双向绑定，UI 会自动更新）
	slp.appState.UpdateProxyStatus()

	slp.appState.Window.SetTitle(fmt.Sprintf("代理已启动: %s (端口: %d)", srv.Name, proxyPort))

	// 保存配置到数据库
	slp.saveConfigToDB()
}

// StartProxyForSelected 对外暴露的“启动当前选中服务器”接口，供主界面一键按钮等复用。
// 内部直接复用现有 onStartProxyFromSelected 逻辑，避免重复实现。
func (slp *ServerListPanel) StartProxyForSelected() {
	slp.onStartProxyFromSelected()
}

// logAndShowError 记录日志并显示错误对话框（统一错误处理）
func (slp *ServerListPanel) logAndShowError(message string, err error) {
	if slp.appState != nil && slp.appState.Logger != nil {
		slp.appState.Logger.Error("%s: %v", message, err)
	}
	if slp.appState != nil && slp.appState.Window != nil {
		slp.appState.Window.SetTitle(fmt.Sprintf("%s: %v", message, err))
	}
}

// saveConfigToDB 保存应用配置到数据库（统一配置保存）
func (slp *ServerListPanel) saveConfigToDB() {
	if slp.appState == nil || slp.appState.Config == nil {
		return
	}
	cfg := slp.appState.Config

	// 保存配置到数据库
	database.SetAppConfig("logLevel", cfg.LogLevel)
	database.SetAppConfig("logFile", cfg.LogFile)
	database.SetAppConfig("autoProxyEnabled", strconv.FormatBool(cfg.AutoProxyEnabled))
	database.SetAppConfig("autoProxyPort", strconv.Itoa(cfg.AutoProxyPort))
}

// onStopProxy 停止代理
func (slp *ServerListPanel) onStopProxy() {
	stopped := false

	// 停止xray实例
	if slp.appState.XrayInstance != nil {
		if slp.appState != nil {
			slp.appState.AppendLog("INFO", "xray", "正在停止xray-core代理...")
		}

		err := slp.appState.XrayInstance.Stop()
		if err != nil {
			// 停止失败，记录日志并显示错误（统一错误处理）
			slp.logAndShowError("停止xray代理失败", err)
			return
		}

		slp.appState.XrayInstance = nil
		stopped = true

		// 记录日志（统一日志记录）
		if slp.appState.Logger != nil {
			slp.appState.Logger.InfoWithType(logging.LogTypeProxy, "xray-core代理已停止")
		}

		// 追加日志到日志面板
		if slp.appState != nil {
			slp.appState.AppendLog("INFO", "xray", "xray-core代理已停止")
		}
	}

	if stopped {
		// 停止成功
		slp.appState.Config.AutoProxyEnabled = false
		slp.appState.Config.AutoProxyPort = 0

		// 更新状态绑定
		slp.appState.UpdateProxyStatus()

		// 保存配置到数据库
		slp.saveConfigToDB()

		slp.appState.Window.SetTitle("代理已停止")
	} else {
		slp.appState.Window.SetTitle("代理未运行")
	}
}

// StopProxy 对外暴露的“停止代理”接口，供主界面一键按钮等复用。
// 内部直接复用现有 onStopProxy 逻辑。
func (slp *ServerListPanel) StopProxy() {
	slp.onStopProxy()
}

// onTestAll 一键测延迟
func (slp *ServerListPanel) onTestAll() {
	// 在goroutine中执行测速
	go func() {
		servers := slp.appState.ServerManager.ListServers()
		enabledCount := 0
		for _, s := range servers {
			if s.Enabled {
				enabledCount++
			}
		}

		// 记录开始测速日志
		if slp.appState != nil {
			slp.appState.AppendLog("INFO", "ping", fmt.Sprintf("开始一键测速，共 %d 个启用的服务器", enabledCount))
		}

		results := slp.appState.PingManager.TestAllServersDelay()

		// 统计结果并记录每个服务器的详细日志
		successCount := 0
		failCount := 0
		for _, srv := range servers {
			if !srv.Enabled {
				continue
			}
			delay, exists := results[srv.ID]
			if !exists {
				continue
			}
			if delay > 0 {
				successCount++
				if slp.appState != nil {
					slp.appState.AppendLog("INFO", "ping", fmt.Sprintf("服务器 %s (%s:%d) 测速完成: %d ms", srv.Name, srv.Addr, srv.Port, delay))
				}
			} else {
				failCount++
				if slp.appState != nil {
					slp.appState.AppendLog("ERROR", "ping", fmt.Sprintf("服务器 %s (%s:%d) 测速失败", srv.Name, srv.Addr, srv.Port))
				}
			}
		}

		// 记录完成日志
		if slp.appState != nil {
			slp.appState.AppendLog("INFO", "ping", fmt.Sprintf("一键测速完成: 成功 %d 个，失败 %d 个，共测试 %d 个服务器", successCount, failCount, len(results)))
		}

		// 更新UI（需要在主线程中执行）
		fyne.Do(func() {
			slp.Refresh()
			slp.appState.Window.SetTitle(fmt.Sprintf("测速完成，共测试 %d 个服务器", len(results)))
		})
	}()
}

// ServerListItem 自定义服务器列表项（支持右键菜单和多列显示）
type ServerListItem struct {
	widget.BaseWidget
	id          widget.ListItemID
	panel       *ServerListPanel
	container   *fyne.Container
	bgContainer *fyne.Container // 背景容器
	regionLabel *widget.Label
	nameLabel   *widget.Label
	portLabel   *widget.Label
	delayLabel  *widget.Label
	isSelected  bool // 是否选中
	isEven      bool // 是否为偶数行（用于交替颜色）
}

// NewServerListItem 创建新的服务器列表项
func NewServerListItem() *ServerListItem {
	// 创建各列标签（地区 / 名称 / 端口 / 延迟）
	regionLabel := widget.NewLabel("")
	regionLabel.Wrapping = fyne.TextTruncate

	nameLabel := widget.NewLabel("")
	nameLabel.Wrapping = fyne.TextTruncate
	nameLabel.TextStyle = fyne.TextStyle{Bold: true}

	portLabel := widget.NewLabel("")
	portLabel.Alignment = fyne.TextAlignCenter

	delayLabel := widget.NewLabel("")
	delayLabel.Alignment = fyne.TextAlignCenter

	// 创建图标（服务器图标）
	serverIcon := widget.NewIcon(theme.ComputerIcon())

	// 创建容器，使用网格布局，确保所有列都能显示
	// 为每列添加一个包含标签的固定大小容器，并添加内边距
	// 使用 GridWrap 来控制宽度，而不是使用已废弃的 Resize
	regionContainer := container.NewGridWrap(
		fyne.NewSize(80, 32),
		container.NewPadded(container.NewStack(regionLabel)),
	)

	nameContainer := container.NewGridWrap(
		fyne.NewSize(220, 32), // 设置合理的宽度和高度
		container.NewHBox(
			serverIcon,
			NewSpacer(SpacingSmall),
			container.NewStack(nameLabel),
		),
	)

	portContainer := container.NewGridWrap(
		fyne.NewSize(80, 32),
		container.NewPadded(container.NewStack(portLabel)),
	)

	delayContainer := container.NewGridWrap(
		fyne.NewSize(90, 32),
		container.NewPadded(container.NewStack(delayLabel)),
	)

	// 使用网格布局组织各列容器
	gridContainer := container.NewGridWithColumns(4,
		regionContainer,
		nameContainer,
		portContainer,
		delayContainer,
	)
	// 添加整体内边距，使列表项更美观
	contentContainer := container.NewPadded(gridContainer)

	// 创建带背景的容器（用于交替颜色和选中效果）
	bgContainer := container.NewWithoutLayout()
	bgContainer.Add(contentContainer)

	item := &ServerListItem{
		container:   contentContainer,
		bgContainer: bgContainer,
		regionLabel: regionLabel,
		nameLabel:   nameLabel,
		portLabel:   portLabel,
		delayLabel:  delayLabel,
		isSelected:  false,
		isEven:      false,
	}
	item.ExtendBaseWidget(item)
	return item
}

// CreateRenderer 创建渲染器
func (s *ServerListItem) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(s.bgContainer)
}

// TappedSecondary 处理右键点击事件
func (s *ServerListItem) TappedSecondary(pe *fyne.PointEvent) {
	if s.panel == nil {
		return
	}
	s.panel.onRightClick(s.id, pe)
}

// Update  更新服务器列表项的信息
func (s *ServerListItem) Update(server config.Server) {
	fyne.Do(func() {
		// 更新选中状态
		s.isSelected = server.Selected

		// 地区：从名称中尝试提取前缀（例如 "US - LA" -> "US"）
		region := "-"
		if server.Name != "" {
			nameLower := strings.TrimSpace(server.Name)
			// 使用 "-" 或 空格 作为简单分隔符
			if idx := strings.Index(nameLower, "-"); idx > 0 {
				region = strings.TrimSpace(nameLower[:idx])
			} else if idx := strings.Index(nameLower, " "); idx > 0 {
				region = strings.TrimSpace(nameLower[:idx])
			}
		}
		s.regionLabel.SetText(region)

		// 服务器名称（带选中标记和图标）
		prefix := ""
		if server.Selected {
			prefix = "★ "
			s.nameLabel.TextStyle = fyne.TextStyle{Bold: true}
		} else {
			s.nameLabel.TextStyle = fyne.TextStyle{Bold: false}
		}
		if !server.Enabled {
			prefix += "[禁用] "
			s.nameLabel.Importance = widget.LowImportance
		} else {
			s.nameLabel.Importance = widget.MediumImportance
		}
		s.nameLabel.SetText(prefix + server.Name)

		// 端口
		s.portLabel.SetText(strconv.Itoa(server.Port))
		if !server.Enabled {
			s.portLabel.Importance = widget.LowImportance
		} else {
			s.portLabel.Importance = widget.MediumImportance
		}

		// 延迟 - 根据延迟值设置重要性（颜色）
		delayText := "未测"
		if server.Delay > 0 {
			delayText = fmt.Sprintf("%d ms", server.Delay)
			// 延迟越低，重要性越高（颜色更明显）
			if server.Delay < 100 {
				s.delayLabel.Importance = widget.HighImportance
			} else if server.Delay < 300 {
				s.delayLabel.Importance = widget.MediumImportance
			} else {
				s.delayLabel.Importance = widget.LowImportance
			}
		} else if server.Delay < 0 {
			delayText = "失败"
			s.delayLabel.Importance = widget.DangerImportance
		} else {
			s.delayLabel.Importance = widget.LowImportance
		}
		s.delayLabel.SetText(delayText)
	})
}
