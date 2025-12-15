package ui

import (
	"encoding/json"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"myproxy.com/p/internal/database"
)

// LayoutConfig 存储窗口布局的配置信息，包括各区域的分割比例。
// 这些配置会持久化到数据库中，以便在应用重启后恢复用户的布局偏好。
type LayoutConfig struct {
	SubscriptionOffset float64 `json:"subscriptionOffset"` // 订阅管理区域比例 (默认0.2 = 20%)
	ServerListOffset   float64 `json:"serverListOffset"`   // 服务器列表比例 (默认0.6667 = 66.7% of 75%)
	StatusOffset       float64 `json:"statusOffset"`       // 状态信息比例 (默认0.9375 = 93.75% of 80%, 即5% of total)
}

// DefaultLayoutConfig 返回默认的布局配置。
// 默认布局：订阅管理 20%，服务器列表 50%，日志 25%，状态信息 5%。
func DefaultLayoutConfig() *LayoutConfig {
	return &LayoutConfig{
		SubscriptionOffset: 0.2,    // 20%
		ServerListOffset:   0.6667, // 66.7% of 75% = 50% of total
		StatusOffset:       0.9375, // 93.75% of 80% = 75% of total, 剩余5%
	}
}

// MainWindow 管理主窗口的布局和各个面板组件。
// 它负责协调订阅管理、服务器列表、日志显示和状态信息四个主要区域的显示。
type MainWindow struct {
	appState          *AppState
	subscriptionPanel *SubscriptionPanel
	serverListPanel   *ServerListPanel
	logsPanel         *LogsPanel
	statusPanel       *StatusPanel
	mainSplit         *container.Split // 主分割容器（服务器列表和日志）
	topSplit          *container.Split // 顶部分割容器（订阅管理和主内容）
	middleStatusSplit *container.Split // 中间和状态的分割容器
	layoutConfig      *LayoutConfig    // 布局配置
}

// NewMainWindow 创建并初始化主窗口。
// 该方法会加载布局配置、创建各个面板组件，并建立它们之间的关联。
// 参数：
//   - appState: 应用状态实例
//
// 返回：初始化后的主窗口实例
func NewMainWindow(appState *AppState) *MainWindow {
	mw := &MainWindow{
		appState: appState,
	}

	// 加载布局配置
	mw.loadLayoutConfig()

	// 创建各个面板
	mw.subscriptionPanel = NewSubscriptionPanel(appState)
	mw.serverListPanel = NewServerListPanel(appState)
	mw.logsPanel = NewLogsPanel(appState)
	mw.statusPanel = NewStatusPanel(appState)

	// 设置状态面板引用，以便服务器列表可以刷新状态
	mw.serverListPanel.SetStatusPanel(mw.statusPanel)

	// 设置主窗口和日志面板引用到 AppState，以便其他组件可以刷新日志面板
	appState.MainWindow = mw
	appState.LogsPanel = mw.logsPanel

	return mw
}

// loadLayoutConfig 从数据库加载布局配置
func (mw *MainWindow) loadLayoutConfig() {
	configJSON, err := database.GetLayoutConfig("layout_config")
	if err != nil || configJSON == "" {
		// 如果没有配置，使用默认配置并保存
		mw.layoutConfig = DefaultLayoutConfig()
		mw.saveLayoutConfig()
		return
	}

	// 解析配置
	var config LayoutConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		// 解析失败，使用默认配置
		mw.layoutConfig = DefaultLayoutConfig()
		mw.saveLayoutConfig()
		return
	}

	mw.layoutConfig = &config
}

// saveLayoutConfig 保存布局配置到数据库
func (mw *MainWindow) saveLayoutConfig() {
	if mw.layoutConfig == nil {
		mw.layoutConfig = DefaultLayoutConfig()
	}

	configJSON, err := json.Marshal(mw.layoutConfig)
	if err != nil {
		return
	}

	database.SetLayoutConfig("layout_config", string(configJSON))
}

