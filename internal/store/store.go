package store

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"myproxy.com/p/internal/database"
	"myproxy.com/p/internal/model"
	"myproxy.com/p/internal/subscription"
)

// Store 是数据层的核心，管理所有应用数据和双向绑定。
// 它封装了所有数据库操作，并提供统一的数据访问接口。
type Store struct {
	// 初始化状态
	initialized bool

	// 节点数据管理
	Nodes *NodesStore

	// 订阅数据管理
	Subscriptions *SubscriptionsStore

	// 布局配置管理
	Layout *LayoutStore

	// 应用配置管理
	AppConfig *AppConfigStore

	// 代理状态管理
	ProxyStatus *ProxyStatusStore
}

// NewStore 创建新的 Store 实例并初始化所有子 Store。
// 注意：不会自动加载数据，需要在 Fyne 应用初始化后调用 LoadAll()。
// 参数：
//   - subscriptionManager: 订阅管理器，用于订阅更新操作（可为 nil，但订阅更新功能将不可用）
//
// 返回：初始化后的 Store 实例
func NewStore(subscriptionManager *subscription.SubscriptionManager) *Store {
	s := &Store{
		Nodes:         NewNodesStore(),
		Subscriptions: NewSubscriptionsStore(subscriptionManager),
		Layout:        NewLayoutStore(),
		AppConfig:     NewAppConfigStore(),
		ProxyStatus:   NewProxyStatusStore(),
	}
	// 设置 SubscriptionsStore 的父 Store 引用
	s.Subscriptions.setParentStore(s)
	return s
}

// LoadAll 从数据库加载所有数据到 Store。
// 该方法会在 Store 初始化时自动调用，也可以在需要时手动调用以刷新数据。
func (s *Store) LoadAll() {
	s.Nodes.Load()
	s.Subscriptions.Load()
	s.Layout.Load()
	s.AppConfig.Load()
	s.initialized = true
}

// IsInitialized 检查 Store 是否已初始化。
func (s *Store) IsInitialized() bool {
	return s.initialized
}

// Reset 重置 Store 状态，允许重新初始化。
func (s *Store) Reset() {
	s.initialized = false
}

// NodesStore 管理节点（服务器）数据，包括列表绑定和所有数据库操作。
type NodesStore struct {
	// 读写锁，保护并发访问
	mu sync.RWMutex

	// 节点列表（内存缓存）
	nodes []*model.Node

	// 双向绑定：节点列表绑定，UI 可以通过此绑定自动更新
	NodesBinding binding.UntypedList

	// 选中节点 ID
	selectedServerID string
}

// NewNodesStore 创建新的 NodesStore 实例。
func NewNodesStore() *NodesStore {
	return &NodesStore{
		nodes:        make([]*model.Node, 0),
		NodesBinding: binding.NewUntypedList(),
	}
}

// Load 从数据库加载所有节点到 Store。
func (ns *NodesStore) Load() error {
	nodes, err := database.GetAllServers()
	if err != nil {
		ns.mu.Lock()
		ns.nodes = []*model.Node{}
		ns.mu.Unlock()
		ns.updateBinding()
		return fmt.Errorf("节点存储: 加载节点列表失败: %w", err)
	}

	// 转换为指针切片
	ns.mu.Lock()
	ns.nodes = make([]*model.Node, len(nodes))
	for i := range nodes {
		ns.nodes[i] = &nodes[i]
	}
	ns.mu.Unlock()

	ns.updateBinding()
	return nil
}

// updateBinding 更新节点列表绑定数据，触发 UI 自动刷新。
func (ns *NodesStore) updateBinding() {
	ns.mu.RLock()
	items := make([]any, len(ns.nodes))
	for i, node := range ns.nodes {
		items[i] = node
	}
	ns.mu.RUnlock()
	_ = ns.NodesBinding.Set(items)
}

// GetAll 返回所有节点列表（只读）。
func (ns *NodesStore) GetAll() []*model.Node {
	ns.mu.RLock()
	defer ns.mu.RUnlock()
	result := make([]*model.Node, len(ns.nodes))
	copy(result, ns.nodes)
	return result
}

