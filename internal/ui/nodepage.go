package ui

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"myproxy.com/p/internal/database"
	"myproxy.com/p/internal/logging"
)

// NodePage 管理服务器列表的显示和操作。
// 它支持服务器选择、延迟测试、代理启动/停止等功能，并提供右键菜单操作。
type NodePage struct {
	appState       *AppState
	nodes          []*database.Node
	selectedNode   *database.Node
	selectedIndex  int
	nodesBinding   binding.UntypedList // 服务器列表绑定
	list           *widget.List        // 列表组件
	content        fyne.CanvasObject   // 内容容器

	// 搜索与过滤相关
	searchEntry *widget.Entry // 节点搜索输入框
	searchText  string        // 当前搜索关键字（小写）
}

// NewNodePage 创建节点管理页面
func NewNodePage(appState *AppState) *NodePage {
	np := &NodePage{
		appState:      appState,
		nodesBinding: binding.NewUntypedList(),
	}
	np.loadNodes()
	
	// 监听绑定数据变化，自动刷新列表
	np.nodesBinding.AddListener(binding.NewDataListener(func() {
		if np.list != nil {
			np.list.Refresh()
		}
	}))
	
	return np
}


func (np *NodePage) loadNodes() {
	nodes, err := database.GetAllServers()
	if err != nil {
		np.nodes = []*database.Node{}
	} else {
		// 转换为指针切片
		np.nodes = make([]*database.Node, len(nodes))
		for i := range nodes {
			np.nodes[i] = &nodes[i]
		}
	}
	
	// 更新绑定数据，触发 UI 自动刷新
	np.updateNodesBinding()
}

// updateNodesBinding 更新节点列表绑定数据
func (np *NodePage) updateNodesBinding() {
	// 将节点列表转换为 any 类型切片
	items := make([]any, len(np.nodes))
	for i, node := range np.nodes {
		items[i] = node
	}
	
	// 使用 Set 方法替换整个列表，这会触发绑定更新
	_ = np.nodesBinding.Set(items)
}


// // SetOnServerSelect 设置服务器选中时的回调函数。
// // 参数：
// //   - callback: 当用户选中服务器时调用的回调函数
// func (np *NodePage) SetOnServerSelect(callback func(server database.Node)) {
// 	np.onServerSelect = callback
// }

