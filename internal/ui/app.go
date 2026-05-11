package ui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"myproxy.com/p/internal/database"
	"myproxy.com/p/internal/logging"
	"myproxy.com/p/internal/service"
	"myproxy.com/p/internal/store"
	"myproxy.com/p/internal/subscription"
	"myproxy.com/p/internal/utils"
	"myproxy.com/p/internal/xray"
)

type AppState struct {
	initialized         bool
	Ping                *utils.Ping
	Logger              *logging.Logger
	SafeLogger          *logging.SafeLogger
	App                 fyne.App
	Window              fyne.Window
	MainWindow          *MainWindow
	TrayManager         *TrayManager
	Store               *store.Store
	ServerService       *service.ServerService
	ConfigService       *service.ConfigService
	ProxyService        *service.ProxyService
	SubscriptionService *service.SubscriptionService
	XrayControlService  *service.XrayControlService
	AccessRecordService *service.AccessRecordService
	DiagnosticsService  *service.DiagnosticsService
	XrayInstance        *xray.XrayInstance
	LogsPanel           *LogsPanel // 日志面板，仅设置页使用；OnLogLine 分发到此
	ProxyStatusBinding  binding.String
	PortBinding         binding.String
	ServerNameBinding   binding.String
	LogCallback         func(level, logType, message string)
	// OnLogLine 统一日志入口：收到完整日志行时调用，用于分发到展示和访问记录。
	// 由 MainWindow 设置，供 Logger 的 panelCallback 和文件读取使用。
	OnLogLine func(logLine string)

	windowSizeSaveMu    sync.Mutex
	windowSizeSaveTimer *time.Timer
}

func NewAppState() *AppState {
	subscriptionManager := subscription.NewSubscriptionManager()
	dataStore := store.NewStore(subscriptionManager)
	serverService := service.NewServerService(dataStore)
	configService := service.NewConfigService(dataStore)
	subscriptionService := service.NewSubscriptionService(dataStore, subscriptionManager)
	pingUtil := utils.NewPing()

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
		ProxyService:        service.NewProxyService(nil, configService),
		XrayControlService:  service.NewXrayControlService(dataStore, configService, nil, nil),
		AccessRecordService: service.NewAccessRecordService(dataStore),
		DiagnosticsService:  service.NewDiagnosticsService(configService, dataStore),
	}

	// LogCallback 保留用于兼容，但展示已改为通过 OnLogLine 统一分发
	appState.LogCallback = nil

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
	a.refreshTrayProxyMenu()
}

// refreshTrayProxyMenu 刷新托盘代理/模式菜单，使托盘状态与 AppState（Store/ConfigService）一致。
func (a *AppState) refreshTrayProxyMenu() {
	if a.TrayManager != nil {
		a.TrayManager.RefreshProxyModeMenu()
	}
}

func (a *AppState) InitApp() error {
	a.App = app.NewWithID("com.myproxy.socks5")
	// 应用主题（从配置加载）
	a.ApplyTheme()

	appIcon := createAppIcon(a)
	if appIcon != nil {
		a.App.SetIcon(appIcon)
		a.SafeLogger.Info("应用图标已设置（包括 Dock 图标）")
	} else {
		a.SafeLogger.Warn("应用图标创建失败")
	}

	a.Window = a.App.NewWindow("myproxy")

	// 必须先加载数据库中的 app_config（含 windowSize），再按配置 Resize，否则会误用默认尺寸并在后续 SetContent 时写回库覆盖用户值。
	if a.Store != nil {
		a.Store.LoadAll()
	}

	defaultSize := fyne.NewSize(420, 520)
	a.Window.Resize(a.LoadWindowSize(defaultSize))

	if a.ConfigService != nil {
		_ = a.ConfigService.SaveDefaultDirectRoutes()
	}

	a.updateStatusBindings()

	return nil
}

