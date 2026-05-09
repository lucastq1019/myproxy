package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"myproxy.com/p/internal/database"
	"myproxy.com/p/internal/logging"
	"myproxy.com/p/internal/model"
	"myproxy.com/p/internal/service"
)

// NodePage 管理服务器列表的显示和操作。
// 它支持服务器选择、延迟测试、代理启动/停止等功能，并提供右键菜单操作。
type NodePage struct {
	appState   *AppState
	list       *widget.List      // 列表组件
	scrollList *container.Scroll // 滚动容器
	content    fyne.CanvasObject // 内容容器
	listener   binding.DataListener

	// 搜索与过滤相关
	searchEntry *widget.Entry // 节点搜索输入框
	searchText  string        // 当前搜索关键字（小写）

	// UI 组件
	selectedServerLabel *widget.Label // 当前选中服务器名标签
}

// NewNodePage 创建节点管理页面
func NewNodePage(appState *AppState) *NodePage {
	np := &NodePage{
		appState: appState,
	}

	// 监听 Store 的节点绑定数据变化，自动刷新列表
	if appState != nil && appState.Store != nil && appState.Store.Nodes != nil {
		np.listener = binding.NewDataListener(func() {
			if np.list != nil {
				np.list.Refresh()
				// 数据更新后，尝试滚动到选中位置
				np.scrollToSelected()
			}
		})
		appState.Store.Nodes.NodesBinding.AddListener(np.listener)
	}

	return np
}

// Cleanup 释放页面持有的监听器，避免重复建页时旧实例被 binding 持有。
func (np *NodePage) Cleanup() {
	if np == nil || np.listener == nil || np.appState == nil || np.appState.Store == nil || np.appState.Store.Nodes == nil {
		return
	}
	np.appState.Store.Nodes.NodesBinding.RemoveListener(np.listener)
	np.listener = nil
}

// loadNodes 从 Store 加载节点（Store 已经维护了绑定，这里只是确保数据最新）
func (np *NodePage) loadNodes() {
	if np.appState != nil && np.appState.Store != nil && np.appState.Store.Nodes != nil {
		_ = np.appState.Store.Nodes.Load()
	}
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
	pad := innerPadding(np.appState)
	// 1. 返回按钮
	backBtn := widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() {
		if np.appState != nil && np.appState.MainWindow != nil {
			np.appState.MainWindow.Back()
		}
	})
	backBtn.Importance = widget.LowImportance

	// 2. 当前选中服务器名标签（在测速按钮左侧）
	np.selectedServerLabel = widget.NewLabel("")
	np.selectedServerLabel.Alignment = fyne.TextAlignLeading
	np.selectedServerLabel.TextStyle = fyne.TextStyle{Bold: true}
	np.selectedServerLabel.Truncation = fyne.TextTruncateEllipsis // 文本过长时显示省略号
	np.selectedServerLabel.Wrapping = fyne.TextTruncate           // 不换行，截断
	np.updateSelectedServerLabel()                                // 初始化标签内容

	// 3. 操作按钮组（参考 subscriptionpage 风格）
	testAllBtn := widget.NewButtonWithIcon("测速", theme.ViewRefreshIcon(), np.onTestAll)
	testAllBtn.Importance = widget.LowImportance

	subscriptionBtn := widget.NewButtonWithIcon("订阅", theme.SettingsIcon(), func() {
		if np.appState != nil && np.appState.MainWindow != nil {
			np.appState.MainWindow.ShowSubscriptionPage()
		}
	})
	subscriptionBtn.Importance = widget.LowImportance

	// 4. 头部栏布局（返回按钮 + 选中服务器标签 + 操作按钮）
	// 使用 Border 布局让 labelContainer 自动占满剩余空间
	labelContainer := newPaddedWithSize(np.selectedServerLabel, pad)
	rightButtons := container.NewHBox(testAllBtn, subscriptionBtn)
	headerBar := container.NewBorder(
		nil, nil, // 上下为空
		backBtn,        // 左侧：返回按钮
		rightButtons,   // 右侧：操作按钮组
		labelContainer, // 中间：选中服务器标签（自动占满剩余空间）
	)

	// 4. 组合头部区域（添加分隔线，移除 padding 降低高度）
	separatorColor := CurrentThemeColor(np.appState.App, theme.ColorNameSeparator)
	headerStack := container.NewVBox(
		headerBar, // 移除 padding 降低功能栏高度
		canvas.NewLine(separatorColor),
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
	np.scrollList = container.NewScroll(np.list)

	// 8. 组合布局：头部 + 搜索栏 + 表头 + 列表
	// 移除所有不必要的 padding，降低高度
	np.content = container.NewBorder(
		container.NewVBox(
			headerStack,
			searchBar,   // 移除 padding
			tableHeader, // 表头直接放置，不添加额外 padding
			canvas.NewLine(separatorColor),
		),
		nil, nil, nil,
		newPaddedWithSize(np.scrollList, pad),
	)

	return np.content
}