// Build 构建并返回服务器列表面板的 UI 组件。
// 返回：包含返回按钮、操作按钮和服务器列表的容器组件
func (np *NodePage) Build() fyne.CanvasObject {
	// 1. 返回按钮
	backBtn := widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() {
		if np.appState != nil && np.appState.MainWindow != nil {
			np.appState.MainWindow.Back()
		}
	})
	backBtn.Importance = widget.LowImportance

	// 2. 操作按钮组（参考 subscriptionpage 风格）
	testAllBtn := widget.NewButtonWithIcon("测速", theme.ViewRefreshIcon(), np.onTestAll)
	testAllBtn.Importance = widget.LowImportance

	subscriptionBtn := widget.NewButtonWithIcon("订阅", theme.SettingsIcon(), func() {
		if np.appState != nil && np.appState.MainWindow != nil {
			np.appState.MainWindow.ShowSubscriptionPage()
		}
	})
	subscriptionBtn.Importance = widget.LowImportance

	refreshBtn := widget.NewButtonWithIcon("刷新", theme.ViewRefreshIcon(), func() {
		if np.appState != nil && np.appState.ServerManager != nil {
			np.Refresh()
			if np.appState.Window != nil {
				np.appState.Window.SetTitle("列表已刷新")
			}
		}
	})
	refreshBtn.Importance = widget.LowImportance

	// 3. 头部栏布局（返回按钮 + 操作按钮）
	headerBar := container.NewHBox(
		backBtn,
		layout.NewSpacer(),
		testAllBtn,
		subscriptionBtn,
		refreshBtn,
	)

	// 4. 组合头部区域（添加分隔线，移除 padding 降低高度）
	headerStack := container.NewVBox(
		headerBar, // 移除 padding 降低功能栏高度
		canvas.NewLine(theme.Color(theme.ColorNameSeparator)),
	)

	// 5. 搜索框（单独一行，在功能栏下方）
	np.searchEntry = widget.NewEntry()
	np.searchEntry.SetPlaceHolder("搜索节点名称或地区...")
	np.searchEntry.OnChanged = func(value string) {
		// 记录小写关键字，便于不区分大小写匹配
		np.searchText = strings.ToLower(strings.TrimSpace(value))
		np.Refresh()
	}
	// 支持回车键搜索
	np.searchEntry.OnSubmitted = func(value string) {
		// 触发搜索
		np.searchText = strings.ToLower(strings.TrimSpace(value))
		np.Refresh()
	}

	// 搜索按钮（放大镜图标）
	searchBtn := widget.NewButtonWithIcon("", theme.SearchIcon(), func() {
		// 触发搜索
		value := np.searchEntry.Text
		np.searchText = strings.ToLower(strings.TrimSpace(value))
		np.Refresh()
	})
	searchBtn.Importance = widget.LowImportance

	// 搜索栏布局（搜索框 + 搜索按钮，移除 padding 降低高度）
	searchBar := container.NewBorder(
		nil, nil, nil,
		searchBtn,
		np.searchEntry, // 移除 padding 降低搜索框高度
	)

	// 6. 表格头（与列表项对齐，使用最小高度）
	regionHeader := widget.NewLabel("地区")
	regionHeader.Alignment = fyne.TextAlignCenter
	regionHeader.TextStyle = fyne.TextStyle{Bold: true}
	regionHeader.Importance = widget.MediumImportance

	nameHeader := widget.NewLabel("节点名称")
	nameHeader.Alignment = fyne.TextAlignLeading
	nameHeader.TextStyle = fyne.TextStyle{Bold: true}
	nameHeader.Importance = widget.MediumImportance

	delayHeader := widget.NewLabel("延迟")
	delayHeader.Alignment = fyne.TextAlignTrailing
	delayHeader.TextStyle = fyne.TextStyle{Bold: true}
	delayHeader.Importance = widget.MediumImportance

	// 表头使用与列表项相同的 GridWithColumns(3) 布局，确保对齐
	// 使用最小 padding 减少高度
	tableHeader := container.NewGridWithColumns(3,
		regionHeader, // 地区列（移除 padding 减少高度）
		nameHeader,   // 名称列
		delayHeader,  // 延迟列
	)

	// 7. 节点列表（支持滚动，参考 subscriptionpage）
	np.list = widget.NewList(
		np.getNodeCount,
		np.createNodeItem,
		np.updateNodeItem,
	)

	// 包装在滚动容器中并设置最小尺寸确保布局占满
	scrollList := container.NewScroll(np.list)

	// 8. 组合布局：头部 + 搜索栏 + 表头 + 列表
	// 移除所有不必要的 padding，降低高度
	np.content = container.NewBorder(
		container.NewVBox(
			headerStack,
			searchBar, // 移除 padding
			tableHeader, // 表头直接放置，不添加额外 padding
			canvas.NewLine(theme.Color(theme.ColorNameSeparator)),
		),
		nil, nil, nil,
		container.NewPadded(scrollList),
	)

	return np.content
}


// Refresh 刷新节点列表的显示，使 UI 反映最新的节点数据。
func (np *NodePage) Refresh() {
	np.loadNodes()
	// 绑定数据更新后会自动触发列表刷新，无需手动调用
}

// getNodeCount 获取节点数量
func (np *NodePage) getNodeCount() int {
	return len(np.getFilteredNodes())
}

// getFilteredNodes 根据当前搜索关键字返回过滤后的节点列表。
// 支持按名称、地址、协议类型进行不区分大小写的匹配。
func (np *NodePage) getFilteredNodes() []*database.Node {
	// 如果没有搜索关键字，直接返回完整列表
	if np.searchText == "" {
		return np.nodes
	}

	filtered := make([]*database.Node, 0, len(np.nodes))
	for _, node := range np.nodes {
		name := strings.ToLower(node.Name)
		addr := strings.ToLower(node.Addr)
		protocol := strings.ToLower(node.ProtocolType)

		if strings.Contains(name, np.searchText) ||
			strings.Contains(addr, np.searchText) ||
			strings.Contains(protocol, np.searchText) {
			filtered = append(filtered, node)
		}
	}
	return filtered
}

// createNodeItem 创建节点列表项
func (np *NodePage) createNodeItem() fyne.CanvasObject {
	return NewServerListItem(np)
}