// Get 根据 ID 获取节点。
func (ns *NodesStore) Get(id string) (*model.Node, error) {
	ns.mu.RLock()
	defer ns.mu.RUnlock()
	for _, node := range ns.nodes {
		if node.ID == id {
			return node, nil
		}
	}
	return nil, fmt.Errorf("节点存储: 节点不存在: %s", id)
}

// GetSelected 返回当前选中的节点。
func (ns *NodesStore) GetSelected() *model.Node {
	ns.mu.RLock()
	defer ns.mu.RUnlock()
	if ns.selectedServerID == "" {
		return nil
	}
	node, _ := ns.Get(ns.selectedServerID)
	return node
}

// GetSelectedID 返回当前选中的节点 ID。
func (ns *NodesStore) GetSelectedID() string {
	ns.mu.RLock()
	defer ns.mu.RUnlock()
	return ns.selectedServerID
}

// Select 选中指定节点（更新数据库并刷新 Store）。
func (ns *NodesStore) Select(id string) error {
	// 先更新数据库
	if err := database.SelectServer(id); err != nil {
		return fmt.Errorf("节点存储: 选中节点失败: %w", err)
	}

	// 更新本地选中状态
	ns.mu.Lock()
	ns.selectedServerID = id
	ns.mu.Unlock()

	// 重新加载以更新选中状态（会更新所有节点的Selected字段）
	return ns.Load()
}

// UpdateDelay 更新节点的延迟值。
func (ns *NodesStore) UpdateDelay(id string, delay int) error {
	if err := database.UpdateServerDelay(id, delay); err != nil {
		return fmt.Errorf("节点存储: 更新节点延迟失败: %w", err)
	}
	return ns.Load()
}

// Delete 删除节点。
func (ns *NodesStore) Delete(id string) error {
	if err := database.DeleteServer(id); err != nil {
		return fmt.Errorf("节点存储: 删除节点失败: %w", err)
	}
	return ns.Load()
}

// Add 添加节点。
func (ns *NodesStore) Add(node *model.Node) error {
	if err := database.AddOrUpdateServer(*node, nil); err != nil {
		return fmt.Errorf("节点存储: 添加节点失败: %w", err)
	}
	return ns.Load()
}

// Update 更新节点。
func (ns *NodesStore) Update(node *model.Node) error {
	if err := database.AddOrUpdateServer(*node, nil); err != nil {
		return fmt.Errorf("节点存储: 更新节点失败: %w", err)
	}
	return ns.Load()
}

// GetBySubscriptionID 根据订阅ID获取节点列表。
func (ns *NodesStore) GetBySubscriptionID(subscriptionID int64) ([]*model.Node, error) {
	nodes, err := database.GetServersBySubscriptionID(subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("节点存储: 获取订阅节点失败: %w", err)
	}

	// 转换为指针切片
	result := make([]*model.Node, len(nodes))
	for i := range nodes {
		result[i] = &nodes[i]
	}

	return result, nil
}

// SubscriptionsStore 管理订阅数据，包括列表绑定和所有数据库操作。
type SubscriptionsStore struct {
	// 读写锁，保护并发访问
	mu sync.RWMutex

	// 订阅列表（内存缓存）
	subscriptions []*database.Subscription

	// 双向绑定：订阅列表绑定，UI 可以通过此绑定自动更新
	SubscriptionsBinding binding.UntypedList

	// 订阅标签绑定（用于状态面板显示）
	LabelsBinding binding.StringList

	// 订阅管理器引用，用于订阅更新操作
	subscriptionManager *subscription.SubscriptionManager

	// 父 Store 引用，用于在订阅更新后同时刷新节点数据
	parentStore *Store
}

