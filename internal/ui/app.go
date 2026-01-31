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

// AppState 管理应用的整体状态，包括管理器、日志和业务状态。
// 它作为应用的核心状态容器，协调各个组件之间的交互。
type AppState struct {
	// 初始化状态
	initialized bool

	Ping       *utils.Ping
	Logger     *logging.Logger
	SafeLogger *logging.SafeLogger // 安全日志包装器，处理 Logger 未初始化的情况
	App        fyne.App
	Window     fyne.Window
	MainWindow *MainWindow // 主窗口实例，用于页面导航

	// Store - 数据层核心，管理所有数据和双向绑定
	Store *store.Store

	// Service 层 - 业务逻辑层
	ServerService       *service.ServerService
	ConfigService       *service.ConfigService
	ProxyService        *service.ProxyService
	SubscriptionService *service.SubscriptionService
	XrayControlService  *service.XrayControlService

	// Xray 实例 - 用于 xray-core 代理
	XrayInstance *xray.XrayInstance

	// 绑定数据 - 用于状态面板自动更新
	ProxyStatusBinding binding.String // 代理状态文本
	PortBinding        binding.String // 端口文本
	ServerNameBinding  binding.String // 服务器名称文本

	// 日志回调函数 - 用于日志系统
	LogCallback func(level, logType, message string)
}

// NewAppState 创建并初始化新的应用状态。
// 返回：初始化后的应用状态实例
func NewAppState() *AppState {
	subscriptionManager := subscription.NewSubscriptionManager()
	dataStore := store.NewStore(subscriptionManager)
	serverService := service.NewServerService(dataStore)
	configService := service.NewConfigService(dataStore)
	subscriptionService := service.NewSubscriptionService(dataStore, subscriptionManager)
	pingUtil := utils.NewPing()

	logCallback := func(level, message string) {
	}

	appState := &AppState{
		Ping:                pingUtil,
		Logger:              nil,
		SafeLogger:          logging.NewSafeLogger(nil),
		Store:               dataStore,
		ServerService:       serverService,
		ConfigService:       configService,
		SubscriptionService: subscriptionService,
		ProxyStatusBinding:  dataStore.ProxyStatus.ProxyStatusBinding,
		PortBinding:         dataStore.ProxyStatus.PortBinding,
		ServerNameBinding:   dataStore.ProxyStatus.ServerNameBinding,
		ProxyService:        service.NewProxyService(nil),
		XrayControlService:  service.NewXrayControlService(dataStore, configService, logCallback),
	}

	appState.LogCallback = func(level, logType, message string) {
		if appState.Logger != nil {
			appState.Logger.Log(level, logType, message)
		}
	}

	return appState
}

// updateStatusBindings 更新状态绑定数据。
// 该方法通过 Store 层的 ProxyStatusStore 来更新绑定数据，符合架构规范。
// 使用双向绑定，数据更新后 UI 会自动刷新。
func (a *AppState) updateStatusBindings() {
	if a.Store == nil || a.Store.ProxyStatus == nil {
		return
	}

	// 通过 Store 层更新绑定数据（双向绑定，UI 会自动更新）
	a.Store.ProxyStatus.UpdateProxyStatus(a.XrayInstance, a.Store.Nodes)
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
	a.App = app.NewWithID("com.myproxy.socks5")

	appIcon := createAppIcon(a)
	if appIcon != nil {
		a.App.SetIcon(appIcon)
		fmt.Println("应用图标已设置（包括 Dock 图标）")
	} else {
		fmt.Println("警告: 应用图标创建失败")
	}

	themeStr := ThemeDark
	if a.Store != nil && a.Store.AppConfig != nil {
		if ts, err := a.Store.AppConfig.GetWithDefault("theme", ThemeDark); err == nil {
			themeStr = ts
		}
	}

	themeVariant := theme.VariantDark
	switch themeStr {
	case ThemeLight:
		themeVariant = theme.VariantLight
	case ThemeSystem:
		themeVariant = a.App.Settings().ThemeVariant()
	default:
		themeVariant = theme.VariantDark
	}
	a.App.Settings().SetTheme(NewMonochromeTheme(themeVariant))

	a.Window = a.App.NewWindow("myproxy")

	defaultSize := fyne.NewSize(420, 520)
	windowSize := LoadWindowSize(a, defaultSize)
	a.Window.Resize(windowSize)

	if a.Store != nil {
		a.Store.LoadAll()
	}

	if a.ConfigService != nil {
		_ = a.ConfigService.SaveDefaultDirectRoutes()
	}

	a.updateStatusBindings()

	return nil
}

// InitLogger 初始化日志记录器。
// 该方法会从 Store 读取日志配置，创建 Logger 并设置日志回调。
func (a *AppState) InitLogger() error {
	logCallback := func(level, logType, message, logLine string) {
		if a.LogCallback != nil {
			a.LogCallback(level, logType, message)
		}
	}

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
		return fmt.Errorf("应用状态: 初始化日志失败: %w", err)
	}

	a.Logger = logger
	a.SafeLogger.SetLogger(logger)

	if a.XrayControlService != nil {
		realLogCallback := func(level, message string) {
			a.AppendLog(level, "xray", message)
		}
		a.XrayControlService = service.NewXrayControlService(a.Store, a.ConfigService, realLogCallback)
	}

	return nil
}