// updateNodeItem 更新节点列表项
func (np *NodePage) updateNodeItem(id widget.ListItemID, obj fyne.CanvasObject) {
	nodes := np.getFilteredNodes()
	if id < 0 || id >= len(nodes) {
		return
	}

	node := nodes[id]
	item := obj.(*ServerListItem)

	// 设置面板引用和ID
	item.panel = np
	item.id = id
	item.isSelected = node.Selected // 设置是否选中
	// 检查是否为当前连接的节点
	item.isConnected = (np.appState != nil && np.appState.XrayInstance != nil && 
		np.appState.XrayInstance.IsRunning() && np.appState.SelectedServerID == node.ID)

	// 使用新的Update方法更新多列信息
	item.Update(*node)
}

// // onSelected 服务器选中事件
// func (np *NodePage) onSelected(id widget.ListItemID) {
// 	servers := np.getFilteredServers()
// 	if id < 0 || id >= len(servers) {
// 		return
// 	}

// 	srv := servers[id]
// 	np.appState.SelectedServerID = srv.ID

// 	// 更新状态绑定（使用双向绑定，UI 会自动更新）
// 	if np.appState != nil {
// 		np.appState.UpdateProxyStatus()
// 	}

// 	// 调用回调
// 	if np.onServerSelect != nil {
// 		np.onServerSelect(srv)
// 	}
// }

// onRightClick 右键菜单 - 注释功能
// func (np *NodePage) onRightClick(id widget.ListItemID, ev *fyne.PointEvent) {
// 	nodes := np.getFilteredNodes()
// 	if id < 0 || id >= len(nodes) {
// 		return
// 	}

// 	node := nodes[id]
// 	np.appState.SelectedServerID = node.ID

// 	// 创建右键菜单
// 	menu := fyne.NewMenu("",
// 		fyne.NewMenuItem("测速", func() {
// 			np.onTestSpeed(id)
// 		}),
// 		fyne.NewMenuItem("启动代理", func() {
// 			np.onStartProxy(id)
// 		}),
// 		fyne.NewMenuItem("停止代理", func() {
// 			np.onStopProxy()
// 		}),
// 	)

// 	// 显示菜单
// 	popup := widget.NewPopUpMenu(menu, np.appState.Window.Canvas())
// 	popup.ShowAtPosition(ev.AbsolutePosition)
// }

// onTestSpeed 测速 - 注释功能
// func (np *NodePage) onTestSpeed(id widget.ListItemID) {
// 	nodes := np.getFilteredNodes()
// 	if id < 0 || id >= len(nodes) {
// 		return
// 	}

// 	node := nodes[id]

// 	// 在goroutine中执行测速
// 	go func() {
// 		// 记录开始测速日志
// 		if np.appState != nil {
// 			np.appState.AppendLog("INFO", "ping", fmt.Sprintf("开始测试服务器延迟: %s (%s:%d)", node.Name, node.Addr, node.Port))
// 		}

// 		delay, err := np.appState.PingManager.TestServerDelay(*node)
// 		if err != nil {
// 			// 记录失败日志
// 			if np.appState != nil {
// 				np.appState.AppendLog("ERROR", "ping", fmt.Sprintf("服务器 %s 测速失败: %v", node.Name, err))
// 			}
// 			fyne.Do(func() {
// 				np.appState.Window.SetTitle(fmt.Sprintf("测速失败: %v", err))
// 			})
// 			return
// 		}

// 		// 更新服务器延迟
// 		np.appState.ServerManager.UpdateServerDelay(node.ID, delay)

// 		// 记录成功日志
// 		if np.appState != nil {
// 			np.appState.AppendLog("INFO", "ping", fmt.Sprintf("服务器 %s 测速完成: %d ms", node.Name, delay))
// 		}

// 		// 更新UI（需要在主线程中执行）
// 		fyne.Do(func() {
// 			np.Refresh()
// 			// 更新状态绑定（使用双向绑定，UI 会自动更新）
// 			if np.appState != nil {
// 				np.appState.UpdateProxyStatus()
// 			}
// 			np.appState.Window.SetTitle(fmt.Sprintf("测速完成: %d ms", delay))
// 		})
// 	}()
// }

// onStartProxyFromSelected 从当前选中的服务器启动代理 - 注释功能
// func (np *NodePage) onStartProxyFromSelected() {
// 	if np.appState.SelectedServerID == "" {
// 		np.appState.Window.SetTitle("请先选择一个服务器")
// 		return
// 	}

