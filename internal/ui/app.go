package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"myproxy.com/p/internal/logging"
	"myproxy.com/p/internal/service"
	"myproxy.com/p/internal/store"
	"myproxy.com/p/internal/subscription"
	"myproxy.com/p/internal/utils"
	"myproxy.com/p/internal/xray"
)

// AppState 管理应用的整体状态，包括管理器、日志和 UI 组件。
// 它作为应用的核心状态容器，协调各个组件之间的交互。
type AppState struct {
	Ping *utils.Ping
	Logger      *logging.Logger
	App         fyne.App
	Window      fyne.Window

	// Store - 数据层核心，管理所有数据和双向绑定
	Store *store.Store

	// Service 层 - 业务逻辑层
	ServerService       *service.ServerService
	ConfigService       *service.ConfigService
	ProxyService        *service.ProxyService
	SubscriptionService *service.SubscriptionService

	// Xray 实例 - 用于 xray-core 代理
	XrayInstance *xray.XrayInstance

	// 绑定数据 - 用于状态面板自动更新
	ProxyStatusBinding binding.String // 代理状态文本
	PortBinding        binding.String // 端口文本
	ServerNameBinding  binding.String // 服务器名称文本

	// 主窗口引用 - 用于刷新日志面板
	MainWindow *MainWindow

	// 日志面板引用 - 用于追加日志
	LogsPanel *LogsPanel
}

// NewAppState 创建并初始化新的应用状态。
// 返回：初始化后的应用状态实例
func NewAppState() *AppState {
	// 创建绑定数据
	proxyStatusBinding := binding.NewString()
	portBinding := binding.NewString()
	serverNameBinding := binding.NewString()

	// 创建 SubscriptionManager（先创建，因为 Store 需要它）
	subscriptionManager := subscription.NewSubscriptionManager()

	// 创建 Store 实例（传入 SubscriptionManager，改进依赖注入）
	dataStore := store.NewStore(subscriptionManager)

	// 创建服务层（按依赖顺序创建）
	serverService := service.NewServerService(dataStore)
	configService := service.NewConfigService(dataStore)
	subscriptionService := service.NewSubscriptionService(dataStore, subscriptionManager)

	// 创建 Ping 工具
	pingUtil := utils.NewPing()

	appState := &AppState{
		Ping:               pingUtil,
		Logger:             nil, // Logger 将在 InitLogger 中创建
		Store:              dataStore,
		ServerService:      serverService,
		ConfigService:      configService,
		SubscriptionService: subscriptionService,
		ProxyStatusBinding: proxyStatusBinding,
		PortBinding:        portBinding,
		ServerNameBinding:  serverNameBinding,
		// ProxyService 将在 XrayInstance 创建后初始化
		ProxyService: nil,
	}

	return appState
}

// updateStatusBindings 更新状态绑定数据
func (a *AppState) updateStatusBindings() {
	// 更新代理状态 - 基于实际运行的代理服务，而不是配置标志
	isRunning := false
	proxyPort := 0

	// 检查 xray 实例是否运行（使用 IsRunning 方法检查真实运行状态）
	if a.XrayInstance != nil && a.XrayInstance.IsRunning() {
		// xray-core 代理正在运行
		isRunning = true
		// 从 xray 实例获取端口
		if a.XrayInstance.GetPort() > 0 {
			proxyPort = a.XrayInstance.GetPort()
		} else {
			proxyPort = 10080 // 默认端口
		}
	}

	if isRunning {
		// 与 UI 设计规范保持一致的文案：当前连接状态 + 已连接
		a.ProxyStatusBinding.Set("当前连接状态: 🟢 已连接")
		if proxyPort > 0 {
			a.PortBinding.Set(fmt.Sprintf("监听端口: %d", proxyPort))
		} else {
			a.PortBinding.Set("监听端口: -")
		}
	} else {
		// 未连接状态文案
		a.ProxyStatusBinding.Set("当前连接状态: ⚪ 未连接")
		a.PortBinding.Set("监听端口: -")
	}

	// 更新当前服务器（符合 UI.md 设计：🌐 节点: US - LA - 32ms）
	if a.Store != nil && a.Store.Nodes != nil {
		selectedNode := a.Store.Nodes.GetSelected()
		if selectedNode != nil {
			// 使用节点名称，格式更简洁
			a.ServerNameBinding.Set(fmt.Sprintf("🌐 节点: %s", selectedNode.Name))
		} else {
			a.ServerNameBinding.Set("🌐 节点: 无")
		}
	} else {
		a.ServerNameBinding.Set("🌐 节点: 无")
	}
}