func (a *AppState) InitLogger() error {
	logCallback := func(level, logType, message, logLine string) {
		if a.OnLogLine != nil {
			a.OnLogLine(logLine)
		}
	}

	logFile := database.AppConfigBuiltinDefault("logFile")
	logLevel := database.AppConfigBuiltinDefault("logLevel")
	if a.Store != nil && a.Store.AppConfig != nil {
		if file, err := a.Store.AppConfig.GetWithDefault("logFile", database.AppConfigBuiltinDefault("logFile")); err == nil {
			logFile = file
		}
		if level, err := a.Store.AppConfig.GetWithDefault("logLevel", database.AppConfigBuiltinDefault("logLevel")); err == nil {
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
		// logCallback: 应用级消息（如启动成功）走 AppendLog
		// rawLogCallback: xray 劫持的原始日志 -> 落盘、展示、解析访问记录
		realLogCallback := func(level, message string) {
			a.AppendLog(level, "xray", message)
		}
		rawLogCallback := func(level, rawLine string) {
			if a.Logger != nil {
				a.Logger.WriteRawLine(rawLine)
			}
			if a.OnLogLine != nil {
				a.OnLogLine(rawLine)
			}
		}
		a.XrayControlService = service.NewXrayControlService(a.Store, a.ConfigService, realLogCallback, rawLogCallback)
	}

	return nil
}

// AppendLog 追加一条日志。由 Logger 写入文件并调用 panelCallback，统一由 OnLogLine 分发到展示和访问记录。
func (a *AppState) AppendLog(level, logType, message string) {
	level = strings.ToUpper(level)
	if strings.ToLower(logType) != "xray" {
		logType = "app"
	}
	if a.Logger != nil {
		a.Logger.Log(level, logType, message)
	}
}

// LoadWindowSize 从配置加载窗口大小，未配置时返回默认尺寸。
func (a *AppState) LoadWindowSize(defaultSize fyne.Size) fyne.Size {
	if a.ConfigService != nil {
		return a.ConfigService.GetWindowSize(defaultSize)
	}
	return defaultSize
}

// SaveWindowSize 将窗口大小保存到配置。
func (a *AppState) SaveWindowSize(size fyne.Size) {
	if a.ConfigService != nil {
		_ = a.ConfigService.SaveWindowSize(size)
	}
}

const persistWindowSizeDebounce = 400 * time.Millisecond

func (a *AppState) stopWindowSizeSaveTimer() {
	if a == nil {
		return
	}
	a.windowSizeSaveMu.Lock()
	defer a.windowSizeSaveMu.Unlock()
	if a.windowSizeSaveTimer != nil {
		a.windowSizeSaveTimer.Stop()
		a.windowSizeSaveTimer = nil
	}
}

// schedulePersistWindowSize 在窗口内容区尺寸变化后防抖写入 windowSize（Fyne 无窗口级 resize 回调）。
func (a *AppState) schedulePersistWindowSize() {
	if a == nil {
		return
	}
	a.windowSizeSaveMu.Lock()
	defer a.windowSizeSaveMu.Unlock()
	if a.windowSizeSaveTimer != nil {
		a.windowSizeSaveTimer.Stop()
	}
	a.windowSizeSaveTimer = time.AfterFunc(persistWindowSizeDebounce, func() {
		a.windowSizeSaveMu.Lock()
		a.windowSizeSaveTimer = nil
		a.windowSizeSaveMu.Unlock()
		if a.Window == nil || a.Window.Canvas() == nil {
			return
		}
		s := a.Window.Canvas().Size()
		if s.Width >= 200 && s.Height >= 200 {
			a.SaveWindowSize(s)
		}
	})
}

// wrapWithWindowSizePersistence 包裹根内容，使拖动/缩放窗口后 windowSize 能落库。
func (a *AppState) wrapWithWindowSizePersistence(inner fyne.CanvasObject) fyne.CanvasObject {
	if a == nil || inner == nil {
		return inner
	}
	return container.New(&windowSizePersistLayout{appState: a}, inner)
}

type windowSizePersistLayout struct {
	appState *AppState
}

func (l *windowSizePersistLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) > 0 && objects[0] != nil {
		objects[0].Resize(size)
		objects[0].Move(fyne.NewPos(0, 0))
	}
	if l.appState != nil && size.Width >= 200 && size.Height >= 200 {
		l.appState.schedulePersistWindowSize()
	}
}

func (l *windowSizePersistLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 || objects[0] == nil {
		return fyne.NewSize(0, 0)
	}
	return objects[0].MinSize()
}

func (a *AppState) SetupTray() {
	a.TrayManager = NewTrayManager(a)
	a.TrayManager.SetupTray()
	a.SafeLogger.Info("系统托盘设置完成")
}