// 	nodes := np.nodes
// 	var srv *database.Node
// 	for _, node := range nodes {
// 		if node.ID == np.appState.SelectedServerID {
// 			srv = node
// 			break
// 		}
// 	}

// 	if srv == nil {
// 		np.appState.Window.SetTitle("选中的服务器不存在")
// 		return
// 	}

// 	// 如果已有代理在运行，先停止
// 	if np.appState.XrayInstance != nil {
// 		np.appState.XrayInstance.Stop()
// 		np.appState.XrayInstance = nil
// 	}

// 	// 把当前的设置为选中
// 	np.appState.ServerManager.SelectServer(srv.ID)
// 	np.appState.SelectedServerID = srv.ID

// 	// 启动代理
// 	np.startProxyWithServer(srv)
// }

// onStartProxy 启动代理（右键菜单使用）- 注释功能
// func (np *NodePage) onStartProxy(id widget.ListItemID) {
// 	nodes := np.getFilteredNodes()
// 	if id < 0 || id >= len(nodes) {
// 		return
// 	}

// 	node := nodes[id]
// 	np.appState.ServerManager.SelectServer(node.ID)
// 	np.appState.SelectedServerID = node.ID

// 	// 如果已有代理在运行，先停止
// 	if np.appState.XrayInstance != nil {
// 		np.appState.XrayInstance.Stop()
// 		np.appState.XrayInstance = nil
// 	}

// 	// 启动代理
// 	np.startProxyWithServer(node)
// }

// startProxyWithServer 使用指定的服务器启动代理 - 注释功能
// func (np *NodePage) startProxyWithServer(srv *database.Node) {
// 	// 使用固定的10080端口监听本地SOCKS5
// 	proxyPort := 10080

// 	// 记录开始启动日志
// 	if np.appState != nil {
// 		np.appState.AppendLog("INFO", "xray", fmt.Sprintf("开始启动xray-core代理: %s", srv.Name))
// 	}

// 	// 使用统一的日志文件路径（与应用日志使用同一个文件）
// 	unifiedLogPath := np.appState.Logger.GetLogFilePath()

// 	// 创建xray配置，设置日志文件路径为统一日志文件
// 	xrayConfigJSON, err := xray.CreateXrayConfig(proxyPort, srv, unifiedLogPath)
// 	if err != nil {
// 		np.logAndShowError("创建xray配置失败", err)
// 		np.appState.Config.AutoProxyEnabled = false
// 		np.appState.XrayInstance = nil
// 		np.appState.UpdateProxyStatus()
// 		np.saveConfigToDB()
// 		return
// 	}

// 	// 记录配置创建成功日志
// 	if np.appState != nil {
// 		np.appState.AppendLog("DEBUG", "xray", fmt.Sprintf("xray配置已创建: %s", srv.Name))
// 	}

// 	// 创建日志回调函数，将 xray 日志转发到应用日志系统
// 	logCallback := func(level, message string) {
// 		if np.appState != nil {
// 			np.appState.AppendLog(level, "xray", message)
// 		}
// 	}

// 	// 创建xray实例，并设置日志回调
// 	xrayInstance, err := xray.NewXrayInstanceFromJSONWithCallback(xrayConfigJSON, logCallback)
// 	if err != nil {
// 		np.logAndShowError("创建xray实例失败", err)
// 		np.appState.Config.AutoProxyEnabled = false
// 		np.appState.XrayInstance = nil
// 		np.appState.UpdateProxyStatus()
// 		np.saveConfigToDB()
// 		return
// 	}

// 	// 启动xray实例
// 	err = xrayInstance.Start()
// 	if err != nil {
// 		np.logAndShowError("启动xray实例失败", err)
// 		np.appState.Config.AutoProxyEnabled = false
// 		np.appState.XrayInstance = nil
// 		np.appState.UpdateProxyStatus()
// 		np.saveConfigToDB()
// 		return
// 	}

// 	// 启动成功，设置端口信息
// 	xrayInstance.SetPort(proxyPort)
// 	np.appState.XrayInstance = xrayInstance
// 	np.appState.Config.AutoProxyEnabled = true
// 	np.appState.Config.AutoProxyPort = proxyPort

// 	// 记录日志（统一日志记录）
// 	if np.appState.Logger != nil {
// 		np.appState.Logger.InfoWithType(logging.LogTypeProxy, "xray-core代理已启动: %s (端口: %d)", srv.Name, proxyPort)
// 	}