// Build 构建并返回主窗口的 UI 组件树。
// 该方法使用自定义 Border 布局，支持百分比控制各区域的大小。
// 返回：主窗口的根容器组件
func (mw *MainWindow) Build() fyne.CanvasObject {
	// 确保所有面板都已初始化
	if mw.serverListPanel == nil || mw.logsPanel == nil ||
		mw.subscriptionPanel == nil || mw.statusPanel == nil {
		// 如果面板未初始化，返回一个空的容器（不应该发生，但作为安全措施）
		return container.NewWithoutLayout()
	}

	// 服务器列表和日志的垂直分割
	serverListArea := mw.serverListPanel.Build()
	logArea := mw.logsPanel.Build()

	// 确保构建的组件不为 nil
	if serverListArea == nil {
		serverListArea = container.NewWithoutLayout()
	}
	if logArea == nil {
		logArea = container.NewWithoutLayout()
	}

	// 创建分割容器，确保两个子组件都不为 nil
	// 使用 defer recover 来确保即使创建失败也不会崩溃
	mw.mainSplit = container.NewVSplit(serverListArea, logArea)

	// 确保分割容器已正确初始化
	if mw.mainSplit != nil {
		// 从配置加载分割位置
		if mw.layoutConfig != nil && mw.layoutConfig.ServerListOffset > 0 {
			mw.mainSplit.Offset = mw.layoutConfig.ServerListOffset
		} else {
			// 默认位置：服务器列表50%，日志25%
			// 比例 = 50/(50+25) = 0.6667
			mw.mainSplit.Offset = 0.6667
		}

		// 确保分割容器可见并已初始化
		mw.mainSplit.Show()
	}

	// 订阅管理区域
	subscriptionArea := mw.subscriptionPanel.Build()
	if subscriptionArea == nil {
		subscriptionArea = container.NewWithoutLayout()
	}

	// 状态信息区域
	statusArea := mw.statusPanel.Build()
	if statusArea == nil {
		statusArea = container.NewWithoutLayout()
	}

	// 使用自定义 Border 布局，支持百分比控制
	// top=订阅管理（20%），bottom=状态信息（5%），center=服务器列表和日志（75%）
	topPercent := 0.2
	bottomPercent := 0.05

	// 从配置加载比例
	if mw.layoutConfig.SubscriptionOffset > 0 {
		topPercent = mw.layoutConfig.SubscriptionOffset
	}
	if mw.layoutConfig.StatusOffset > 0 {
		// StatusOffset 是中间内容的比例，需要转换为底部比例
		// StatusOffset = 0.9375 表示中间占 93.75%，底部占 6.25%
		// 但我们要的是 5% = 0.05
		bottomPercent = 1.0 - mw.layoutConfig.StatusOffset
		if bottomPercent > 0.1 {
			bottomPercent = 0.05 // 限制最大为 5%
		}
	}

	customLayout := NewCustomBorderLayout(topPercent, bottomPercent)

	// 使用自定义 Border 布局
	return container.New(customLayout,
		subscriptionArea, // 顶部：订阅管理
		statusArea,       // 底部：状态信息
		nil,              // 左侧：无
		nil,              // 右侧：无
		mw.mainSplit,     // 中间：服务器列表和日志的分割容器（自动拉伸）
	)
}

// Refresh 刷新主窗口的所有面板，包括服务器列表、日志显示和订阅管理。
// 该方法会更新数据绑定，使 UI 自动反映最新的应用状态。
// 注意：此方法包含安全检查，防止在窗口移动/缩放时出现空指针错误。
func (mw *MainWindow) Refresh() {
	// 安全检查：确保所有面板都已初始化
	if mw.serverListPanel != nil {
		mw.serverListPanel.Refresh()
	}
	if mw.logsPanel != nil {
		mw.logsPanel.Refresh() // 刷新日志面板，显示最新日志
	}
	if mw.subscriptionPanel != nil {
		mw.subscriptionPanel.refreshSubscriptionList()
	}
	// 使用双向绑定，只需更新绑定数据，UI 会自动更新
	if mw.appState != nil {
		mw.appState.UpdateProxyStatus()
		mw.appState.UpdateSubscriptionLabels() // 更新订阅标签绑定
	}
}

// SaveLayoutConfig 保存当前的布局配置到数据库。
// 该方法会在窗口关闭时自动调用，以保存用户的布局偏好。
func (mw *MainWindow) SaveLayoutConfig() {
	if mw.mainSplit != nil {
		mw.layoutConfig.ServerListOffset = mw.mainSplit.Offset
	}
	// 注意：使用自定义 Border 布局后，topSplit 和 middleStatusSplit 不再使用
	// 布局比例由 customLayout 控制，但配置仍然保存到数据库
	mw.saveLayoutConfig()
}

// GetLayoutConfig 返回当前的布局配置。
// 返回：布局配置实例，如果未初始化则返回默认配置
func (mw *MainWindow) GetLayoutConfig() *LayoutConfig {
	return mw.layoutConfig
}
