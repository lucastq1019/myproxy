package service

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"myproxy.com/p/internal/store"
)

// 默认的国内域名直连路由列表
var defaultDirectRoutes = []string{
	"domain:baidu.com",
	"domain:qq.com",
	"domain:weixin.com",
	"domain:taobao.com",
	"domain:jd.com",
	"domain:aliyun.com",
	"domain:163.com",
	"domain:sina.com",
	"domain:sohu.com",
	"domain:youku.com",
	"domain:tudou.com",
	"domain:iqiyi.com",
	"domain:cntv.cn",
	"domain:mi.com",
	"domain:huawei.com",
	"domain:oppo.com",
	"domain:vivo.com",
	"domain:meituan.com",
	"domain:dianping.com",
	"domain:amap.com",
	"domain:ctrip.com",
	"domain:elong.com",
	"domain:tongcheng.com",
	"domain:qunar.com",
	"domain:kaola.com",
	"domain:suning.com",
	"domain:gome.com.cn",
	"domain:tmall.com",
	"domain:alicdn.com",
	"domain:cdn.baidustatic.com",
	"domain:qqstatic.com",
	"domain:wxstatic.com",
	"domain:taobaocdn.com",
	"domain:jdcdn.com",
	"domain:aliyuncdn.com",
	"domain:163cdn.com",
	"domain:sinaimg.cn",
}

// ConfigService 应用配置服务层，提供配置相关的业务逻辑。
type ConfigService struct {
	store *store.Store
}

// NewConfigService 创建新的配置服务实例。
// 参数：
//   - store: Store 实例，用于数据访问
//
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
//
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
//
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
//
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
//
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
//
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
//
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
//
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
//
// 返回：错误（如果有）
func (cs *ConfigService) Set(key, value string) error {
	if cs.store == nil || cs.store.AppConfig == nil {
		return fmt.Errorf("Store 未初始化")
	}
	return cs.store.AppConfig.Set(key, value)
}

// GetDirectRoutes 获取直连路由列表（域名或 IP/CIDR，每行一条，对应 xray 规则）。
// 返回：直连地址列表，空切片表示未配置
func (cs *ConfigService) GetDirectRoutes() []string {
	if cs.store == nil || cs.store.AppConfig == nil {
		return nil
	}
	raw, err := cs.store.AppConfig.GetWithDefault("directRoutes", "")
	if err != nil || raw == "" {
		return nil
	}
	return parseDirectRoutes(raw)
}

// GetDirectRoutesRaw 获取直连路由原始字符串（换行分隔），供 UI 多行输入框使用。
func (cs *ConfigService) GetDirectRoutesRaw() string {
	routes := cs.GetDirectRoutes()
	if len(routes) == 0 {
		return ""
	}
	return formatDirectRoutes(routes)
}

// SetDirectRoutesFromRaw 从 UI 多行字符串保存直连路由（会解析并规范化后存储）。
func (cs *ConfigService) SetDirectRoutesFromRaw(raw string) error {
	routes := parseDirectRoutes(raw)
	return cs.SetDirectRoutes(routes)
}

// SetDirectRoutes 保存直连路由列表。
// 参数：直连地址列表，会序列化为换行分隔的字符串存储
func (cs *ConfigService) SetDirectRoutes(routes []string) error {
	if cs.store == nil || cs.store.AppConfig == nil {
		return fmt.Errorf("Store 未初始化")
	}
	raw := formatDirectRoutes(routes)
	return cs.store.AppConfig.Set("directRoutes", raw)
}

// GetDirectRoutesUseProxy 获取「直连列表中的地址是否走代理」。
// true：直连列表中的地址走代理；false：走直连。
func (cs *ConfigService) GetDirectRoutesUseProxy() bool {
	if cs.store == nil || cs.store.AppConfig == nil {
		return false
	}
	v, _ := cs.store.AppConfig.GetWithDefault("directRoutesUseProxy", "false")
	return v == "true"
}

// SetDirectRoutesUseProxy 设置「直连列表中的地址是否走代理」。
func (cs *ConfigService) SetDirectRoutesUseProxy(useProxy bool) error {
	if cs.store == nil || cs.store.AppConfig == nil {
		return fmt.Errorf("Store 未初始化")
	}
	val := "false"
	if useProxy {
		val = "true"
	}
	return cs.store.AppConfig.Set("directRoutesUseProxy", val)
}