// Refresh 刷新节点列表的显示，使 UI 反映最新的节点数据。
func (np *NodePage) Refresh() {
	np.loadNodes()
	np.updateSelectedServerLabel() // 更新选中服务器标签
	// 绑定数据更新后会自动触发列表刷新，无需手动调用
	if np.list != nil {
		np.list.Refresh()
	}
}

// scrollToSelected 滚动到选中的节点位置
func (np *NodePage) scrollToSelected() {
	if np.list == nil || np.appState == nil || np.appState.Store == nil || np.appState.Store.Nodes == nil {
		return
	}

	// 获取选中的节点ID
	selectedID := np.appState.Store.Nodes.GetSelectedID()
	if selectedID == "" {
		return
	}

	// 在过滤后的节点列表中找到选中节点的索引
	nodes := np.getFilteredNodes()
	for i, node := range nodes {
		if node.ID == selectedID {
			// 滚动到该位置（Fyne v2 的 widget.List 支持 ScrollTo 方法）
			// 使用 widget.ListItemID 类型（即 int）
			np.list.ScrollTo(widget.ListItemID(i))
			return
		}
	}
}

// updateSelectedServerLabel 更新当前选中服务器名标签
func (np *NodePage) updateSelectedServerLabel() {
	if np.selectedServerLabel == nil {
		return
	}

	// 从 Store 获取选中的服务器
	var selectedNode *model.Node
	if np.appState != nil && np.appState.Store != nil && np.appState.Store.Nodes != nil {
		selectedNode = np.appState.Store.Nodes.GetSelected()
	}

	if selectedNode == nil {
		np.selectedServerLabel.SetText("未选中")
		np.selectedServerLabel.Importance = widget.LowImportance
		return
	}

	// 显示服务器名称
	np.selectedServerLabel.SetText(selectedNode.Name)
	np.selectedServerLabel.Importance = widget.MediumImportance
}

// getNodeCount 获取节点数量
func (np *NodePage) getNodeCount() int {
	return len(np.getFilteredNodes())
}

