package service

import (
	"fmt"

	"myproxy.com/p/internal/model"
	"myproxy.com/p/internal/store"
)

// ServerService 服务器服务层，提供服务器相关的业务逻辑。
// 它封装了对 Store 的访问，提供统一的服务器操作接口。
type ServerService struct {
	store *store.Store
}

// NewServerService 创建新的服务器服务实例。
// 参数：
//   - store: Store 实例，用于数据访问
//
// 返回：初始化后的 ServerService 实例
func NewServerService(store *store.Store) *ServerService {
	return &ServerService{
		store: store,
	}
}

// GetAllServers 获取所有服务器。
// 返回：服务器列表和错误（如果有）
func (ss *ServerService) GetAllServers() ([]*model.Node, error) {
	if ss.store == nil || ss.store.Nodes == nil {
		return nil, fmt.Errorf("服务器服务: Store 未初始化")
	}
	return ss.store.Nodes.GetAll(), nil
}

// GetServerByID 根据ID获取服务器。
// 参数：
//   - id: 服务器ID
//
// 返回：服务器节点和错误（如果有）
func (ss *ServerService) GetServerByID(id string) (*model.Node, error) {
	if ss.store == nil || ss.store.Nodes == nil {
		return nil, fmt.Errorf("服务器服务: Store 未初始化")
	}
	return ss.store.Nodes.Get(id)
}

// ListServers 获取当前选中订阅的服务器列表。
// 如果未选择订阅或选择了全部订阅（ID为0），返回所有服务器。
// 否则返回指定订阅下的服务器。
// 返回：服务器列表
func (ss *ServerService) ListServers() []model.Node {
	if ss.store == nil || ss.store.Nodes == nil {
		return nil
	}

	// 获取当前选中的订阅ID（从 AppConfig 或默认值）
	selectedSubscriptionID := ss.getSelectedSubscriptionID()

	// 如果未选择订阅或选择了全部订阅（ID为0），返回所有服务器
	if selectedSubscriptionID == 0 {
		nodes := ss.store.Nodes.GetAll()
		result := make([]model.Node, len(nodes))
		for i, node := range nodes {
			result[i] = *node
		}
		return result
	}

	// 否则返回指定订阅下的服务器
	servers, err := ss.GetServersBySubscriptionID(selectedSubscriptionID)
	if err != nil {
		// 如果获取失败，返回所有服务器作为后备
		nodes := ss.store.Nodes.GetAll()
		result := make([]model.Node, len(nodes))
		for i, node := range nodes {
			result[i] = *node
		}
		return result
	}

	return servers
}

// GetServersBySubscriptionID 根据订阅ID获取服务器列表。
// 参数：
//   - subscriptionID: 订阅ID
//
// 返回：服务器列表和错误（如果有）
func (ss *ServerService) GetServersBySubscriptionID(subscriptionID int64) ([]model.Node, error) {
	if ss.store == nil || ss.store.Nodes == nil {
		return nil, fmt.Errorf("服务器服务: Store 未初始化")
	}

	nodes, err := ss.store.Nodes.GetBySubscriptionID(subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("获取订阅服务器列表失败: %w", err)
	}

	result := make([]model.Node, len(nodes))
	for i, node := range nodes {
		result[i] = *node
	}

	return result, nil
}

// UpdateServerDelay 更新服务器延迟。
// 参数：
//   - id: 服务器ID
//   - delay: 延迟值（毫秒）
//
// 返回：错误（如果有）
func (ss *ServerService) UpdateServerDelay(id string, delay int) error {
	if ss.store == nil || ss.store.Nodes == nil {
		return fmt.Errorf("服务器服务: Store 未初始化")
	}

	return ss.store.Nodes.UpdateDelay(id, delay)
}

// AddOrUpdateServer 添加或更新服务器。
// 参数：
//   - node: 服务器节点
//   - subscriptionID: 订阅ID（可选）
//
// 返回：错误（如果有）
func (ss *ServerService) AddOrUpdateServer(node model.Node, subscriptionID *int64) error {
	if ss.store == nil || ss.store.Nodes == nil {
		return fmt.Errorf("服务器服务: Store 未初始化")
	}

	return ss.store.Nodes.Add(&node)
}

// DeleteServer 删除服务器。
// 参数：
//   - id: 服务器ID
//
// 返回：错误（如果有）
func (ss *ServerService) DeleteServer(id string) error {
	if ss.store == nil || ss.store.Nodes == nil {
		return fmt.Errorf("服务器服务: Store 未初始化")
	}

	return ss.store.Nodes.Delete(id)
}

// GetSelectedSubscriptionID 获取当前选中的订阅ID。
// 返回：订阅ID，0表示全部
func (ss *ServerService) GetSelectedSubscriptionID() int64 {
	return ss.getSelectedSubscriptionID()
}

// SetSelectedSubscriptionID 设置当前选中的订阅ID。
// 参数：
//   - subscriptionID: 订阅ID，0表示全部
func (ss *ServerService) SetSelectedSubscriptionID(subscriptionID int64) {
	ss.setSelectedSubscriptionID(subscriptionID)
}

// getSelectedSubscriptionID 内部方法：获取当前选中的订阅ID。
func (ss *ServerService) getSelectedSubscriptionID() int64 {
	if ss.store == nil || ss.store.AppConfig == nil {
		return 0
	}

	// 从 AppConfig 获取选中的订阅ID
	// 注意：这里假设订阅ID存储在 AppConfig 中，key 为 "selectedSubscriptionID"
	// 如果 Store 中没有这个字段，可以考虑在 AppConfigStore 中添加专门的方法
	subIDStr, err := ss.store.AppConfig.GetWithDefault("selectedSubscriptionID", "0")
	if err != nil {
		return 0
	}

	// 解析字符串为 int64
	var subID int64
	if _, err := fmt.Sscanf(subIDStr, "%d", &subID); err != nil {
		return 0
	}

	return subID
}

// setSelectedSubscriptionID 内部方法：设置当前选中的订阅ID。
func (ss *ServerService) setSelectedSubscriptionID(subscriptionID int64) {
	if ss.store == nil || ss.store.AppConfig == nil {
		return
	}

	// 保存到 AppConfig
	subIDStr := fmt.Sprintf("%d", subscriptionID)
	_ = ss.store.AppConfig.Set("selectedSubscriptionID", subIDStr)
}
