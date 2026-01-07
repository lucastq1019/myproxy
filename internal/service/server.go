package service

import (
	"fmt"

	"myproxy.com/p/internal/database"
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
// 返回：初始化后的 ServerService 实例
func NewServerService(store *store.Store) *ServerService {
	return &ServerService{
		store: store,
	}
}

// ListServers 获取当前选中订阅的服务器列表。
// 如果未选择订阅或选择了全部订阅（ID为0），返回所有服务器。
// 否则返回指定订阅下的服务器。
// 返回：服务器列表
func (ss *ServerService) ListServers() []database.Node {
	if ss.store == nil || ss.store.Nodes == nil {
		return nil
	}

	// 获取当前选中的订阅ID（从 AppConfig 或默认值）
	selectedSubscriptionID := ss.getSelectedSubscriptionID()

	// 如果未选择订阅或选择了全部订阅（ID为0），返回所有服务器
	if selectedSubscriptionID == 0 {
		nodes := ss.store.Nodes.GetAll()
		result := make([]database.Node, len(nodes))
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
		result := make([]database.Node, len(nodes))
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
// 返回：服务器列表和错误（如果有）
func (ss *ServerService) GetServersBySubscriptionID(subscriptionID int64) ([]database.Node, error) {
	// 从数据库获取指定订阅下的服务器
	servers, err := database.GetServersBySubscriptionID(subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("获取订阅服务器列表失败: %w", err)
	}

	return servers, nil
}

// UpdateServerDelay 更新服务器延迟。
// 参数：
//   - id: 服务器ID
//   - delay: 延迟值（毫秒）
// 返回：错误（如果有）
func (ss *ServerService) UpdateServerDelay(id string, delay int) error {
	if ss.store == nil || ss.store.Nodes == nil {
		return fmt.Errorf("Store 未初始化")
	}

	// 通过 Store 更新延迟（会自动更新数据库并刷新 Store）
	return ss.store.Nodes.UpdateDelay(id, delay)
}

// AddServer 添加服务器。
// 参数：
//   - server: 服务器节点
// 返回：错误（如果有）
func (ss *ServerService) AddServer(server database.Node) error {
	// 添加到数据库（subscription_id 为 nil，表示手动添加的服务器）
	if err := database.AddOrUpdateServer(server, nil); err != nil {
		return fmt.Errorf("添加服务器到数据库失败: %w", err)
	}

	// 刷新 Store
	if ss.store != nil && ss.store.Nodes != nil {
		return ss.store.Nodes.Load()
	}

	return nil
}

// UpdateServer 更新服务器信息。
// 参数：
//   - server: 服务器节点
// 返回：错误（如果有）
func (ss *ServerService) UpdateServer(server database.Node) error {
	// 检查服务器是否存在
	if ss.store == nil || ss.store.Nodes == nil {
		return fmt.Errorf("Store 未初始化")
	}

	_, err := ss.store.Nodes.Get(server.ID)
	if err != nil {
		return fmt.Errorf("服务器不存在: %s", server.ID)
	}

	// 更新数据库（保留原有的 subscription_id）
	// 使用 nil 作为 subscriptionID，AddOrUpdateServer 会自动保持原有的 subscription_id
	if err := database.AddOrUpdateServer(server, nil); err != nil {
		return fmt.Errorf("更新服务器到数据库失败: %w", err)
	}

	// 刷新 Store
	return ss.store.Nodes.Load()
}

// GetServer 获取服务器。
// 参数：
//   - id: 服务器ID
// 返回：服务器节点和错误（如果有）
func (ss *ServerService) GetServer(id string) (*database.Node, error) {
	if ss.store == nil || ss.store.Nodes == nil {
		return nil, fmt.Errorf("Store 未初始化")
	}

	return ss.store.Nodes.Get(id)
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


