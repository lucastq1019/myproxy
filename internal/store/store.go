package store

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"myproxy.com/p/internal/database"
	"myproxy.com/p/internal/model"
	"myproxy.com/p/internal/subscription"
)

type Store struct {
	initialized   bool
	Nodes         *NodesStore
	Subscriptions *SubscriptionsStore
	Layout        *LayoutStore
	AppConfig     *AppConfigStore
	ProxyStatus   *ProxyStatusStore
	AccessRecords *AccessRecordsStore
}

func NewStore(subscriptionManager *subscription.SubscriptionManager) *Store {
	s := &Store{
		Nodes:         NewNodesStore(),
		Subscriptions: NewSubscriptionsStore(subscriptionManager),
		Layout:        NewLayoutStore(),
		AppConfig:     NewAppConfigStore(),
		ProxyStatus:   NewProxyStatusStore(),
		AccessRecords: NewAccessRecordsStore(),
	}
	s.Subscriptions.setParentStore(s)
	return s
}

func (s *Store) LoadAll() {
	_ = s.Nodes.Load()
	s.Subscriptions.Load()
	s.Layout.Load()
	s.AppConfig.Load()
	_ = s.AccessRecords.Load()
	// 将当前选中的服务器 ID 同步到 AppConfig，供自动启动等逻辑使用
	if id := s.Nodes.GetSelectedID(); id != "" {
		_ = s.AppConfig.Set("selectedServerID", id)
	}
	s.initialized = true
}

func (s *Store) IsInitialized() bool {
	return s.initialized
}

func (s *Store) Reset() {
	s.initialized = false
}

type NodesStore struct {
	mu               sync.RWMutex
	nodes            []*model.Node
	NodesBinding     binding.UntypedList
	selectedServerID string
}

func NewNodesStore() *NodesStore {
	return &NodesStore{
		nodes:        make([]*model.Node, 0),
		NodesBinding: binding.NewUntypedList(),
	}
}

func (ns *NodesStore) Load() error {
	nodes, err := database.GetAllServers()
	if err != nil {
		ns.mu.Lock()
		ns.nodes = []*model.Node{}
		ns.selectedServerID = ""
		ns.mu.Unlock()
		ns.updateBinding()
		return fmt.Errorf("节点存储: 加载节点列表失败: %w", err)
	}

	ns.mu.Lock()
	ns.nodes = make([]*model.Node, len(nodes))
	for i := range nodes {
		ns.nodes[i] = &nodes[i]
	}
	// 从数据库恢复“选中”状态，使应用层与列表页一致
	ns.selectedServerID = ""
	for _, node := range ns.nodes {
		if node.Selected {
			ns.selectedServerID = node.ID
			break
		}
	}
	ns.mu.Unlock()

	ns.updateBinding()
	return nil
}

func (ns *NodesStore) updateBinding() {
	ns.mu.RLock()
	items := make([]any, len(ns.nodes))
	for i, node := range ns.nodes {
		items[i] = node
	}
	ns.mu.RUnlock()
	_ = ns.NodesBinding.Set(items)
}

func (ns *NodesStore) GetAll() []*model.Node {
	ns.mu.RLock()
	defer ns.mu.RUnlock()
	result := make([]*model.Node, len(ns.nodes))
	copy(result, ns.nodes)
	return result
}

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

func (ns *NodesStore) GetSelected() *model.Node {
	ns.mu.RLock()
	defer ns.mu.RUnlock()
	if ns.selectedServerID == "" {
		return nil
	}
	node, _ := ns.Get(ns.selectedServerID)
	return node
}

func (ns *NodesStore) GetSelectedID() string {
	ns.mu.RLock()
	defer ns.mu.RUnlock()
	return ns.selectedServerID
}

func (ns *NodesStore) Select(id string) error {
	if err := database.SelectServer(id); err != nil {
		return fmt.Errorf("节点存储: 选中节点失败: %w", err)
	}
	ns.mu.Lock()
	ns.selectedServerID = id
	ns.mu.Unlock()
	return ns.Load()
}

