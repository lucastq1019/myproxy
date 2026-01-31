package ui

import (
	"fmt"

	"myproxy.com/p/internal/logging"
	"myproxy.com/p/internal/service"
	"myproxy.com/p/internal/store"
	"myproxy.com/p/internal/subscription"
	"myproxy.com/p/internal/utils"
)

// ApplicationFactory 应用工厂，负责创建和初始化所有应用组件
// 集中管理初始化顺序，确保依赖关系正确

type ApplicationFactory struct {
	initialized bool
}

// NewApplicationFactory 创建新的应用工厂实例
func NewApplicationFactory() *ApplicationFactory {
	return &ApplicationFactory{
		initialized: false,
	}
}

// CreateAppState 创建并初始化应用状态
// 按正确的依赖顺序初始化所有组件
func (af *ApplicationFactory) CreateAppState() (*AppState, error) {
	if af.initialized {
		return nil, fmt.Errorf("应用工厂: 已经初始化过")
	}

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
		LogCallback: func(level, logType, message string) {
		},
	}

	af.initialized = true
	return appState, nil
}

// InitializeApplication 初始化整个应用
// 按顺序执行所有初始化步骤
func (af *ApplicationFactory) InitializeApplication(appState *AppState) error {
	if appState == nil {
		return fmt.Errorf("应用工厂: AppState 为 nil")
	}

	if err := appState.InitApp(); err != nil {
		return fmt.Errorf("应用工厂: 初始化应用失败: %w", err)
	}

	mainWindow := NewMainWindow(appState)
	appState.MainWindow = mainWindow

	if err := appState.InitLogger(); err != nil {
		return fmt.Errorf("应用工厂: 初始化日志失败: %w", err)
	}

	content := mainWindow.Build()
	if content != nil {
		appState.Window.SetContent(content)
	}

	appState.SetupTray()
	appState.SetupWindowCloseHandler()

	if err := appState.autoLoadProxyConfig(); err != nil {
		appState.AppendLog("INFO", "app", "自动加载代理配置失败: "+err.Error())
	}

	return nil
}

// IsInitialized 检查工厂是否已经初始化
func (af *ApplicationFactory) IsInitialized() bool {
	return af.initialized
}

// Reset 重置工厂状态，允许重新初始化
func (af *ApplicationFactory) Reset() {
	af.initialized = false
}