// getFilteredNodes 根据当前搜索关键字返回过滤后的节点列表。
// 支持按名称、地址、协议类型进行不区分大小写的匹配。
func (np *NodePage) getFilteredNodes() []*model.Node {
	// 从 Store 获取所有节点
	var allNodes []*model.Node
	if np.appState != nil && np.appState.Store != nil && np.appState.Store.Nodes != nil {
		allNodes = np.appState.Store.Nodes.GetAll()
	} else {
		allNodes = []*model.Node{}
	}

	// 如果没有搜索关键字，直接返回完整列表
	if np.searchText == "" {
		return allNodes
	}

	filtered := make([]*model.Node, 0, len(allNodes))
	for _, node := range allNodes {
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
	return NewServerListItem(np, np.appState)
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
	selectedID := ""
	if np.appState != nil && np.appState.Store != nil && np.appState.Store.Nodes != nil {
		selectedID = np.appState.Store.Nodes.GetSelectedID()
	}
	item.isConnected = (np.appState != nil && np.appState.XrayInstance != nil &&
		np.appState.XrayInstance.IsRunning() && selectedID == node.ID)

	// 使用新的Update方法更新多列信息
	item.Update(*node)
}

// onNodeSelected 节点选中事件（单击选中）
func (np *NodePage) onNodeSelected(id widget.ListItemID) {
	nodes := np.getFilteredNodes()
	if id < 0 || id >= len(nodes) {
		return
	}

	node := nodes[id]

	// 通过 Store 选中节点并同步到 AppConfig（应用层与列表页一致）
	if np.appState != nil && np.appState.Store != nil {
		if err := np.appState.Store.SelectServer(node.ID); err != nil {
			if np.appState.Logger != nil {
				np.appState.Logger.Error("选中服务器失败: %v", err)
			}
			return
		}
	}

	// 更新选中服务器标签
	np.updateSelectedServerLabel()

	// 强制刷新列表显示（确保选中状态立即更新）
	if np.list != nil {
		np.list.Refresh()
	}

	// 滚动到选中位置
	np.scrollToSelected()

	// 更新主界面的节点信息显示（使用双向绑定，只需更新绑定数据，UI 会自动更新）
	if np.appState != nil {
		// 更新绑定数据（serverNameLabel 会自动更新，因为使用了双向绑定）
		np.appState.UpdateProxyStatus()
		// 注意：不再显示延迟，已从节点信息区域移除
	}
}

// onRightClick 右键菜单 - 显示操作菜单
func (np *NodePage) onRightClick(id widget.ListItemID, ev *fyne.PointEvent) {
	nodes := np.getFilteredNodes()
	if id < 0 || id >= len(nodes) {
		return
	}

	// 先选中该节点
	np.onNodeSelected(id)

	// 创建右键菜单
	menuItems := []*fyne.MenuItem{
		fyne.NewMenuItem("连接", func() {
			// 启动代理连接
			np.onStartProxy(id)
		}),
		fyne.NewMenuItem("测速", func() {
			// 测速
			np.onTestSpeed(id)
		}),
	}

	// 如果代理正在运行，添加停止选项
	if np.appState != nil && np.appState.XrayInstance != nil && np.appState.XrayInstance.IsRunning() {
		menuItems = append(menuItems, fyne.NewMenuItemSeparator())
		menuItems = append(menuItems, fyne.NewMenuItem("停止代理", func() {
			// 停止代理
			np.onStopProxy()
		}))
	}

	menu := fyne.NewMenu("", menuItems...)

	// 显示菜单
	if np.appState != nil && np.appState.Window != nil {
		popup := widget.NewPopUpMenu(menu, np.appState.Window.Canvas())
		popup.ShowAtPosition(ev.AbsolutePosition)
	}
}

// onTestSpeed 测速
func (np *NodePage) onTestSpeed(id widget.ListItemID) {
	nodes := np.getFilteredNodes()
	if id < 0 || id >= len(nodes) {
		return
	}

	node := nodes[id]

	// 在goroutine中执行测速
	go func() {
		// 记录开始测速日志
		if np.appState != nil {
			np.appState.AppendLog("INFO", "ping", fmt.Sprintf("开始测试服务器延迟: %s (%s:%d)", node.Name, node.Addr, node.Port))
		}

		delay, err := np.appState.Ping.TestServerDelay(*node)
		if err != nil {
			// 记录失败日志
			if np.appState != nil {
				np.appState.AppendLog("ERROR", "ping", fmt.Sprintf("服务器 %s 测速失败: %v", node.Name, err))
			}
			fyne.Do(func() {
				if np.appState != nil && np.appState.Window != nil {
					dialog.ShowError(fmt.Errorf("测速失败: %w", err), np.appState.Window)
				}
			})
			return
		}

		// 通过 Store 更新服务器延迟（会自动更新数据库和绑定）
		if np.appState != nil && np.appState.Store != nil && np.appState.Store.Nodes != nil {
			if err := np.appState.Store.Nodes.UpdateDelay(node.ID, delay); err != nil {
				if np.appState != nil {
					np.appState.AppendLog("ERROR", "ping", fmt.Sprintf("更新延迟失败: %v", err))
				}
			}
		}

		// 记录成功日志
		if np.appState != nil {
			np.appState.AppendLog("INFO", "ping", fmt.Sprintf("服务器 %s 测速完成: %d ms", node.Name, delay))
		}

		// 更新UI（需要在主线程中执行）
		fyne.Do(func() {
			np.Refresh()
			// 更新状态绑定（使用双向绑定，UI 会自动更新）
			if np.appState != nil {
				np.appState.UpdateProxyStatus()
			}
			if np.appState != nil && np.appState.Window != nil {
				message := fmt.Sprintf("节点: %s\n延迟: %d ms", node.Name, delay)
				dialog.ShowInformation("测速完成", message, np.appState.Window)
			}
		})
	}()
}

// onStartProxy 启动代理（右键菜单使用）
func (np *NodePage) onStartProxy(id widget.ListItemID) {
	nodes := np.getFilteredNodes()
	if id < 0 || id >= len(nodes) {
		return
	}

	// 先选中该节点
	np.onNodeSelected(id)

	// 启动代理（使用 StartProxyForSelected 方法）
	np.StartProxyForSelected()
}

// startProxyWithServer 使用指定的服务器启动代理 - 注释功能
// func (np *NodePage) startProxyWithServer(srv *database.Node) {
// 	// 使用固定的10808端口监听本地SOCKS5
// 	proxyPort := 10808

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

// StartProxyForSelected 启动当前选中服务器的代理。
// 使用 XrayControlService 来处理代理启动逻辑
func (np *NodePage) StartProxyForSelected() {
	if np.appState == nil {
		np.logAndShowError("启动代理失败", fmt.Errorf("AppState 未初始化"))
		return
	}

	if np.appState.XrayControlService == nil {
		np.logAndShowError("启动代理失败", fmt.Errorf("XrayControlService 未初始化"))
		return
	}

	// 使用统一的日志文件路径（与应用日志使用同一个文件）
	unifiedLogPath := ""
	if np.appState.Logger != nil {
		unifiedLogPath = np.appState.Logger.GetLogFilePath()
	}

	// 调用 service 启动代理
	result := np.appState.XrayControlService.StartProxy(np.appState.XrayInstance, unifiedLogPath)

	if result.Error != nil {
		np.logAndShowError("启动代理失败", result.Error)
		np.appState.UpdateProxyStatus()
		return
	}

	// 启动成功，更新 AppState 中的 XrayInstance
	np.appState.XrayInstance = result.XrayInstance

	// 更新 ProxyService 的 xray 实例引用
	if np.appState.ProxyService != nil {
		np.appState.ProxyService.UpdateXrayInstance(result.XrayInstance)
	} else {
		// 延迟初始化 ProxyService
		np.appState.ProxyService = service.NewProxyService(result.XrayInstance, np.appState.ConfigService)
	}

	// 记录日志（统一日志记录）
	if np.appState.Logger != nil && result.XrayInstance != nil {
		selectedNode := np.appState.Store.Nodes.GetSelected()
		if selectedNode != nil {
			np.appState.Logger.InfoWithType(logging.LogTypeProxy, "xray-core代理已启动: %s (端口: %d)", selectedNode.Name, result.XrayInstance.GetPort())
		}
	}

	np.Refresh()
	// 更新状态绑定（使用双向绑定，UI 会自动更新）
	np.appState.UpdateProxyStatus()

	// 与主界面主开关按钮状态同步
	if np.appState.MainWindow != nil {
		np.appState.MainWindow.RefreshMainToggleButton()
	}

	// 显示成功对话框
	if np.appState.Window != nil && result.XrayInstance != nil {
		selectedNode := np.appState.Store.Nodes.GetSelected()
		if selectedNode != nil {
			message := fmt.Sprintf("代理已启动\n节点: %s\n端口: %d", selectedNode.Name, result.XrayInstance.GetPort())
			dialog.ShowInformation("代理启动成功", message, np.appState.Window)
		}
	}
}

// logAndShowError 记录日志并显示错误对话框（统一错误处理）
func (np *NodePage) logAndShowError(message string, err error) {
	if np.appState != nil && np.appState.Logger != nil {
		np.appState.Logger.Error("%s: %v", message, err)
	}
	if np.appState != nil && np.appState.Window != nil {
		errorMsg := fmt.Errorf("%s: %w", message, err)
		dialog.ShowError(errorMsg, np.appState.Window)
	}
}

// saveConfigToDB 保存应用配置到数据库（统一配置保存）
func (np *NodePage) saveConfigToDB() {
	// 配置已由 Store.AppConfig 管理，这里不再需要保存
	// 如果需要保存特定配置，应该通过 Store.AppConfig.Set() 方法
}

// onStopProxy 停止代理。
// 使用 XrayControlService 来处理代理停止逻辑
func (np *NodePage) onStopProxy() {
	if np.appState == nil {
		np.logAndShowError("停止代理失败", fmt.Errorf("AppState 未初始化"))
		return
	}

	if np.appState.XrayControlService == nil {
		np.logAndShowError("停止代理失败", fmt.Errorf("XrayControlService 未初始化"))
		return
	}

	// 调用 service 停止代理
	result := np.appState.XrayControlService.StopProxy(np.appState.XrayInstance)

	if result.Error != nil {
		np.logAndShowError("停止代理失败", result.Error)
		return
	}

	// 停止成功，销毁实例（生命周期 = 代理运行生命周期）
	np.appState.XrayInstance = nil

	// 记录日志（统一日志记录）
	if np.appState.Logger != nil {
		np.appState.Logger.InfoWithType(logging.LogTypeProxy, "xray-core代理已停止")
	}

	// 更新状态绑定
	np.appState.UpdateProxyStatus()

	// 与主界面主开关按钮状态同步
	if np.appState.MainWindow != nil {
		np.appState.MainWindow.RefreshMainToggleButton()
	}

	// 显示成功对话框
	if np.appState.Window != nil {
		if result.LogMessage == "代理未运行" {
			dialog.ShowInformation("提示", "代理未运行", np.appState.Window)
		} else {
			dialog.ShowInformation("代理停止成功", "代理已停止", np.appState.Window)
		}
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
		var servers []*database.Node
		if np.appState != nil && np.appState.Store != nil && np.appState.Store.Nodes != nil {
			servers = np.appState.Store.Nodes.GetAll()
		}
		enabledCount := 0
		for _, s := range servers {
			if s != nil && s.Enabled {
				enabledCount++
			}
		}

		// 记录开始测速日志
		if np.appState != nil {
			np.appState.AppendLog("INFO", "ping", fmt.Sprintf("开始一键测速，共 %d 个启用的服务器", enabledCount))
		}

		// 转换为 model.Node 列表
		serverList := make([]model.Node, 0, len(servers))
		for _, s := range servers {
			if s != nil && s.Enabled {
				serverList = append(serverList, *s)
			}
		}

		// 测试所有服务器延迟
		results := np.appState.Ping.TestAllServersDelay(serverList)

		// 统计结果并记录每个服务器的详细日志，同时更新延迟
		successCount := 0
		failCount := 0
		for _, srv := range servers {
			if srv == nil || !srv.Enabled {
				continue
			}
			delay, exists := results[srv.ID]
			if !exists {
				continue
			}
			if delay > 0 {
				successCount++
				// 通过 Store 更新服务器延迟（会自动更新数据库和绑定）
				if np.appState != nil && np.appState.Store != nil && np.appState.Store.Nodes != nil {
					if err := np.appState.Store.Nodes.UpdateDelay(srv.ID, delay); err != nil {
						if np.appState != nil {
							np.appState.AppendLog("ERROR", "ping", fmt.Sprintf("更新服务器 %s 延迟失败: %v", srv.Name, err))
						}
					}
				}
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
			if np.appState != nil && np.appState.Window != nil {
				message := fmt.Sprintf("测速完成\n成功: %d 个\n失败: %d 个\n共测试: %d 个服务器", successCount, failCount, len(results))
				dialog.ShowInformation("批量测速完成", message, np.appState.Window)
			}
		})
	}()
}

// rightAlignLayout 将单个子对象右对齐、垂直居中放置（用于延迟列）。
type rightAlignLayout struct {
	minWidth float32
}

func (r rightAlignLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) != 1 {
		return
	}
	obj := objects[0]
	min := obj.MinSize()
	x := size.Width - min.Width
	if x < 0 {
		x = 0
	}
	y := (size.Height - min.Height) / 2
	if y < 0 {
		y = 0
	}
	obj.Resize(min)
	obj.Move(fyne.NewPos(x, y))
}