// SelectServer 选中指定服务器并同步到 AppConfig（应用层与列表页一致，供托盘/自动启动等使用）。
func (s *Store) SelectServer(id string) error {
	if err := s.Nodes.Select(id); err != nil {
		return err
	}
	if err := s.AppConfig.Set("selectedServerID", id); err != nil {
		return err
	}
	return s.AppConfig.Set("lastNodeSwitchAt", time.Now().Format(time.RFC3339))
}

func (ns *NodesStore) UpdateDelay(id string, delay int) error {
	if err := database.UpdateServerDelay(id, delay); err != nil {
		return fmt.Errorf("节点存储: 更新节点延迟失败: %w", err)
	}
	return ns.Load()
}

func (ns *NodesStore) Delete(id string) error {
	if err := database.DeleteServer(id); err != nil {
		return fmt.Errorf("节点存储: 删除节点失败: %w", err)
	}
	return ns.Load()
}

func (ns *NodesStore) Add(node *model.Node) error {
	if err := database.AddOrUpdateServer(*node, nil); err != nil {
		return fmt.Errorf("节点存储: 添加节点失败: %w", err)
	}
	return ns.Load()
}

func (ns *NodesStore) Update(node *model.Node) error {
	if err := database.AddOrUpdateServer(*node, nil); err != nil {
		return fmt.Errorf("节点存储: 更新节点失败: %w", err)
	}
	return ns.Load()
}

func (ns *NodesStore) GetBySubscriptionID(subscriptionID int64) ([]*model.Node, error) {
	nodes, err := database.GetServersBySubscriptionID(subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("节点存储: 获取订阅节点失败: %w", err)
	}
	result := make([]*model.Node, len(nodes))
	for i := range nodes {
		result[i] = &nodes[i]
	}

	return result, nil
}

type SubscriptionsStore struct {
	mu                   sync.RWMutex
	subscriptions        []*model.Subscription
	SubscriptionsBinding binding.UntypedList
	LabelsBinding        binding.StringList
	subscriptionManager  *subscription.SubscriptionManager
	parentStore          *Store
}

func NewSubscriptionsStore(subscriptionManager *subscription.SubscriptionManager) *SubscriptionsStore {
	return &SubscriptionsStore{
		subscriptions:        make([]*database.Subscription, 0),
		SubscriptionsBinding: binding.NewUntypedList(),
		LabelsBinding:        binding.NewStringList(),
		subscriptionManager:  subscriptionManager,
	}
}

func (ss *SubscriptionsStore) setParentStore(parent *Store) {
	ss.parentStore = parent
}

func (ss *SubscriptionsStore) SetSubscriptionManager(subscriptionManager *subscription.SubscriptionManager) {
	ss.subscriptionManager = subscriptionManager
}

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

func (ss *SubscriptionsStore) updateBinding() {
	ss.mu.RLock()
	items := make([]any, len(ss.subscriptions))
	for i, sub := range ss.subscriptions {
		items[i] = sub
	}
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

func (ss *SubscriptionsStore) GetAll() []*database.Subscription {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	result := make([]*database.Subscription, len(ss.subscriptions))
	copy(result, ss.subscriptions)
	return result
}

func (ss *SubscriptionsStore) GetSubscriptionCount() int {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	if ss.subscriptions == nil {
		return 0
	}
	return len(ss.subscriptions)
}

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

func (ss *SubscriptionsStore) Add(url, label string) (*database.Subscription, error) {
	sub, err := database.AddOrUpdateSubscription(url, label)
	if err != nil {
		return nil, fmt.Errorf("订阅存储: 添加订阅失败: %w", err)
	}
	return sub, ss.Load()
}

func (ss *SubscriptionsStore) Update(id int64, url, label string) error {
	if err := database.UpdateSubscriptionByID(id, url, label); err != nil {
		return fmt.Errorf("订阅存储: 更新订阅失败: %w", err)
	}
	return ss.Load()
}

func (ss *SubscriptionsStore) Delete(id int64) error {
	if err := database.DeleteSubscription(id); err != nil {
		return fmt.Errorf("订阅存储: 删除订阅失败: %w", err)
	}
	return ss.Load()
}

func (ss *SubscriptionsStore) GetServerCount(id int64) (int, error) {
	return database.GetServerCountBySubscriptionID(id)
}

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
	if ss.parentStore != nil && ss.parentStore.AppConfig != nil {
		_ = ss.parentStore.AppConfig.Set("lastSubscriptionUpdateAt", time.Now().Format(time.RFC3339))
	}

	return nil
}

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
	if ss.parentStore != nil && ss.parentStore.AppConfig != nil {
		_ = ss.parentStore.AppConfig.Set("lastSubscriptionUpdateAt", time.Now().Format(time.RFC3339))
	}

	return nil
}

