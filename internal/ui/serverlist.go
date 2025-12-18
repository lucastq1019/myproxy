package ui

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
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
	appState       *AppState
	serverList     *widget.List
	onServerSelect func(server config.Server)
	statusPanel    *StatusPanel // 状态面板引用（用于刷新和一键操作）

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

	// 服务器列表（行高通过ServerListItem的MinSize控制，设置为52px改善可读性和点击区域）
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
// 返回：包含返回按钮、操作按钮和服务器列表的容器组件
func (slp *ServerListPanel) Build() fyne.CanvasObject {
	// 返回按钮 - 返回主界面
	backBtn := NewStyledButton("← 返回", nil, func() {
		if slp.appState != nil && slp.appState.MainWindow != nil {
			slp.appState.MainWindow.ShowHomePage()
		}
	})

	// 操作按钮 - 一键测速（符合 UI.md 设计）
	testAllBtn := NewStyledButton("测速", theme.ViewRefreshIcon(), slp.onTestAll)

	// 收藏按钮（显示收藏节点）
	favoriteBtn := NewStyledButton("收藏", nil, func() {
		// TODO: 实现收藏节点筛选功能
		if slp.appState != nil && slp.appState.Window != nil {
			slp.appState.Window.SetTitle("收藏功能开发中")
		}
	})

	// 订阅管理按钮
	subscriptionBtn := NewStyledButton("订阅", theme.SettingsIcon(), func() {
		// 跳转到订阅管理页面
		if slp.appState != nil && slp.appState.MainWindow != nil {
			slp.appState.MainWindow.ShowSubscriptionPage()
		}
	})

	// 刷新按钮
	refreshBtn := NewStyledButton("刷新", theme.ViewRefreshIcon(), func() {
		if slp.appState != nil && slp.appState.ServerManager != nil {
			// 刷新服务器列表
			slp.Refresh()
			if slp.appState.Window != nil {
				slp.appState.Window.SetTitle("列表已刷新")
			}
		}
	})

	// 全局搜索栏：支持按名称、地址、协议实时搜索（符合 UI.md 设计）
	// 使用弹性布局：搜索框自适应剩余空间，按钮固定大小
	slp.searchEntry = widget.NewEntry()
	slp.searchEntry.SetPlaceHolder("🔍 搜索节点名称或地区...")
	slp.searchEntry.OnChanged = func(value string) {
		// 记录小写关键字，便于不区分大小写匹配
		slp.searchText = strings.ToLower(strings.TrimSpace(value))
		slp.Refresh()
	}

	// 顶部栏：返回按钮 + 搜索框 + 操作按钮组
	// 符合 UI.md 设计：[← 返回] [搜索框🔍] [⭐收藏] [📊测速] [⚙️订阅管理] [🔄刷新]
	headerArea := container.NewPadded(container.NewHBox(
		backBtn,                // 返回按钮
		NewSpacer(SpacingLarge), // 间距
		slp.searchEntry,        // 搜索框自适应剩余空间
		NewSpacer(SpacingLarge), // 间距
		favoriteBtn,            // 收藏按钮
		testAllBtn,             // 一键测速按钮
		subscriptionBtn,        // 订阅管理按钮
		refreshBtn,             // 刷新按钮
	))

	// 创建列标题行
	columnHeaders := slp.createColumnHeaders()

	// 分组标题
	allNodesHeader := NewSubtitleLabel("🌍 所有节点 (All Nodes)")

	// 服务器列表滚动区域
	serverScroll := container.NewScroll(slp.serverList)

	// 列表上方插入分组标题（目前所有节点都显示在“所有节点”下方）
	// 顶部固定内容：分组标题 + 分隔符 + 列标题 + 分隔符
	topContent := container.NewVBox(
		// TODO: 未来在这里插入真正的“收藏”节点列表
		allNodesHeader,
		NewSeparator(),
		columnHeaders,
		NewSeparator(),
	)

	// 使用Border布局：顶部放固定内容，中心放滚动列表（自动填充剩余空间）
	listContent := container.NewBorder(
		topContent,
		nil,
		nil,
		nil,
		serverScroll, // 中心位置，自动填充剩余空间
	)

	// 返回包含标题和列表的容器
		return container.NewBorder(
		headerArea,
		nil,
		nil,
		nil,
		listContent,
	)
}

