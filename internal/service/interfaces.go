package service

import (
	"myproxy.com/p/internal/model"
	"myproxy.com/p/internal/store"
)

// ProxyServiceInterface 定义代理服务接口
type ProxyServiceInterface interface {
	UpdateXrayInstance(xrayInstance interface{})
	ApplySystemProxyMode(mode string) *ApplySystemProxyModeResult
}

// ServerServiceInterface 定义服务器服务接口
type ServerServiceInterface interface {
	GetAllServers() ([]*model.Node, error)
	GetServerByID(id string) (*model.Node, error)
	GetServersBySubscriptionID(subscriptionID int64) ([]*model.Node, error)
	UpdateServerDelay(id string, delay int) error
	DeleteServer(id string) error
	AddOrUpdateServer(node model.Node, subscriptionID *int64) error
}

// ConfigServiceInterface 定义配置服务接口
type ConfigServiceInterface interface {
	Get(key string) (string, error)
	GetWithDefault(key, defaultValue string) (string, error)
	Set(key, value string) error
	GetWindowSize(defaultSize interface{}) interface{}
	SaveWindowSize(size interface{}) error
}

// SubscriptionServiceInterface 定义订阅服务接口
type SubscriptionServiceInterface interface {
	GetAllSubscriptions() ([]interface{}, error)
	GetSubscriptionByID(id int64) (interface{}, error)
	GetSubscriptionByURL(url string) (interface{}, error)
	AddSubscription(url, label string) (interface{}, error)
	UpdateSubscription(id int64, url, label string) error
	DeleteSubscription(id int64) error
	UpdateSubscriptionByID(id int64) error
	FetchSubscription(url string, label ...string) (interface{}, error)
	GetServerCount(id int64) (int, error)
}

// XrayControlServiceInterface 定义 Xray 控制服务接口
type XrayControlServiceInterface interface {
	StartProxy(xrayInstance interface{}, logPath string) *StartProxyResult
	StopProxy() error
	GetProxyStatus() interface{}
}

// StoreInterface 定义 Store 接口
type StoreInterface interface {
	IsInitialized() bool
	Reset()
	LoadAll()
	GetNodes() []*model.Node
	GetSubscriptions() []interface{}
	GetAppConfig() interface{}
	GetProxyStatus() interface{}
}

// NodesStoreInterface 定义节点存储接口
type NodesStoreInterface interface {
	Load() error
	GetAll() []*model.Node
	Get(id string) (*model.Node, error)
	GetSelected() *model.Node
	GetSelectedID() string
	Select(id string) error
	UpdateDelay(id string, delay int) error
	Delete(id string) error
	Add(node *model.Node) error
	Update(node *model.Node) error
	GetBySubscriptionID(subscriptionID int64) ([]*model.Node, error)
}

// SubscriptionsStoreInterface 定义订阅存储接口
type SubscriptionsStoreInterface interface {
	Load() error
	GetAll() []interface{}
	GetAllNodeCount() int
	Get(id int64) (interface{}, error)
	GetByURL(url string) (interface{}, error)
	Add(url, label string) (interface{}, error)
	Update(id int64, url, label string) error
	Delete(id int64) error
	GetServerCount(id int64) (int, error)
}

// AppConfigStoreInterface 定义应用配置存储接口
type AppConfigStoreInterface interface {
	Load() error
	Get(key string) (string, error)
	GetWithDefault(key, defaultValue string) (string, error)
	Set(key, value string) error
}

// ProxyStatusStoreInterface 定义代理状态存储接口
type ProxyStatusStoreInterface interface {
	UpdateProxyStatus(xrayInstance interface{}, nodesStore *store.NodesStore)
}