// GetTerminalProxyEnabled 获取是否启用终端代理配置。
// 返回：是否启用终端代理配置
func (cs *ConfigService) GetTerminalProxyEnabled() bool {
	if cs.store == nil || cs.store.AppConfig == nil {
		return false // 默认不启用
	}
	v, _ := cs.store.AppConfig.GetWithDefault("terminalProxyEnabled", "false")
	return v == "true"
}

// SetTerminalProxyEnabled 设置是否启用终端代理配置。
// 参数：
//   - enabled: 是否启用终端代理配置
//
// 返回：错误（如果有）
func (cs *ConfigService) SetTerminalProxyEnabled(enabled bool) error {
	if cs.store == nil || cs.store.AppConfig == nil {
		return fmt.Errorf("Store 未初始化")
	}
	val := "false"
	if enabled {
		val = "true"
	}
	return cs.store.AppConfig.Set("terminalProxyEnabled", val)
}

// GetProxyType 获取代理类型配置。
// 返回：代理类型（https 或 socks5）
func (cs *ConfigService) GetProxyType() string {
	if cs.store == nil || cs.store.AppConfig == nil {
		return "socks5" // 默认使用 socks5
	}
	v, _ := cs.store.AppConfig.GetWithDefault("proxyType", "socks5")
	return v
}

// SetProxyType 设置代理类型配置。
// 参数：
//   - proxyType: 代理类型（https 或 socks5）
//
// 返回：错误（如果有）
func (cs *ConfigService) SetProxyType(proxyType string) error {
	if cs.store == nil || cs.store.AppConfig == nil {
		return fmt.Errorf("Store 未初始化")
	}
	return cs.store.AppConfig.Set("proxyType", proxyType)
}

// parseDirectRoutes 从换行分隔的字符串解析直连路由列表。
// 支持 domain:xxx、ip 或 cidr，纯域名会补全为 domain:xxx。
func parseDirectRoutes(raw string) []string {
	var out []string
	for _, line := range strings.Split(raw, "\n") {
		s := strings.TrimSpace(line)
		if s == "" {
			continue
		}
		// 已是 domain: 或 geosite: 等前缀则保持
		if strings.HasPrefix(s, "domain:") || strings.HasPrefix(s, "geosite:") ||
			strings.HasPrefix(s, "regexp:") || strings.HasPrefix(s, "full:") {
			out = append(out, s)
			continue
		}
		// 简单启发式：含有点且非纯数字，视为域名
		if strings.Contains(s, ".") && !isLikelyIPOrCIDR(s) {
			out = append(out, "domain:"+s)
		} else {
			out = append(out, s)
		}
	}
	return out
}

func isLikelyIPOrCIDR(s string) bool {
	// 含 / 视为 CIDR；否则简单检查是否像 IP
	if strings.Contains(s, "/") {
		return true
	}
	for _, r := range s {
		if (r >= '0' && r <= '9') || r == '.' {
			continue
		}
		return false
	}
	return true
}

// formatDirectRoutes 将直连路由列表格式化为换行分隔的字符串。
func formatDirectRoutes(routes []string) string {
	return strings.TrimSpace(strings.Join(routes, "\n"))
}

// SaveDefaultDirectRoutes 保存默认的直连路由到数据库（仅在第一次运行时调用）。
// 如果数据库中已有路由配置，则不会覆盖。
func (cs *ConfigService) SaveDefaultDirectRoutes() error {
	if cs.store == nil || cs.store.AppConfig == nil {
		return fmt.Errorf("Store 未初始化")
	}

	existing, err := cs.store.AppConfig.Get("directRoutes")
	if err == nil && existing != "" {
		return nil
	}

	return cs.SetDirectRoutes(defaultDirectRoutes)
}

// RestoreDefaultDirectRoutes 恢复默认的直连路由（覆盖当前配置）。
func (cs *ConfigService) RestoreDefaultDirectRoutes() error {
	return cs.SetDirectRoutes(defaultDirectRoutes)
}

// GetDefaultDirectRoutes 获取默认的直连路由列表（不修改数据库）。
func (cs *ConfigService) GetDefaultDirectRoutes() []string {
	return defaultDirectRoutes
}