// NewSubscriptionsStore 创建新的 SubscriptionsStore 实例。
// 参数：
//   - subscriptionManager: 订阅管理器，用于订阅更新操作（可为 nil）
func NewSubscriptionsStore(subscriptionManager *subscription.SubscriptionManager) *SubscriptionsStore {
	return &SubscriptionsStore{
		subscriptions:        make([]*database.Subscription, 0),
		SubscriptionsBinding: binding.NewUntypedList(),
		LabelsBinding:        binding.NewStringList(),
		subscriptionManager:  subscriptionManager,
		parentStore:          nil, // 将在 Store 创建后设置
	}
}

// setParentStore 设置父 Store 引用（内部方法，由 Store 调用）
func (ss *SubscriptionsStore) setParentStore(parent *Store) {
	ss.parentStore = parent
}

// SetSubscriptionManager 设置订阅管理器引用（用于延迟设置）
func (ss *SubscriptionsStore) SetSubscriptionManager(subscriptionManager *subscription.SubscriptionManager) {
	ss.subscriptionManager = subscriptionManager
}

// Load 从数据库加载所有订阅到 Store。
func (ss *SubscriptionsStore) Load() error {
	subscriptions, err := database.GetAllSubscriptions()
	if err != nil {
		ss.mu.Lock()
		ss.subscriptions = []*database.Subscription{}
		ss.mu.Unlock()
		ss.updateBinding()
		return fmt.Errorf("订阅存储: 加载订阅列表失败: %w", err)
	}

	ss.mu.Lock()
	ss.subscriptions = subscriptions
	ss.mu.Unlock()
	ss.updateBinding()
	return nil
}

// updateBinding 更新订阅列表绑定数据，触发 UI 自动刷新。
func (ss *SubscriptionsStore) updateBinding() {
	ss.mu.RLock()
	// 更新订阅列表绑定
	items := make([]any, len(ss.subscriptions))
	for i, sub := range ss.subscriptions {
		items[i] = sub
	}
	// 更新标签绑定
	labels := make([]string, 0, len(ss.subscriptions))
	for _, sub := range ss.subscriptions {
		if sub.Label != "" {
			labels = append(labels, sub.Label)
		}
	}
	ss.mu.RUnlock()
	_ = ss.SubscriptionsBinding.Set(items)
	_ = ss.LabelsBinding.Set(labels)
}

// GetAll 返回所有订阅列表（只读）。
func (ss *SubscriptionsStore) GetAll() []*database.Subscription {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	result := make([]*database.Subscription, len(ss.subscriptions))
	copy(result, ss.subscriptions)
	return result
}

// GetAll 返回所有订阅列表（只读）。
func (ss *SubscriptionsStore) GetAllNodeCount() int {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	if ss.subscriptions == nil {
		return 0
	}
	return len(ss.subscriptions)
}

// Get 根据 ID 获取订阅。
func (ss *SubscriptionsStore) Get(id int64) (*database.Subscription, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	for _, sub := range ss.subscriptions {
		if sub.ID == id {
			return sub, nil
		}
	}
	return nil, fmt.Errorf("订阅存储: 订阅不存在: %d", id)
}

// GetByURL 根据 URL 获取订阅。
func (ss *SubscriptionsStore) GetByURL(url string) (*database.Subscription, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	for _, sub := range ss.subscriptions {
		if sub.URL == url {
			return sub, nil
		}
	}
	return nil, fmt.Errorf("订阅存储: 订阅不存在: %s", url)
}

// Add 添加新订阅。
func (ss *SubscriptionsStore) Add(url, label string) (*database.Subscription, error) {
	sub, err := database.AddOrUpdateSubscription(url, label)
	if err != nil {
		return nil, fmt.Errorf("订阅存储: 添加订阅失败: %w", err)
	}
	return sub, ss.Load()
}

// Update 更新订阅。
func (ss *SubscriptionsStore) Update(id int64, url, label string) error {
	if err := database.UpdateSubscriptionByID(id, url, label); err != nil {
		return fmt.Errorf("订阅存储: 更新订阅失败: %w", err)
	}
	return ss.Load()
}