type LayoutStore struct {
	config        *LayoutConfig
	ConfigBinding binding.Untyped
}

type LayoutConfig struct {
	SubscriptionOffset float64 `json:"subscriptionOffset"`
	ServerListOffset   float64 `json:"serverListOffset"`
	StatusOffset       float64 `json:"statusOffset"`
}

func DefaultLayoutConfig() *LayoutConfig {
	return &LayoutConfig{
		SubscriptionOffset: 0.2,
		ServerListOffset:   0.6667,
		StatusOffset:       0.9375,
	}
}

func NewLayoutStore() *LayoutStore {
	return &LayoutStore{
		config:        DefaultLayoutConfig(),
		ConfigBinding: binding.NewUntyped(),
	}
}

func (ls *LayoutStore) Load() error {
	configJSON, err := database.GetLayoutConfig("layout_config")
	if err != nil || configJSON == "" {
		ls.config = DefaultLayoutConfig()
		ls.save()
		ls.updateBinding()
		return nil
	}
	var config LayoutConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		ls.config = DefaultLayoutConfig()
		ls.save()
		ls.updateBinding()
		return nil
	}

	ls.config = &config
	ls.updateBinding()
	return nil
}

func (ls *LayoutStore) updateBinding() {
	_ = ls.ConfigBinding.Set(ls.config)
}

func (ls *LayoutStore) Get() *LayoutConfig {
	return ls.config
}

func (ls *LayoutStore) Save(config *LayoutConfig) error {
	if config == nil {
		config = DefaultLayoutConfig()
	}
	ls.config = config
	return ls.save()
}

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

type AppConfigStore struct {
	config     map[string]string
	windowSize fyne.Size
}

func NewAppConfigStore() *AppConfigStore {
	return &AppConfigStore{
		config: make(map[string]string),
	}
}