func (r rightAlignLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) != 1 {
		return fyne.NewSize(0, 0)
	}
	w := r.minWidth
	if w < objects[0].MinSize().Width {
		w = objects[0].MinSize().Width
	}
	return fyne.NewSize(w, objects[0].MinSize().Height)
}

// ServerListItem 自定义服务器列表项（支持右键菜单和多列显示）
type ServerListItem struct {
	widget.BaseWidget
	id          widget.ListItemID
	panel       *NodePage
	appState    *AppState
	renderObj   fyne.CanvasObject // 渲染对象
	bgRect      *canvas.Rectangle // 背景矩形（用于动态改变颜色）
	regionLabel *widget.Label
	nameLabel   *widget.Label
	delayText   *canvas.Text   // 延迟列（按 50/150ms 阈值着色）
	statusIcon  *widget.Icon   // 在线/离线状态图标
	menuButton  *widget.Button // 右侧"..."菜单按钮
	isSelected  bool           // 是否选中
	isConnected bool           // 是否当前连接
}

// NewServerListItem 创建新的服务器列表项
// 参数：
//   - panel: NodePage实例
//   - appState: 应用状态
func NewServerListItem(panel *NodePage, appState *AppState) *ServerListItem {
	item := &ServerListItem{
		panel:       panel,
		appState:    appState,
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

	item.delayText = canvas.NewText("", CurrentThemeColor(appState.App, theme.ColorNameForeground))
	item.delayText.Alignment = fyne.TextAlignTrailing
	if appState != nil && appState.App != nil {
		item.delayText.TextSize = theme.DefaultTheme().Size(theme.SizeNameText)
	}

	// 使用 setupLayout 创建渲染对象（参考 SubscriptionCard 的设计）
	item.renderObj = item.setupLayout()
	item.ExtendBaseWidget(item)
	return item
}

// setupLayout 设置列表项布局（参考 SubscriptionCard 的设计）
func (s *ServerListItem) setupLayout() fyne.CanvasObject {
	bgColor := CurrentThemeColor(s.appState.App, theme.ColorNameInputBackground)
	s.bgRect = canvas.NewRectangle(bgColor)
	s.bgRect.CornerRadius = 4 // 较小的圆角，适合列表项

	delayCell := container.New(&rightAlignLayout{minWidth: 70}, s.delayText)
	content := container.NewGridWithColumns(3,
		s.regionLabel,
		s.nameLabel,
		delayCell,
	)

	// 使用 Stack 布局：背景 + 内容
	// 移除 padding，删除列表项之间的间距
	// 使用 Padded 确保内容区域可点击
	return container.NewStack(s.bgRect, newPaddedWithSize(content, innerPadding(s.appState)))
}

// MinSize 返回列表项的最小尺寸（设置行高为52px，符合UI改进建议：48-56px）
func (s *ServerListItem) MinSize() fyne.Size {
	return fyne.NewSize(0, 52)
}

// CreateRenderer 创建渲染器（参考 SubscriptionCard）
func (s *ServerListItem) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(s.renderObj)
}