// Delete 删除订阅。
func (ss *SubscriptionsStore) Delete(id int64) error {
	if err := database.DeleteSubscription(id); err != nil {
		return fmt.Errorf("订阅存储: 删除订阅失败: %w", err)
	}
	return ss.Load()
}

// GetServerCount 获取订阅下的服务器数量。
func (ss *SubscriptionsStore) GetServerCount(id int64) (int, error) {
	return database.GetServerCountBySubscriptionID(id)
}

// UpdateByID 根据订阅 ID 更新订阅（拉取最新内容）。
// 注意：此方法已废弃，业务逻辑已移至 SubscriptionService。
// 保留此方法仅用于向后兼容，建议使用 SubscriptionService.UpdateByID。
// 参数：
//   - id: 订阅 ID
//
// 返回：错误（如果有）
func (ss *SubscriptionsStore) UpdateByID(id int64) error {
	if ss.subscriptionManager == nil {
		return fmt.Errorf("订阅存储: 订阅管理器未初始化，无法更新订阅")
	}

	if err := ss.subscriptionManager.UpdateSubscriptionByID(id); err != nil {
		return fmt.Errorf("订阅存储: 更新订阅失败: %w", err)
	}

	if err := ss.Load(); err != nil {
		return fmt.Errorf("订阅存储: 刷新订阅数据失败: %w", err)
	}

	if ss.parentStore != nil && ss.parentStore.Nodes != nil {
		if err := ss.parentStore.Nodes.Load(); err != nil {
			return fmt.Errorf("订阅存储: 刷新节点数据失败: %w", err)
		}
	}

	return nil
}

// Fetch 从 URL 获取订阅服务器列表并保存。
// 注意：此方法已废弃，业务逻辑已移至 SubscriptionService。
// 保留此方法仅用于向后兼容，建议使用 SubscriptionService.Fetch。
// 参数：
//   - url: 订阅 URL
//   - label: 订阅标签（可选）
//
// 返回：错误（如果有）
func (ss *SubscriptionsStore) Fetch(url string, label ...string) error {
	if ss.subscriptionManager == nil {
		return fmt.Errorf("订阅存储: 订阅管理器未初始化，无法获取订阅")
	}

	_, err := ss.subscriptionManager.FetchSubscription(url, label...)
	if err != nil {
		return fmt.Errorf("订阅存储: 获取订阅失败: %w", err)
	}

	if err := ss.Load(); err != nil {
		return fmt.Errorf("订阅存储: 刷新订阅数据失败: %w", err)
	}

	if ss.parentStore != nil && ss.parentStore.Nodes != nil {
		if err := ss.parentStore.Nodes.Load(); err != nil {
			return fmt.Errorf("订阅存储: 刷新节点数据失败: %w", err)
		}
	}

	return nil
}

// LayoutStore 管理布局配置数据。
type LayoutStore struct {
	// 布局配置（内存缓存）
	config *LayoutConfig

	// 配置绑定（如果需要 UI 绑定）
	ConfigBinding binding.Untyped
}

// LayoutConfig 布局配置结构。
type LayoutConfig struct {
	SubscriptionOffset float64 `json:"subscriptionOffset"` // 订阅管理区域比例 (默认0.2 = 20%)
	ServerListOffset   float64 `json:"serverListOffset"`   // 服务器列表比例 (默认0.6667 = 66.7% of 75%)
	StatusOffset       float64 `json:"statusOffset"`       // 状态信息比例 (默认0.9375 = 93.75% of 80%, 即5% of total)
}

// DefaultLayoutConfig 返回默认的布局配置。
func DefaultLayoutConfig() *LayoutConfig {
	return &LayoutConfig{
		SubscriptionOffset: 0.2,    // 20%
		ServerListOffset:   0.6667, // 66.7% of 75% = 50% of total
		StatusOffset:       0.9375, // 93.75% of 80% = 75% of total, 剩余5%
	}
}

// NewLayoutStore 创建新的 LayoutStore 实例。
func NewLayoutStore() *LayoutStore {
	return &LayoutStore{
		config:        DefaultLayoutConfig(),
		ConfigBinding: binding.NewUntyped(),
	}
}