// 	// 追加日志到日志面板
// 	if np.appState != nil {
// 		np.appState.AppendLog("INFO", "xray", fmt.Sprintf("xray-core代理已启动: %s (端口: %d)", srv.Name, proxyPort))
// 		np.appState.AppendLog("INFO", "xray", fmt.Sprintf("服务器信息: %s:%d, 协议: %s", srv.Addr, srv.Port, srv.ProtocolType))
// 	}

// 	np.Refresh()
// 	// 更新状态绑定（使用双向绑定，UI 会自动更新）
// 	np.appState.UpdateProxyStatus()

// 	np.appState.Window.SetTitle(fmt.Sprintf("代理已启动: %s (端口: %d)", srv.Name, proxyPort))

// 	// 保存配置到数据库
// 	np.saveConfigToDB()
// }

// StartProxyForSelected 对外暴露的"启动当前选中服务器"接口，供主界面一键按钮等复用。
// 内部直接复用现有 onStartProxyFromSelected 逻辑，避免重复实现。
func (np *NodePage) StartProxyForSelected() {
	// np.onStartProxyFromSelected()
}

// logAndShowError 记录日志并显示错误对话框（统一错误处理）
func (np *NodePage) logAndShowError(message string, err error) {
	if np.appState != nil && np.appState.Logger != nil {
		np.appState.Logger.Error("%s: %v", message, err)
	}
	if np.appState != nil && np.appState.Window != nil {
		np.appState.Window.SetTitle(fmt.Sprintf("%s: %v", message, err))
	}
}

// saveConfigToDB 保存应用配置到数据库（统一配置保存）
func (np *NodePage) saveConfigToDB() {
	if np.appState == nil || np.appState.Config == nil {
		return
	}
	cfg := np.appState.Config

	// 保存配置到数据库
	database.SetAppConfig("logLevel", cfg.LogLevel)
	database.SetAppConfig("logFile", cfg.LogFile)
	database.SetAppConfig("autoProxyEnabled", strconv.FormatBool(cfg.AutoProxyEnabled))
	database.SetAppConfig("autoProxyPort", strconv.Itoa(cfg.AutoProxyPort))
}

// onStopProxy 停止代理 - 注释功能
func (np *NodePage) onStopProxy() {
	stopped := false

	// 停止xray实例
	if np.appState.XrayInstance != nil {
		if np.appState != nil {
			np.appState.AppendLog("INFO", "xray", "正在停止xray-core代理...")
		}

		err := np.appState.XrayInstance.Stop()
		if err != nil {
			// 停止失败，记录日志并显示错误（统一错误处理）
			np.logAndShowError("停止xray代理失败", err)
			return
		}

		np.appState.XrayInstance = nil
		stopped = true

		// 记录日志（统一日志记录）
		if np.appState.Logger != nil {
			np.appState.Logger.InfoWithType(logging.LogTypeProxy, "xray-core代理已停止")
		}

		// 追加日志到日志面板
		if np.appState != nil {
			np.appState.AppendLog("INFO", "xray", "xray-core代理已停止")
		}
	}

	if stopped {
		// 停止成功
		np.appState.Config.AutoProxyEnabled = false
		np.appState.Config.AutoProxyPort = 0

		// 更新状态绑定
		np.appState.UpdateProxyStatus()

		// 保存配置到数据库
		np.saveConfigToDB()

		np.appState.Window.SetTitle("代理已停止")
	} else {
		np.appState.Window.SetTitle("代理未运行")
	}
}

// StopProxy 对外暴露的"停止代理"接口，供主界面一键按钮等复用。
// 内部直接复用现有 onStopProxy 逻辑。
func (np *NodePage) StopProxy() {
	np.onStopProxy()
}