// createColumnHeaders 创建列标题行，使用弹性布局
// 根据 UI.md 设计：地区 | 节点名称 | 延迟 
func (slp *ServerListPanel) createColumnHeaders() fyne.CanvasObject {
	// 创建列标题标签：地区 / 节点名称 / 延迟 
	regionHeader := NewSubtitleLabel("地区")
	regionHeader.Alignment = fyne.TextAlignCenter

	nameHeader := NewSubtitleLabel("节点名称")
	nameHeader.Alignment = fyne.TextAlignLeading

	delayHeader := NewSubtitleLabel("延迟")
	delayHeader.Alignment = fyne.TextAlignCenter
	// 使用弹性布局：GridWithColumns会自动分配空间，每个列内部内容自适应
	// 地区列：居中显示，使用Padded添加内边距
	regionContainer := container.NewPadded(regionHeader)

	// 名称列：仅保留标题，使用Padded添加内边距
	nameContainer := container.NewPadded(nameHeader)

	// 延迟列：居中显示，使用Padded添加内边距
	delayContainer := container.NewPadded(delayHeader)

	// 使用网格布局组织各列容器（5列），GridWithColumns会自动平均分配空间
	gridContainer := container.NewGridWithColumns(5,
		regionContainer,
		nameContainer,
		delayContainer,
	)

	return gridContainer
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
	return NewServerListItem(slp)
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
	item.isSelected = srv.Selected // 设置是否选中
	// 检查是否为当前连接的节点
	item.isConnected = (slp.appState != nil && slp.appState.XrayInstance != nil && 
		slp.appState.XrayInstance.IsRunning() && slp.appState.SelectedServerID == srv.ID)

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
	delayLabel  *widget.Label
	statusIcon  *widget.Icon   // 在线/离线状态图标
	menuButton  *widget.Button // 右侧"..."菜单按钮
	isSelected  bool           // 是否选中
	isConnected bool           // 是否当前连接
}

// NewServerListItem 创建新的服务器列表项
// 参数：
//   - panel: ServerListPanel实例
func NewServerListItem(panel *ServerListPanel) *ServerListItem {

	// 创建各列标签（地区 / 名称 / 延迟）- 根据 UI.md 设计，移除端口列
	regionLabel := widget.NewLabel("")
	regionLabel.Wrapping = fyne.TextTruncate

	nameLabel := widget.NewLabel("")
	nameLabel.Wrapping = fyne.TextTruncate
	nameLabel.TextStyle = fyne.TextStyle{Bold: true}

	delayLabel := widget.NewLabel("")
	delayLabel.Alignment = fyne.TextAlignCenter

	// 使用弹性布局：GridWithColumns会自动分配空间，每个列内部内容自适应
	// 与列标题布局保持一致，确保对齐
	// 地区列：居中显示，使用Padded添加内边距
	regionContainer := container.NewPadded(regionLabel)

	// 名称列：仅保留标签，使用Padded添加内边距
	nameContainer := container.NewPadded(nameLabel)

	// 延迟列：居中显示，使用Padded添加内边距
	delayContainer := container.NewPadded(delayLabel)

	// 使用网格布局组织各列容器（5列：地区、名称、延迟、）
	// 与列标题使用相同的布局方式，确保对齐
	gridContainer := container.NewGridWithColumns(3,
		regionContainer,
		nameContainer,
		delayContainer,
	)

	// 创建带背景的容器（用于交替颜色和选中效果）
	bgContainer := container.NewWithoutLayout()
	bgContainer.Add(gridContainer)

	item := &ServerListItem{
		container:   gridContainer,
		bgContainer: bgContainer,
		regionLabel: regionLabel,
		nameLabel:   nameLabel,
		delayLabel:  delayLabel,
		isSelected:  false,
		isConnected: false,
	}
	item.ExtendBaseWidget(item)
	return item
}