// UpdateProxyStatus 更新代理状态并刷新 UI 绑定数据。
// 该方法会检查代理转发器的实际运行状态，并更新相关的绑定数据，
// 使状态面板能够自动反映最新的代理状态。
func (a *AppState) UpdateProxyStatus() {
	a.updateStatusBindings()
}

// InitApp 初始化 Fyne 应用和窗口。
// 该方法会创建应用实例、设置主题、创建主窗口，并加载 Store 数据。
// 注意：必须在创建 UI 组件之前调用此方法。
func (a *AppState) InitApp() error {
	// 创建 Fyne 应用
	a.App = app.NewWithID("com.myproxy.socks5")
	
	// 设置应用图标（使用自定义图标）
	// 这会同时设置 Dock 图标和窗口图标（在 macOS 上）
	appIcon := createAppIcon(a)
	if appIcon != nil {
		a.App.SetIcon(appIcon)
		fmt.Println("应用图标已设置（包括 Dock 图标）")
	} else {
		fmt.Println("警告: 应用图标创建失败")
	}
	
	// 从 Store 加载主题配置，默认使用黑色主题
	themeVariant := theme.VariantDark
	if a.Store != nil && a.Store.AppConfig != nil {
		if themeStr, err := a.Store.AppConfig.GetWithDefault("theme", "dark"); err == nil && themeStr == "light" {
			themeVariant = theme.VariantLight
		}
	}
	a.App.Settings().SetTheme(NewMonochromeTheme(themeVariant))
	
	// 创建主窗口
	a.Window = a.App.NewWindow("myproxy")
	
	// 从 Store 读取窗口大小，如果没有则使用默认值
	defaultSize := fyne.NewSize(420, 520)
	windowSize := LoadWindowSize(a, defaultSize)
	a.Window.Resize(windowSize)

	// Fyne 应用初始化后，可以加载 Store 数据（必须在 Fyne 应用初始化后）
	if a.Store != nil {
		a.Store.LoadAll()
	}
	
	// 更新状态绑定
	a.updateStatusBindings()

	return nil
}

// InitLogger 初始化日志记录器。
// 该方法会从 Store 读取日志配置，创建 Logger 并设置日志面板回调。
// 注意：必须在 MainWindow 和 LogsPanel 创建后调用此方法。
func (a *AppState) InitLogger() error {
	if a.LogsPanel == nil {
		return fmt.Errorf("LogsPanel 未初始化，无法创建 Logger")
	}

	// 创建日志回调函数，用于实时更新UI（确保日志文件写入和UI显示一致）
	logCallback := func(level, logType, message, logLine string) {
		if a.LogsPanel != nil {
			// 直接使用完整的日志行，确保格式与文件中的格式完全一致
			a.LogsPanel.AppendLogLine(logLine)
		}
	}

	// 从 Store 读取日志配置并初始化 logger（使用硬编码默认值）
	logFile := "myproxy.log"
	logLevel := "info"
	if a.Store != nil && a.Store.AppConfig != nil {
		if file, err := a.Store.AppConfig.GetWithDefault("logFile", "myproxy.log"); err == nil {
			logFile = file
		}
		if level, err := a.Store.AppConfig.GetWithDefault("logLevel", "info"); err == nil {
			logLevel = level
		}
	}

	logger, err := logging.NewLogger(logFile, logLevel == "debug", logLevel, logCallback)
	if err != nil {
		return fmt.Errorf("初始化日志失败: %w", err)
	}

	// 设置 logger 到 appState
	a.Logger = logger

	// Logger 初始化后，启动日志文件监控（用于监控 xray 日志等直接从文件写入的日志）
	if a.LogsPanel != nil {
		a.LogsPanel.StartLogFileWatcher()
	}

	return nil
}

