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

type AppState struct {
	initialized bool
	Ping       *utils.Ping
	Logger     *logging.Logger
	SafeLogger *logging.SafeLogger
	App        fyne.App
	Window     fyne.Window
	MainWindow *MainWindow
	Store      *store.Store
	ServerService       *service.ServerService
	ConfigService       *service.ConfigService
	ProxyService        *service.ProxyService
	SubscriptionService *service.SubscriptionService
	XrayControlService  *service.XrayControlService
	XrayInstance        *xray.XrayInstance
	ProxyStatusBinding  binding.String
	PortBinding         binding.String
	ServerNameBinding   binding.String
	LogCallback         func(level, logType, message string)
}

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

func (a *AppState) updateStatusBindings() {
	if a.Store == nil || a.Store.ProxyStatus == nil {
		return
	}
	a.Store.ProxyStatus.UpdateProxyStatus(a.XrayInstance, a.Store.Nodes)
}

func (a *AppState) UpdateProxyStatus() {
	a.updateStatusBindings()
}

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

func (a *AppState) AppendLog(level, logType, message string) {
	level = strings.ToUpper(level)
	switch strings.ToLower(logType) {
	case "xray":
		logType = "xray"
	default:
		logType = "app"
	}
	if a.LogCallback != nil {
		a.LogCallback(level, logType, message)
	}
	if a.Logger != nil {
		a.Logger.Log(level, logType, message)
	}
}

func LoadWindowSize(appState *AppState, defaultSize fyne.Size) fyne.Size {
	if appState != nil && appState.Store != nil && appState.Store.AppConfig != nil {
		return appState.Store.AppConfig.GetWindowSize(defaultSize)
	}
	return defaultSize
}

func SaveWindowSize(appState *AppState, size fyne.Size) {
	if appState != nil && appState.Store != nil && appState.Store.AppConfig != nil {
		_ = appState.Store.AppConfig.SaveWindowSize(size)
	}
}

func (a *AppState) SetupTray() {
	trayManager := NewTrayManager(a)
	fmt.Println("开始设置系统托盘...")
	trayManager.SetupTray()
	fmt.Println("系统托盘设置完成")
}

func (a *AppState) SetupWindowCloseHandler() {
	if a.Window == nil {
		return
	}

	a.Window.SetCloseIntercept(func() {
		if a.Window != nil && a.Window.Canvas() != nil {
			SaveWindowSize(a, a.Window.Canvas().Size())
		}
		a.Window.Hide()
	})
	fmt.Println("设置窗口关闭事件")
}

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

func (a *AppState) IsInitialized() bool {
	return a.initialized
}

func (a *AppState) Reset() {
	a.initialized = false
}

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

func (a *AppState) Run() {
	if a.Window != nil {
		a.Window.Show()
	}
	if a.App != nil {
		defer a.Cleanup()
		a.App.Run()
	}
}