// MinSize 返回列表项的最小尺寸（设置行高为52px，符合UI改进建议：48-56px）
func (s *ServerListItem) MinSize() fyne.Size {
	return fyne.NewSize(0, 52)
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
		
		// 检查是否为当前连接的节点
		if s.panel != nil && s.panel.appState != nil {
			s.isConnected = (s.panel.appState.XrayInstance != nil && 
				s.panel.appState.XrayInstance.IsRunning() && 
				s.panel.appState.SelectedServerID == server.ID)
		}

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

		// 服务器名称（带选中标记和连接状态）
		prefix := ""
		if s.isConnected {
			prefix = "🔵 " // 当前连接的节点用蓝色标记
			s.nameLabel.TextStyle = fyne.TextStyle{Bold: true}
		} else if server.Selected {
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

		// 延迟 - 根据延迟值设置重要性（颜色）
		// 符合 UI.md 设计：< 100ms绿色(🟢)，100-200ms黄色(🟡)，> 200ms红色(🔴)
		// 空状态处理：显示"测速中..."或"未测速"
		delayText := "未测速"
		if server.Delay > 0 {
			delayText = fmt.Sprintf("%d ms", server.Delay)
			// 延迟颜色规则：< 100ms绿色，100-200ms黄色，> 200ms红色
			if server.Delay < 100 {
				s.delayLabel.Importance = widget.HighImportance // 绿色
			} else if server.Delay <= 200 {
				s.delayLabel.Importance = widget.MediumImportance // 黄色
			} else {
				s.delayLabel.Importance = widget.DangerImportance // 红色
			}
		} else if server.Delay < 0 {
			delayText = "测试失败"
			s.delayLabel.Importance = widget.DangerImportance
		} else {
			delayText = "未测速"
			s.delayLabel.Importance = widget.LowImportance
		}
		s.delayLabel.SetText(delayText)

		// 更新在线/离线状态图标
		if s.statusIcon != nil {
			if server.Delay > 0 {
				// 有延迟数据，表示在线
				s.statusIcon.SetResource(theme.ConfirmIcon())
			} else if server.Delay < 0 {
				// 延迟为负，表示测试失败
				s.statusIcon.SetResource(theme.CancelIcon())
			} else {
				// 未测试，显示未知状态
				s.statusIcon.SetResource(theme.QuestionIcon())
			}
		}

		// 设置菜单按钮的点击事件（快速操作菜单）
		if s.menuButton != nil && s.panel != nil {
			s.menuButton.OnTapped = func() {
				s.showQuickMenu(server)
			}
		}

		// 如果当前连接，添加蓝色边框效果（通过背景容器实现）
		if s.isConnected {
			// 可以通过设置背景颜色或边框来突出显示
			// 这里暂时通过选中状态来体现
		}
	})
}

// showQuickMenu 显示快速操作菜单
func (s *ServerListItem) showQuickMenu(server config.Server) {
	if s.panel == nil || s.panel.appState == nil || s.panel.appState.Window == nil {
		return
	}

	// 创建快速操作菜单
	menu := fyne.NewMenu("",
		fyne.NewMenuItem("连接", func() {
			if s.panel != nil {
				s.panel.onStartProxy(s.id)
			}
		}),
		fyne.NewMenuItem("测速", func() {
			if s.panel != nil {
				s.panel.onTestSpeed(s.id)
			}
		}),
		fyne.NewMenuItem("收藏", func() {
			// TODO: 实现收藏功能
			if s.panel != nil && s.panel.appState != nil {
				s.panel.appState.Window.SetTitle("收藏功能开发中")
			}
		}),
		fyne.NewMenuItem("复制信息", func() {
			// TODO: 实现复制节点信息功能
			info := fmt.Sprintf("名称: %s\n地址: %s:%d\n协议: %s", 
				server.Name, server.Addr, server.Port, server.ProtocolType)
			if s.panel != nil && s.panel.appState != nil && s.panel.appState.Window != nil {
				s.panel.appState.Window.Clipboard().SetContent(info)
				s.panel.appState.Window.SetTitle("节点信息已复制到剪贴板")
			}
		}),
	)

	// 显示菜单
	popup := widget.NewPopUpMenu(menu, s.panel.appState.Window.Canvas())
	// 在菜单按钮位置显示
	if s.menuButton != nil {
		pos := fyne.NewPos(s.menuButton.Position().X, s.menuButton.Position().Y+s.menuButton.Size().Height)
		popup.ShowAtPosition(pos)
	}
}