// Load 从数据库加载布局配置。
func (ls *LayoutStore) Load() error {
	configJSON, err := database.GetLayoutConfig("layout_config")
	if err != nil || configJSON == "" {
		// 如果没有配置，使用默认配置并保存
		ls.config = DefaultLayoutConfig()
		ls.save()
		ls.updateBinding()
		return nil
	}

	// 解析配置
	var config LayoutConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		// 解析失败，使用默认配置
		ls.config = DefaultLayoutConfig()
		ls.save()
		ls.updateBinding()
		return nil
	}

	ls.config = &config
	ls.updateBinding()
	return nil
}

// updateBinding 更新配置绑定。
func (ls *LayoutStore) updateBinding() {
	_ = ls.ConfigBinding.Set(ls.config)
}

// Get 返回当前布局配置（只读）。
func (ls *LayoutStore) Get() *LayoutConfig {
	return ls.config
}

// Save 保存布局配置到数据库。
func (ls *LayoutStore) Save(config *LayoutConfig) error {
	if config == nil {
		config = DefaultLayoutConfig()
	}
	ls.config = config
	return ls.save()
}

// save 内部方法：保存配置到数据库。
func (ls *LayoutStore) save() error {
	configJSON, err := json.Marshal(ls.config)
	if err != nil {
		return fmt.Errorf("布局存储: 序列化布局配置失败: %w", err)
	}

	if err := database.SetLayoutConfig("layout_config", string(configJSON)); err != nil {
		return fmt.Errorf("布局存储: 保存布局配置失败: %w", err)
	}

	ls.updateBinding()
	return nil
}

// AppConfigStore 管理应用配置数据。
type AppConfigStore struct {
	// 配置缓存（key-value）
	config map[string]string

	// 窗口大小
	windowSize fyne.Size
}

// NewAppConfigStore 创建新的 AppConfigStore 实例。
func NewAppConfigStore() *AppConfigStore {
	return &AppConfigStore{
		config: make(map[string]string),
	}
}

// Load 从数据库加载应用配置。
func (acs *AppConfigStore) Load() error {
	// 加载窗口大小
	defaultSize := fyne.NewSize(420, 520)
	sizeStr, err := database.GetAppConfig("windowSize")
	if err != nil || sizeStr == "" {
		acs.windowSize = defaultSize
	} else {
		// 解析格式：width,height
		parts := splitSizeString(sizeStr)
		if len(parts) == 2 {
			width, err1 := strconv.ParseFloat(parts[0], 32)
			height, err2 := strconv.ParseFloat(parts[1], 32)
			if err1 == nil && err2 == nil {
				acs.windowSize = fyne.NewSize(float32(width), float32(height))
			} else {
				acs.windowSize = defaultSize
			}
		} else {
			acs.windowSize = defaultSize
		}
	}
	return nil
}

// GetWindowSize 返回窗口大小。
func (acs *AppConfigStore) GetWindowSize(defaultSize fyne.Size) fyne.Size {
	if acs.windowSize.Width == 0 && acs.windowSize.Height == 0 {
		return defaultSize
	}
	return acs.windowSize
}

// SaveWindowSize 保存窗口大小到数据库。
func (acs *AppConfigStore) SaveWindowSize(size fyne.Size) error {
	acs.windowSize = size
	sizeStr := fmt.Sprintf("%.0f,%.0f", float64(size.Width), float64(size.Height))
	if err := database.SetAppConfig("windowSize", sizeStr); err != nil {
		return fmt.Errorf("应用配置存储: 保存窗口大小失败: %w", err)
	}
	return nil
}

// Get 获取配置值。
func (acs *AppConfigStore) Get(key string) (string, error) {
	return database.GetAppConfig(key)
}

// GetWithDefault 获取配置值，如果不存在则返回默认值。
func (acs *AppConfigStore) GetWithDefault(key, defaultValue string) (string, error) {
	return database.GetAppConfigWithDefault(key, defaultValue)
}