func (a *AppState) SetupWindowCloseHandler() {
	if a.Window == nil {
		return
	}

	a.Window.SetCloseIntercept(func() {
		a.stopWindowSizeSaveTimer()
		if a.Window != nil && a.Window.Canvas() != nil {
			sz := a.Window.Canvas().Size()
			if sz.Width >= 200 && sz.Height >= 200 {
				a.SaveWindowSize(sz)
			}
		}
		a.Window.Hide()
	})
}

func (a *AppState) Startup() error {
	if a.initialized {
		return fmt.Errorf("应用状态: 已经初始化过")
	}

	if err := a.InitApp(); err != nil {
		return fmt.Errorf("应用状态: 初始化应用失败: %w", err)
	}

	if a.DiagnosticsService != nil {
		if err := a.DiagnosticsService.Start(); err != nil {
			return fmt.Errorf("应用状态: 启动诊断服务失败: %w", err)
		}
	}

	// 创建日志面板并设置 OnLogLine，需在 InitLogger 之前完成
	a.LogsPanel = NewLogsPanel(a)
	a.OnLogLine = func(logLine string) {
		if a.LogsPanel != nil {
			a.LogsPanel.AppendLogLine(logLine)
		}
	}

	mainWindow := NewMainWindow(a)
	a.MainWindow = mainWindow

	if err := a.InitLogger(); err != nil {
		return fmt.Errorf("应用状态: 初始化日志失败: %w", err)
	}

	// xray 日志由劫持 handler 落盘并分发，无需文件监控

	content := mainWindow.Build()
	if content != nil {
		a.Window.SetContent(a.wrapWithWindowSizePersistence(content))
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

	autoStart, err := a.Store.AppConfig.GetWithDefault("autoStartProxy", database.AppConfigBuiltinDefault("autoStartProxy"))
	if err != nil || autoStart != "true" {
		return nil
	}

	selectedServerID, err := a.Store.AppConfig.GetWithDefault("selectedServerID", database.AppConfigBuiltinDefault("selectedServerID"))
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
	a.stopWindowSizeSaveTimer()

	if a.MainWindow != nil {
		a.MainWindow.Cleanup()
		a.MainWindow = nil
	}

	if a.LogsPanel != nil {
		a.LogsPanel.Stop()
		a.LogsPanel = nil
	}

	if a.XrayInstance != nil {
		if a.XrayInstance.IsRunning() {
			_ = a.XrayInstance.Stop()
		}
		a.XrayInstance = nil
	}

	if a.AccessRecordService != nil {
		if err := a.AccessRecordService.Flush(); err != nil && a.Logger != nil {
			a.Logger.Error("刷盘访问记录失败: %v", err)
		}
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

	if a.DiagnosticsService != nil {
		a.DiagnosticsService.Stop()
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

// GetTheme 获取主题配置。
// 返回：主题变体（dark、light 或 system）
func (a *AppState) GetTheme() string {
	if a.ConfigService != nil {
		return a.ConfigService.GetTheme()
	}
	return ThemeDark
}

// SetTheme 设置主题配置并应用到 Fyne App。
// 参数：
//   - themeStr: 主题变体（dark、light 或 system）
//
// 返回：错误（如果有）
func (a *AppState) SetTheme(themeStr string) error {
	// 保存配置
	if a.ConfigService != nil {
		if err := a.ConfigService.SetTheme(themeStr); err != nil {
			return err
		}
	}

	// 应用主题到 Fyne
	if a.App != nil {
		variant := theme.VariantDark
		switch themeStr {
		case ThemeLight:
			variant = theme.VariantLight
		case ThemeSystem:
			variant = a.App.Settings().ThemeVariant()
		default:
			variant = theme.VariantDark
		}
		a.App.Settings().SetTheme(NewMonochromeTheme(variant))
	}

	// 使主窗口与托盘图标跟随主题：清除缓存并重新生成
	ClearIconCaches()
	if a.App != nil {
		if icon := createAppIcon(a); icon != nil {
			a.App.SetIcon(icon)
		}
	}
	if a.TrayManager != nil {
		a.TrayManager.RefreshTrayIcon()
	}

	return nil
}

// ApplyTheme 从配置加载并应用主题。
func (a *AppState) ApplyTheme() {
	themeStr := a.GetTheme()
	_ = a.SetTheme(themeStr)
}