func (acs *AppConfigStore) Load() error {
	defaultSize := fyne.NewSize(420, 520)
	sizeStr, err := database.GetAppConfig("windowSize")
	if err != nil || sizeStr == "" {
		acs.windowSize = defaultSize
	} else {
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

func (acs *AppConfigStore) GetWindowSize(defaultSize fyne.Size) fyne.Size {
	if acs.windowSize.Width == 0 && acs.windowSize.Height == 0 {
		return defaultSize
	}
	return acs.windowSize
}

func (acs *AppConfigStore) SaveWindowSize(size fyne.Size) error {
	acs.windowSize = size
	sizeStr := fmt.Sprintf("%.0f,%.0f", float64(size.Width), float64(size.Height))
	if err := database.SetAppConfig("windowSize", sizeStr); err != nil {
		return fmt.Errorf("应用配置存储: 保存窗口大小失败: %w", err)
	}
	return nil
}

func (acs *AppConfigStore) Get(key string) (string, error) {
	return database.GetAppConfig(key)
}

func (acs *AppConfigStore) GetWithDefault(key, defaultValue string) (string, error) {
	return database.GetAppConfigWithDefault(key, defaultValue)
}

func (acs *AppConfigStore) Set(key, value string) error {
	if err := database.SetAppConfig(key, value); err != nil {
		return fmt.Errorf("应用配置存储: 保存配置失败: %w", err)
	}
	acs.config[key] = value
	return nil
}

func splitSizeString(s string) []string {
	return strings.Split(s, ",")
}

type ProxyStatusStore struct {
	ProxyStatusBinding binding.String
	PortBinding        binding.String
	ServerNameBinding  binding.String
}

func NewProxyStatusStore() *ProxyStatusStore {
	return &ProxyStatusStore{
		ProxyStatusBinding: binding.NewString(),
		PortBinding:        binding.NewString(),
		ServerNameBinding:  binding.NewString(),
	}
}

func (ps *ProxyStatusStore) UpdateProxyStatus(xrayInstance interface {
	IsRunning() bool
	GetPort() int
}, nodesStore *NodesStore) {
	isRunning := false
	proxyPort := 0
	if xrayInstance != nil {
		v := reflect.ValueOf(xrayInstance)
		if v.Kind() == reflect.Ptr && v.IsNil() {
			isRunning = false
			proxyPort = 0
		} else {
			func() {
				defer func() {
					if r := recover(); r != nil {
						isRunning = false
						proxyPort = 0
					}
				}()
				if xrayInstance.IsRunning() {
					isRunning = true
					if xrayInstance.GetPort() > 0 {
						proxyPort = xrayInstance.GetPort()
					} else {
						proxyPort = database.DefaultMixedInboundPort
					}
				}
			}()
		}
	}
	if isRunning {
		ps.ProxyStatusBinding.Set("当前连接状态: 🟢 已连接")
		if proxyPort > 0 {
			ps.PortBinding.Set(fmt.Sprintf("监听端口: %d", proxyPort))
		} else {
			ps.PortBinding.Set("监听端口: -")
		}
	} else {
		ps.ProxyStatusBinding.Set("当前连接状态: ⚪ 未连接")
		ps.PortBinding.Set("监听端口: -")
	}
	if nodesStore != nil {
		selectedNode := nodesStore.GetSelected()
		if selectedNode != nil {
			ps.ServerNameBinding.Set(selectedNode.Name)
		} else {
			ps.ServerNameBinding.Set("无")
		}
	} else {
		ps.ServerNameBinding.Set("无")
	}
}

// AccessRecordsStore 访问记录存储，用于流量分析。
type AccessRecordsStore struct {
	mu      sync.RWMutex
	records []model.AccessRecord
}

func NewAccessRecordsStore() *AccessRecordsStore {
	return &AccessRecordsStore{
		records: make([]model.AccessRecord, 0),
	}
}

func (ars *AccessRecordsStore) Load() error {
	records, err := database.GetAllAccessRecords()
	if err != nil {
		return fmt.Errorf("访问记录存储: 加载失败: %w", err)
	}
	ars.mu.Lock()
	ars.records = records
	ars.mu.Unlock()
	return nil
}

func (ars *AccessRecordsStore) GetAll() []model.AccessRecord {
	ars.mu.RLock()
	defer ars.mu.RUnlock()
	result := make([]model.AccessRecord, len(ars.records))
	copy(result, ars.records)
	return result
}

// RecordAccess 记录访问，address 为 host:port。
// 成功写入数据库后不调用 Load：避免每条 xray 访问日志都全表重载（长期运行会放大 SQLite 与内存压力）。
// 需要展示最新数据时由 UI 调用 Load 后再 GetAll。
func (ars *AccessRecordsStore) RecordAccess(address string, count, uploadBytes, downloadBytes int64) error {
	return database.InsertOrUpdateAccessRecord(address, count, uploadBytes, downloadBytes)
}

// RecordAccessBatch 批量记录访问，key 为 address (host:port)。
// 与 RecordAccess 相同，不在此处全表 Load；由调用方在适当时机 Load。
func (ars *AccessRecordsStore) RecordAccessBatch(addressCounts map[string]int64) error {
	return database.BatchInsertOrUpdateAccessRecords(addressCounts)
}

func (ars *AccessRecordsStore) Delete(id int64) error {
	if err := database.DeleteAccessRecord(id); err != nil {
		return err
	}
	return ars.Load()
}

func (ars *AccessRecordsStore) ClearAll() error {
	if err := database.ClearAllAccessRecords(); err != nil {
		return err
	}
	ars.mu.Lock()
	ars.records = nil
	ars.mu.Unlock()
	return nil
}