// Set 设置配置值。
func (acs *AppConfigStore) Set(key, value string) error {
	if err := database.SetAppConfig(key, value); err != nil {
		return fmt.Errorf("应用配置存储: 保存配置失败: %w", err)
	}
	acs.config[key] = value
	return nil
}

// splitSizeString 辅助函数：分割窗口大小字符串。
func splitSizeString(s string) []string {
	// 简单的逗号分割
	parts := make([]string, 0, 2)
	start := 0
	for i, r := range s {
		if r == ',' {
			if i > start {
				parts = append(parts, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		parts = append(parts, s[start:])
	}
	return parts
}

// ProxyStatusStore 管理代理状态数据，包括代理状态、端口和服务器名称的双向绑定。
type ProxyStatusStore struct {
	// 双向绑定：代理状态文本
	ProxyStatusBinding binding.String

	// 双向绑定：端口文本
	PortBinding binding.String

	// 双向绑定：服务器名称文本
	ServerNameBinding binding.String
}

// NewProxyStatusStore 创建新的 ProxyStatusStore 实例。
func NewProxyStatusStore() *ProxyStatusStore {
	return &ProxyStatusStore{
		ProxyStatusBinding: binding.NewString(),
		PortBinding:        binding.NewString(),
		ServerNameBinding:  binding.NewString(),
	}
}

// UpdateProxyStatus 更新代理状态绑定数据。
// 该方法会根据 XrayInstance 的运行状态和选中的节点自动更新所有绑定数据。
// 参数：
//   - xrayInstance: Xray 实例指针，用于检查运行状态和端口（可为 nil）
//   - nodesStore: 节点 Store，用于获取选中的节点（可为 nil）
func (ps *ProxyStatusStore) UpdateProxyStatus(xrayInstance interface {
	IsRunning() bool
	GetPort() int
}, nodesStore *NodesStore) {
	// 更新代理状态 - 基于实际运行的代理服务
	isRunning := false
	proxyPort := 0

	// 检查 xray 实例是否运行
	// 使用反射检查接口值的底层值是否为 nil，避免 nil 指针 panic
	if xrayInstance != nil {
		// 使用反射检查底层值是否为 nil
		v := reflect.ValueOf(xrayInstance)
		if v.Kind() == reflect.Ptr && v.IsNil() {
			// 接口值不为 nil，但底层指针为 nil，跳过
			isRunning = false
			proxyPort = 0
		} else {
			// 使用 defer recover 捕获可能的 nil 指针 panic（双重保护）
			func() {
				defer func() {
					if r := recover(); r != nil {
						// 如果调用方法时发生 panic（可能是 nil 指针），忽略错误
						isRunning = false
						proxyPort = 0
					}
				}()

				// 安全地调用方法
				if xrayInstance.IsRunning() {
					isRunning = true
					if xrayInstance.GetPort() > 0 {
						proxyPort = xrayInstance.GetPort()
					} else {
						proxyPort = 10808 // 默认端口
					}
				}
			}()
		}
	}

	if isRunning {
		// 与 UI 设计规范保持一致的文案：当前连接状态 + 已连接
		ps.ProxyStatusBinding.Set("当前连接状态: 🟢 已连接")
		if proxyPort > 0 {
			ps.PortBinding.Set(fmt.Sprintf("监听端口: %d", proxyPort))
		} else {
			ps.PortBinding.Set("监听端口: -")
		}
	} else {
		// 未连接状态文案
		ps.ProxyStatusBinding.Set("当前连接状态: ⚪ 未连接")
		ps.PortBinding.Set("监听端口: -")
	}

	// 更新当前服务器
	if nodesStore != nil {
		selectedNode := nodesStore.GetSelected()
		if selectedNode != nil {
			// 使用节点名称，格式更简洁
			ps.ServerNameBinding.Set(fmt.Sprintf("%s", selectedNode.Name))
		} else {
			ps.ServerNameBinding.Set("无")
		}
	} else {
		ps.ServerNameBinding.Set("无")
	}
}
