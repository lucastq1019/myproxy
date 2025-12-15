package ui

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
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
	statusPanel    *StatusPanel // 状态面板引用（用于刷新）
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
}

// Build 构建并返回服务器列表面板的 UI 组件。
// 返回：包含操作按钮和服务器列表的容器组件
func (slp *ServerListPanel) Build() fyne.CanvasObject {
	// 操作按钮
	testAllBtn := widget.NewButton("一键测延迟", slp.onTestAll)
	startProxyBtn := widget.NewButton("启动代理", slp.onStartProxyFromSelected)
	stopProxyBtn := widget.NewButton("停止代理", slp.onStopProxy)

	// 服务器列表标题和按钮
	headerArea := container.NewHBox(
		widget.NewLabel("服务器列表"),
		layout.NewSpacer(),
		testAllBtn,
		startProxyBtn,
		stopProxyBtn,
	)

	// 服务器列表滚动区域（不再展示右侧详情）
	serverScroll := container.NewScroll(slp.serverList)

	// 返回包含标题和列表的容器
	return container.NewBorder(
		headerArea,
		nil,
		nil,
		nil,
		serverScroll,
	)
}

// Refresh 刷新服务器列表的显示，使 UI 反映最新的服务器数据。
func (slp *ServerListPanel) Refresh() {
	fyne.Do(func() {
		slp.serverList.Refresh()
	})
}

// getServerCount 获取服务器数量
func (slp *ServerListPanel) getServerCount() int {
	return len(slp.appState.ServerManager.ListServers())
}

// createServerItem 创建服务器列表项
func (slp *ServerListPanel) createServerItem() fyne.CanvasObject {
	return NewServerListItem()
}

// updateServerItem 更新服务器列表项
func (slp *ServerListPanel) updateServerItem(id widget.ListItemID, obj fyne.CanvasObject) {
	servers := slp.appState.ServerManager.ListServers()
	if id < 0 || id >= len(servers) {
		return
	}

	srv := servers[id]
	item := obj.(*ServerListItem)

	// 设置面板引用和ID
	item.panel = slp
	item.id = id

	// 构建显示文本
	prefix := ""
	if srv.Selected {
		prefix = "★ "
	}
	if !srv.Enabled {
		prefix += "[禁用] "
	}

	delay := "未测"
	if srv.Delay > 0 {
		delay = fmt.Sprintf("%d ms", srv.Delay)
	} else if srv.Delay < 0 {
		delay = "失败"
	}

	text := fmt.Sprintf("%s%s  %s:%d  [%s]", prefix, srv.Name, srv.Addr, srv.Port, delay)
	item.SetText(text)
}

// onSelected 服务器选中事件
func (slp *ServerListPanel) onSelected(id widget.ListItemID) {
	servers := slp.appState.ServerManager.ListServers()
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
	servers := slp.appState.ServerManager.ListServers()
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
	servers := slp.appState.ServerManager.ListServers()
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
	if slp.appState.ProxyForwarder != nil && slp.appState.ProxyForwarder.IsRunning {
		slp.appState.ProxyForwarder.Stop()
		slp.appState.ProxyForwarder = nil
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
	if slp.appState.ProxyForwarder != nil && slp.appState.ProxyForwarder.IsRunning {
		slp.appState.ProxyForwarder.Stop()
		slp.appState.ProxyForwarder = nil
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

	// 如果还有旧的转发器在运行，也停止它（兼容旧代码）
	if slp.appState.ProxyForwarder != nil && slp.appState.ProxyForwarder.IsRunning {
		err := slp.appState.ProxyForwarder.Stop()
		if err != nil {
			// 停止失败，记录日志并显示错误（统一错误处理）
			slp.logAndShowError("停止代理转发器失败", err)
			return
		}

		slp.appState.ProxyForwarder = nil
		stopped = true

		// 记录日志（统一日志记录）
		if slp.appState.Logger != nil {
			slp.appState.Logger.InfoWithType(logging.LogTypeProxy, "代理转发器已停止")
		}

		// 追加日志到日志面板
		if slp.appState != nil {
			slp.appState.AppendLog("INFO", "proxy", "代理转发器已停止")
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

// getSelectedIndex 获取当前选中的索引
func (slp *ServerListPanel) getSelectedIndex() widget.ListItemID {
	servers := slp.appState.ServerManager.ListServers()
	for i, srv := range servers {
		if srv.ID == slp.appState.SelectedServerID {
			return widget.ListItemID(i)
		}
	}
	return -1
}

// ServerListItem 自定义服务器列表项（支持右键菜单）
type ServerListItem struct {
	widget.BaseWidget
	label *widget.Label
	id    widget.ListItemID
	panel *ServerListPanel
}

// NewServerListItem 创建新的服务器列表项
func NewServerListItem() *ServerListItem {
	item := &ServerListItem{
		label: widget.NewLabel(""),
	}
	item.ExtendBaseWidget(item)
	return item
}

// CreateRenderer 创建渲染器
func (s *ServerListItem) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(s.label)
}

// TappedSecondary 处理右键点击事件
func (s *ServerListItem) TappedSecondary(pe *fyne.PointEvent) {
	if s.panel == nil {
		return
	}
	s.panel.onRightClick(s.id, pe)
}

// SetText 设置文本
func (s *ServerListItem) SetText(text string) {
	fyne.Do(func() {
		s.label.SetText(text)
	})
}