// AppendLog 追加一条日志到日志面板（全局接口）
// 该方法可以从任何地方调用，会自动追加到日志缓冲区并更新显示
// 参数：
//   - level: 日志级别 (DEBUG, INFO, WARN, ERROR, FATAL)
//   - logType: 日志类型 (app 或 xray；其他将归并为 app)
//   - message: 日志消息
func (a *AppState) AppendLog(level, logType, message string) {
	// 规范化：级别大写，来源仅 app/xray
	level = strings.ToUpper(level)
	switch strings.ToLower(logType) {
	case "xray":
		logType = "xray"
	default:
		logType = "app"
	}
	if a.LogsPanel != nil {
		a.LogsPanel.AppendLog(level, logType, message)
	}
}

// LoadWindowSize 从 Store 加载窗口大小，如果不存在则返回默认值
// 参数：
//   - appState: 应用状态（包含 Store）
//   - defaultSize: 默认窗口大小
// 返回：窗口大小
func LoadWindowSize(appState *AppState, defaultSize fyne.Size) fyne.Size {
	if appState != nil && appState.Store != nil && appState.Store.AppConfig != nil {
		return appState.Store.AppConfig.GetWindowSize(defaultSize)
	}
	return defaultSize
}

// SaveWindowSize 保存窗口大小到 Store
// 参数：
//   - appState: 应用状态（包含 Store）
//   - size: 窗口大小
func SaveWindowSize(appState *AppState, size fyne.Size) {
	if appState != nil && appState.Store != nil && appState.Store.AppConfig != nil {
		_ = appState.Store.AppConfig.SaveWindowSize(size)
	}
}

// SetupTray 设置系统托盘
func (a *AppState) SetupTray() {
	trayManager := NewTrayManager(a)
	fmt.Println("开始设置系统托盘...")
	trayManager.SetupTray()
	fmt.Println("系统托盘设置完成")
}

// SetupWindowCloseHandler 设置窗口关闭事件处理
func (a *AppState) SetupWindowCloseHandler() {
	if a.Window == nil || a.MainWindow == nil {
		return
	}
	
	a.Window.SetCloseIntercept(func() {
		// 保存窗口大小到数据库（通过 Store）
		if a.Window != nil && a.Window.Canvas() != nil {
			SaveWindowSize(a, a.Window.Canvas().Size())
		}
		// 保存布局配置到数据库（通过 Store）
		a.MainWindow.SaveLayoutConfig()
		// 配置已由 Store 自动管理，无需手动保存
		// 隐藏窗口而不是关闭（Fyne 会自动处理 Dock 图标点击显示窗口）
		a.Window.Hide()
	})
	fmt.Println("设置窗口关闭事件")
}

// Startup 统一管理应用启动的所有初始化步骤。
// 该方法按顺序执行：初始化 Fyne 应用、创建主窗口、初始化日志、设置窗口内容、设置托盘和关闭事件。
// 注意：必须在数据库初始化后调用此方法。
func (a *AppState) Startup() error {
	// 1. 初始化 Fyne 应用和窗口，加载 Store 数据
	if err := a.InitApp(); err != nil {
		return fmt.Errorf("初始化应用失败: %w", err)
	}

	// 2. 创建主窗口（此时 LogsPanel 已创建）
	// 注意：NewMainWindow 内部已经设置了 a.MainWindow 和 a.LogsPanel
	mainWindow := NewMainWindow(a)

	// 3. 初始化 Logger（需要在 LogsPanel 创建后）
	if err := a.InitLogger(); err != nil {
		return fmt.Errorf("初始化日志失败: %w", err)
	}

	// 4. 设置窗口内容
	content := mainWindow.Build()
	if content != nil {
		a.Window.SetContent(content)
	}

	// 5. 设置系统托盘
	a.SetupTray()

	// 6. 设置窗口关闭事件
	a.SetupWindowCloseHandler()

	return nil
}

// Run 显示窗口并运行应用的事件循环。
// 这是应用启动的最后一步，会阻塞直到应用退出。
func (a *AppState) Run() {
	if a.Window != nil {
		a.Window.Show()
	}
	if a.App != nil {
		a.App.Run()
	}
}