// AppendLog 追加一条日志到日志系统（全局接口）
// 该方法可以从任何地方调用，会自动追加到日志系统
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
	// 使用 LogCallback 回调函数
	if a.LogCallback != nil {
		a.LogCallback(level, logType, message)
	}
	// 如果 Logger 已初始化，也通过 Logger 记录
	if a.Logger != nil {
		a.Logger.Log(level, logType, message)
	}
}

// LoadWindowSize 从 Store 加载窗口大小，如果不存在则返回默认值
// 参数：
//   - appState: 应用状态（包含 Store）
//   - defaultSize: 默认窗口大小
//
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
	if a.Window == nil {
		return
	}

	a.Window.SetCloseIntercept(func() {
		// 保存窗口大小到数据库（通过 Store）
		if a.Window != nil && a.Window.Canvas() != nil {
			SaveWindowSize(a, a.Window.Canvas().Size())
		}
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
	if a.initialized {
		return fmt.Errorf("应用状态: 已经初始化过")
	}

	if err := a.InitApp(); err != nil {
		return fmt.Errorf("应用状态: 初始化应用失败: %w", err)
	}

	mainWindow := NewMainWindow(a)
	a.MainWindow = mainWindow

	if err := a.InitLogger(); err != nil {
		return fmt.Errorf("应用状态: 初始化日志失败: %w", err)
	}

	content := mainWindow.Build()
	if content != nil {
		a.Window.SetContent(content)
	}

	a.SetupTray()
	a.SetupWindowCloseHandler()

	if err := a.autoLoadProxyConfig(); err != nil {
		a.AppendLog("INFO", "app", "自动加载代理配置失败: "+err.Error())
	}

	a.initialized = true
	return nil
}

// IsInitialized 检查应用状态是否已初始化。
func (a *AppState) IsInitialized() bool {
	return a.initialized
}

// Reset 重置应用状态，允许重新初始化。
func (a *AppState) Reset() {
	a.initialized = false
}

// autoLoadProxyConfig 系统启动时自动加载代理配置
// 检查是否有保存的代理状态，如果有则尝试启动代理服务
func (a *AppState) autoLoadProxyConfig() error {
	if a.Store == nil || a.Store.AppConfig == nil {
		return fmt.Errorf("应用状态: Store 未初始化")
	}

	autoStart, err := a.Store.AppConfig.GetWithDefault("autoStartProxy", "false")
	if err != nil || autoStart != "true" {
		return nil
	}

	selectedServerID, err := a.Store.AppConfig.GetWithDefault("selectedServerID", "")
	if err != nil || selectedServerID == "" {
		return fmt.Errorf("应用状态: 未找到保存的选中服务器")
	}

	if err := a.Store.Nodes.Select(selectedServerID); err != nil {
		return fmt.Errorf("应用状态: 选中服务器失败: %w", err)
	}

	a.AppendLog("INFO", "app", "正在自动启动代理服务...")

	if a.XrayControlService == nil {
		return fmt.Errorf("应用状态: XrayControlService 未初始化")
	}

	unifiedLogPath := ""
	if a.Logger != nil {
		unifiedLogPath = a.Logger.GetLogFilePath()
	}
	result := a.XrayControlService.StartProxy(a.XrayInstance, unifiedLogPath)
	if result.Error != nil {
		return fmt.Errorf("应用状态: 启动代理失败: %w", result.Error)
	}

	a.XrayInstance = result.XrayInstance

	if a.ProxyService != nil {
		a.ProxyService.UpdateXrayInstance(a.XrayInstance)
	}

	a.updateStatusBindings()

	a.AppendLog("INFO", "app", "代理服务自动启动成功")
	return nil
}

// Cleanup 清理应用资源，在应用退出时调用。
// 根据架构规范，xray 实例由 App 持有，退出时需要停止并清理。
func (a *AppState) Cleanup() {
	if a.XrayInstance != nil {
		if a.XrayInstance.IsRunning() {
			_ = a.XrayInstance.Stop()
		}
		a.XrayInstance = nil
	}

	if a.Logger != nil {
		a.Logger.Close()
		a.Logger = nil
	}

	if a.SafeLogger != nil {
		a.SafeLogger.SetLogger(nil)
	}

	if a.Store != nil {
		a.Store.Reset()
	}

	if a.ProxyService != nil {
		a.ProxyService.UpdateXrayInstance(nil)
	}
}

// Run 显示窗口并运行应用的事件循环。
// 这是应用启动的最后一步，会阻塞直到应用退出。
func (a *AppState) Run() {
	if a.Window != nil {
		a.Window.Show()
	}
	if a.App != nil {
		// 在应用退出前清理资源
		defer a.Cleanup()
		a.App.Run()
	}
}
