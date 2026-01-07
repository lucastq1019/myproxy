package service

import (
	"fmt"

	"fyne.io/fyne/v2"
	"myproxy.com/p/internal/store"
)

// ConfigService 应用配置服务层，提供配置相关的业务逻辑。
type ConfigService struct {
	store *store.Store
}

// NewConfigService 创建新的配置服务实例。
// 参数：
//   - store: Store 实例，用于数据访问
// 返回：初始化后的 ConfigService 实例
func NewConfigService(store *store.Store) *ConfigService {
	return &ConfigService{
		store: store,
	}
}

// GetTheme 获取主题配置。
// 返回：主题变体（dark 或 light）
func (cs *ConfigService) GetTheme() string {
	if cs.store == nil || cs.store.AppConfig == nil {
		return "dark"
	}
	themeStr, err := cs.store.AppConfig.GetWithDefault("theme", "dark")
	if err != nil {
		return "dark"
	}
	return themeStr
}

// SetTheme 设置主题配置。
// 参数：
//   - theme: 主题变体（dark 或 light）
// 返回：错误（如果有）
func (cs *ConfigService) SetTheme(theme string) error {
	if cs.store == nil || cs.store.AppConfig == nil {
		return fmt.Errorf("Store 未初始化")
	}
	return cs.store.AppConfig.Set("theme", theme)
}

// GetWindowSize 获取窗口大小。
// 参数：
//   - defaultSize: 默认窗口大小
// 返回：窗口大小
func (cs *ConfigService) GetWindowSize(defaultSize fyne.Size) fyne.Size {
	if cs.store == nil || cs.store.AppConfig == nil {
		return defaultSize
	}
	return cs.store.AppConfig.GetWindowSize(defaultSize)
}

// SaveWindowSize 保存窗口大小。
// 参数：
//   - size: 窗口大小
// 返回：错误（如果有）
func (cs *ConfigService) SaveWindowSize(size fyne.Size) error {
	if cs.store == nil || cs.store.AppConfig == nil {
		return fmt.Errorf("Store 未初始化")
	}
	return cs.store.AppConfig.SaveWindowSize(size)
}

// GetLogsCollapsed 获取日志面板折叠状态。
// 返回：是否折叠
func (cs *ConfigService) GetLogsCollapsed() bool {
	if cs.store == nil || cs.store.AppConfig == nil {
		return true // 默认折叠
	}
	collapsed, err := cs.store.AppConfig.GetWithDefault("logsCollapsed", "true")
	if err != nil {
		return true
	}
	return collapsed == "true"
}

// SetLogsCollapsed 设置日志面板折叠状态。
// 参数：
//   - collapsed: 是否折叠
// 返回：错误（如果有）
func (cs *ConfigService) SetLogsCollapsed(collapsed bool) error {
	if cs.store == nil || cs.store.AppConfig == nil {
		return fmt.Errorf("Store 未初始化")
	}
	state := "false"
	if collapsed {
		state = "true"
	}
	return cs.store.AppConfig.Set("logsCollapsed", state)
}

// GetSystemProxyMode 获取系统代理模式。
// 返回：系统代理模式（clear, auto, terminal）
func (cs *ConfigService) GetSystemProxyMode() string {
	if cs.store == nil || cs.store.AppConfig == nil {
		return ""
	}
	mode, err := cs.store.AppConfig.Get("systemProxyMode")
	if err != nil {
		return ""
	}
	return mode
}

// SetSystemProxyMode 设置系统代理模式。
// 参数：
//   - mode: 系统代理模式（clear, auto, terminal）
// 返回：错误（如果有）
func (cs *ConfigService) SetSystemProxyMode(mode string) error {
	if cs.store == nil || cs.store.AppConfig == nil {
		return fmt.Errorf("Store 未初始化")
	}
	return cs.store.AppConfig.Set("systemProxyMode", mode)
}

// Get 获取配置值。
// 参数：
//   - key: 配置键
// 返回：配置值和错误（如果有）
func (cs *ConfigService) Get(key string) (string, error) {
	if cs.store == nil || cs.store.AppConfig == nil {
		return "", fmt.Errorf("Store 未初始化")
	}
	return cs.store.AppConfig.Get(key)
}

// GetWithDefault 获取配置值，如果不存在则返回默认值。
// 参数：
//   - key: 配置键
//   - defaultValue: 默认值
// 返回：配置值
func (cs *ConfigService) GetWithDefault(key, defaultValue string) (string, error) {
	if cs.store == nil || cs.store.AppConfig == nil {
		return defaultValue, nil
	}
	return cs.store.AppConfig.GetWithDefault(key, defaultValue)
}

// Set 设置配置值。
// 参数：
//   - key: 配置键
//   - value: 配置值
// 返回：错误（如果有）
func (cs *ConfigService) Set(key, value string) error {
	if cs.store == nil || cs.store.AppConfig == nil {
		return fmt.Errorf("Store 未初始化")
	}
	return cs.store.AppConfig.Set(key, value)
}