// Tapped 处理单击事件 - 选中服务器
func (s *ServerListItem) Tapped(pe *fyne.PointEvent) {
	if s.panel == nil {
		return
	}
	s.panel.onNodeSelected(s.id)
}

// TappedSecondary 处理右键点击事件 - 显示操作菜单
func (s *ServerListItem) TappedSecondary(pe *fyne.PointEvent) {
	if s.panel == nil {
		return
	}
	s.panel.onRightClick(s.id, pe)
}

// Update  更新服务器列表项的信息
func (s *ServerListItem) Update(server model.Node) {
	fyne.Do(func() {
		// 更新选中状态
		s.isSelected = server.Selected

		// 检查是否为当前连接的节点
		if s.panel != nil && s.panel.appState != nil {
			selectedID := ""
			if s.panel.appState.Store != nil && s.panel.appState.Store.Nodes != nil {
				selectedID = s.panel.appState.Store.Nodes.GetSelectedID()
			}
			s.isConnected = (s.panel.appState.XrayInstance != nil &&
				s.panel.appState.XrayInstance.IsRunning() &&
				selectedID == server.ID)
		}

		// 仅按选中/未选中设置背景色，不单独区分连接状态
		if s.bgRect != nil {
			if s.isSelected {
				s.bgRect.FillColor = CurrentThemeColor(s.appState.App, theme.ColorNameSelection)
				s.bgRect.StrokeColor = CurrentThemeColor(s.appState.App, theme.ColorNameSeparator)
				s.bgRect.StrokeWidth = 1
			} else {
				s.bgRect.FillColor = CurrentThemeColor(s.appState.App, theme.ColorNameInputBackground)
				s.bgRect.StrokeColor = CurrentThemeColor(s.appState.App, theme.ColorNameSeparator)
				s.bgRect.StrokeWidth = 0
			}
			s.bgRect.Refresh()
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

		// 延迟 - 按 0-60ms 绿 / 60-150ms 黄 / >150ms 红 / 超时或未测速 灰 着色
		delayDisplay := "未测速"
		if server.Delay > 0 {
			delayDisplay = fmt.Sprintf("%d ms", server.Delay)
		} else if server.Delay < 0 {
			delayDisplay = "测试失败"
		}
		s.delayText.Text = delayDisplay
		s.delayText.Color = DelayColor(s.appState.App, server.Delay)
		s.delayText.Refresh()

		// 更新在线/离线状态图标
		if s.statusIcon != nil {
			if server.Delay > 0 {
				// 有延迟数据，表示在线
				s.statusIcon.SetResource(theme.ConfirmIcon())
			} else if server.Delay < 0 {
				// 延迟为负，表示测试失败
				s.statusIcon.SetResource(theme.CancelIcon())
			} else {
				// 未测速
				s.statusIcon.SetResource(theme.InfoIcon())
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
func (s *ServerListItem) showQuickMenu(server model.Node) {
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
			if s.panel != nil && s.panel.appState != nil && s.panel.appState.Window != nil {
				dialog.ShowInformation("提示", "收藏功能开发中", s.panel.appState.Window)
			}
		}),
		fyne.NewMenuItem("复制信息", func() {
			// TODO: 实现复制节点信息功能
			info := fmt.Sprintf("名称: %s\n地址: %s:%d\n协议: %s",
				server.Name, server.Addr, server.Port, server.ProtocolType)
			if s.panel != nil && s.panel.appState != nil && s.panel.appState.Window != nil {
				s.panel.appState.Window.Clipboard().SetContent(info)
				dialog.ShowInformation("提示", "节点信息已复制到剪贴板", s.panel.appState.Window)
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
