package service

import (
	"fmt"

	"myproxy.com/p/internal/store"
	"myproxy.com/p/internal/subscription"
)

// SubscriptionService 订阅服务层，提供订阅相关的业务逻辑。
type SubscriptionService struct {
	store              *store.Store
	subscriptionManager *subscription.SubscriptionManager
}

// NewSubscriptionService 创建新的订阅服务实例。
// 参数：
//   - store: Store 实例，用于数据访问
//   - subscriptionManager: 订阅管理器，用于订阅更新操作
// 返回：初始化后的 SubscriptionService 实例
func NewSubscriptionService(store *store.Store, subscriptionManager *subscription.SubscriptionManager) *SubscriptionService {
	return &SubscriptionService{
		store:              store,
		subscriptionManager: subscriptionManager,
	}
}

// UpdateByID 根据订阅 ID 更新订阅（拉取最新内容）。
// 参数：
//   - id: 订阅 ID
// 返回：错误（如果有）
func (ss *SubscriptionService) UpdateByID(id int64) error {
	if ss.subscriptionManager == nil {
		return fmt.Errorf("订阅管理器未初始化，无法更新订阅")
	}
	if ss.store == nil || ss.store.Subscriptions == nil {
		return fmt.Errorf("Store 未初始化")
	}

	// 调用 SubscriptionManager 更新订阅（会更新数据库中的订阅和节点）
	if err := ss.subscriptionManager.UpdateSubscriptionByID(id); err != nil {
		return fmt.Errorf("更新订阅失败: %w", err)
	}

	// 更新后重新加载订阅数据
	if err := ss.store.Subscriptions.Load(); err != nil {
		return fmt.Errorf("刷新订阅数据失败: %w", err)
	}

	// 同时刷新节点数据（因为订阅更新会添加/更新节点）
	if ss.store.Nodes != nil {
		if err := ss.store.Nodes.Load(); err != nil {
			return fmt.Errorf("刷新节点数据失败: %w", err)
		}
	}

	return nil
}

// Fetch 从 URL 获取订阅服务器列表并保存。
// 参数：
//   - url: 订阅 URL
//   - label: 订阅标签（可选）
// 返回：错误（如果有）
func (ss *SubscriptionService) Fetch(url string, label ...string) error {
	if ss.subscriptionManager == nil {
		return fmt.Errorf("订阅管理器未初始化，无法获取订阅")
	}
	if ss.store == nil || ss.store.Subscriptions == nil {
		return fmt.Errorf("Store 未初始化")
	}

	// 调用 SubscriptionManager 获取订阅（会更新数据库中的订阅和节点）
	_, err := ss.subscriptionManager.FetchSubscription(url, label...)
	if err != nil {
		return fmt.Errorf("获取订阅失败: %w", err)
	}

	// 获取后重新加载订阅数据
	if err := ss.store.Subscriptions.Load(); err != nil {
		return fmt.Errorf("刷新订阅数据失败: %w", err)
	}

	// 同时刷新节点数据（因为订阅获取会添加节点）
	if ss.store.Nodes != nil {
		if err := ss.store.Nodes.Load(); err != nil {
			return fmt.Errorf("刷新节点数据失败: %w", err)
		}
	}

	return nil
}