// onTestAll 一键测延迟 - 注释功能
func (np *NodePage) onTestAll() {
	// 在goroutine中执行测速
	go func() {
		servers := np.appState.ServerManager.ListServers()
		enabledCount := 0
		for _, s := range servers {
			if s.Enabled {
				enabledCount++
			}
		}

		// 记录开始测速日志
		if np.appState != nil {
			np.appState.AppendLog("INFO", "ping", fmt.Sprintf("开始一键测速，共 %d 个启用的服务器", enabledCount))
		} 

		results := np.appState.PingManager.TestAllServersDelay()

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
				if np.appState != nil {
					np.appState.AppendLog("INFO", "ping", fmt.Sprintf("服务器 %s (%s:%d) 测速完成: %d ms", srv.Name, srv.Addr, srv.Port, delay))
				}
			} else {
				failCount++
				if np.appState != nil {
					np.appState.AppendLog("ERROR", "ping", fmt.Sprintf("服务器 %s (%s:%d) 测速失败", srv.Name, srv.Addr, srv.Port))
				}
			}
		}

		// 记录完成日志
		if np.appState != nil {
			np.appState.AppendLog("INFO", "ping", fmt.Sprintf("一键测速完成: 成功 %d 个，失败 %d 个，共测试 %d 个服务器", successCount, failCount, len(results)))
		}

		// 更新UI（需要在主线程中执行）
		fyne.Do(func() {
			np.Refresh()
			np.appState.Window.SetTitle(fmt.Sprintf("测速完成，共测试 %d 个服务器", len(results)))
		})
	}()
}

// ServerListItem 自定义服务器列表项（支持右键菜单和多列显示）
type ServerListItem struct {
	widget.BaseWidget
	id          widget.ListItemID
	panel       *NodePage
	renderObj   fyne.CanvasObject // 渲染对象
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
//   - panel: NodePage实例
func NewServerListItem(panel *NodePage) *ServerListItem {
	item := &ServerListItem{
		panel:       panel,
		isSelected:  false,
		isConnected: false,
	}

	// 创建标签组件
	item.regionLabel = widget.NewLabel("")
	item.regionLabel.Wrapping = fyne.TextTruncate
	item.regionLabel.Alignment = fyne.TextAlignCenter

	item.nameLabel = widget.NewLabel("")
	item.nameLabel.Wrapping = fyne.TextTruncate
	item.nameLabel.TextStyle = fyne.TextStyle{Bold: true}

	item.delayLabel = widget.NewLabel("")
	item.delayLabel.Alignment = fyne.TextAlignTrailing

	// 使用 setupLayout 创建渲染对象（参考 SubscriptionCard 的设计）
	item.renderObj = item.setupLayout()
	item.ExtendBaseWidget(item)
	return item
}

// setupLayout 设置列表项布局（参考 SubscriptionCard 的设计）
func (s *ServerListItem) setupLayout() fyne.CanvasObject {
	// 创建背景（使用输入背景色，与列表项区分）
	bg := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	bg.CornerRadius = 4 // 较小的圆角，适合列表项

	// 使用 GridWithColumns 自动分配列宽：地区（固定比例）+ 名称（自适应）+ 延迟（固定比例）
	// 减少 padding，使用最小间距
	content := container.NewGridWithColumns(3,
		s.regionLabel, // 地区列（移除 padding，使用最小间距）
		s.nameLabel,   // 名称列
		s.delayLabel,  // 延迟列
	)

	// 使用 Stack 布局：背景 + 内容
	// 移除 padding，删除列表项之间的间距
	return container.NewStack(bg, content)
}

// MinSize 返回列表项的最小尺寸（设置行高为52px，符合UI改进建议：48-56px）
func (s *ServerListItem) MinSize() fyne.Size {
	return fyne.NewSize(0, 52)
}

// CreateRenderer 创建渲染器（参考 SubscriptionCard）
func (s *ServerListItem) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(s.renderObj)
}

// TappedSecondary 处理右键点击事件 - 注释功能
// func (s *ServerListItem) TappedSecondary(pe *fyne.PointEvent) {
// 	if s.panel == nil {
// 		return
// 	}
// 	s.panel.onRightClick(s.id, pe)
// }

// Update  更新服务器列表项的信息
func (s *ServerListItem) Update(server database.Node) {
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
		// 符合 md 设计：< 100ms绿色(🟢)，100-200ms黄色(🟡)，> 200ms红色(🔴)
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

// showQuickMenu 显示快速操作菜单 - 注释功能
func (s *ServerListItem) showQuickMenu(server database.Node) {
	if s.panel == nil || s.panel.appState == nil || s.panel.appState.Window == nil {
		return
	}

	// 创建快速操作菜单
	menu := fyne.NewMenu("",
		fyne.NewMenuItem("连接", func() {
			if s.panel != nil {
				// s.panel.onStartProxy(s.id)
			}
		}),
		fyne.NewMenuItem("测速", func() {
			if s.panel != nil {
				// s.panel.onTestSpeed(s.id)
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
