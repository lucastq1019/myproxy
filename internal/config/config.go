package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Server 表示一个代理服务器的配置信息。
type Server struct {
	ID               string `json:"id"`                // 服务器唯一标识
	Name             string `json:"name"`              // 服务器名称
	Addr             string `json:"addr"`              // 服务器地址
	Port             int    `json:"port"`              // 服务器端口
	Username         string `json:"username"`          // 认证用户名
	Password         string `json:"password"`          // 认证密码
	Delay            int    `json:"delay"`             // 延迟（毫秒）
	Selected         bool   `json:"selected"`          // 是否被选中
	Enabled          bool   `json:"enabled"`           // 是否启用
	ProtocolType     string `json:"protocol_type"`     // 协议类型: vmess, ss, ssr, socks5, etc.
	
	// VMess 协议字段
	VMessVersion     string `json:"vmess_version,omitempty"`     // VMess 版本 (v)
	VMessUUID        string `json:"vmess_uuid,omitempty"`        // VMess UUID (id)
	VMessAlterID     int    `json:"vmess_alter_id,omitempty"`    // VMess AlterID (aid)
	VMessSecurity    string `json:"vmess_security,omitempty"`    // VMess 加密方式
	VMessNetwork     string `json:"vmess_network,omitempty"`     // VMess 传输协议 (net): tcp, kcp, ws, h2, quic, grpc
	VMessType        string `json:"vmess_type,omitempty"`        // VMess 伪装类型 (type): none, http, srtp, utp, wechat-video
	VMessHost        string `json:"vmess_host,omitempty"`        // VMess 伪装域名 (host)
	VMessPath        string `json:"vmess_path,omitempty"`        // VMess 路径 (path)
	VMessTLS         string `json:"vmess_tls,omitempty"`         // VMess TLS 配置 (tls): "", "tls"
	
	// Shadowsocks 协议字段
	SSMethod         string `json:"ss_method,omitempty"`         // Shadowsocks 加密方法
	SSPlugin         string `json:"ss_plugin,omitempty"`         // Shadowsocks 插件
	SSPluginOpts     string `json:"ss_plugin_opts,omitempty"`    // Shadowsocks 插件选项
	
	// ShadowsocksR 协议字段
	SSRObfs          string `json:"ssr_obfs,omitempty"`          // SSR 混淆
	SSRObfsParam     string `json:"ssr_obfs_param,omitempty"`    // SSR 混淆参数
	SSRProtocol      string `json:"ssr_protocol,omitempty"`      // SSR 协议
	SSRProtocolParam string `json:"ssr_protocol_param,omitempty"` // SSR 协议参数
	
	// 原始配置 JSON（用于存储完整的协议配置，便于未来扩展）
	RawConfig        string `json:"raw_config,omitempty"`        // 原始配置 JSON 字符串
}

// Config 存储应用的配置信息。
// 注意：GUI 应用使用数据库存储服务器和订阅信息，此配置主要用于日志和自动代理设置。
type Config struct {
	Servers          []Server `json:"servers"`          // 服务器列表（保留用于向后兼容，GUI 应用主要使用数据库）
	SelectedServerID string   `json:"selectedServerID"` // 当前选中的服务器ID
	AutoProxyEnabled bool     `json:"autoProxyEnabled"` // 自动代理是否启用
	AutoProxyPort    int      `json:"autoProxyPort"`    // 自动代理监听端口
	LogLevel         string   `json:"logLevel"`         // 日志级别
	LogFile          string   `json:"logFile"`          // 日志文件路径
}

// DefaultConfig 返回默认的应用配置。
// 返回：包含默认值的配置实例
func DefaultConfig() *Config {
	return &Config{
		AutoProxyEnabled: false,
		AutoProxyPort:    1080,
		LogLevel:         "info",
		LogFile:          "myproxy.log",
		Servers:          []Server{},
		SelectedServerID: "",
	}
}

// LoadConfig 从指定的 JSON 文件加载配置。
// 如果文件不存在，会创建包含默认配置的新文件。
// 参数：
//   - filePath: 配置文件路径
//
// 返回：配置实例和错误（如果有）
func LoadConfig(filePath string) (*Config, error) {
	// 如果文件不存在，返回默认配置
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		defaultConfig := DefaultConfig()
		// 保存默认配置到文件
		if err := SaveConfig(defaultConfig, filePath); err != nil {
			return nil, fmt.Errorf("保存默认配置失败: %w", err)
		}
		return defaultConfig, nil
	}

	// 读取配置文件
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析配置
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return &config, nil
}

// SaveConfig 将配置保存到指定的 JSON 文件。
// 如果目录不存在，会自动创建。
// 参数：
//   - config: 要保存的配置实例
//   - filePath: 配置文件路径
//
// 返回：错误（如果有）
func SaveConfig(config *Config, filePath string) error {
	// 验证配置
	if err := config.Validate(); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}

	// 创建目录（如果不存在）
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	// 序列化配置
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	// 保存到文件
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}

// Validate 验证配置的有效性。
// 该方法会检查日志级别、端口范围和服务器配置的合法性。
// 返回：如果配置无效则返回错误，否则返回 nil
func (c *Config) Validate() error {
	// 检查日志级别
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
		"fatal": true,
	}
	if c.LogLevel != "" && !validLogLevels[c.LogLevel] {
		return fmt.Errorf("无效的日志级别: %s", c.LogLevel)
	}

	// 注意：自动代理端口不进行有效性检查，允许用户根据实际情况选择任意端口

	// 检查服务器列表（如果存在）
	for i, server := range c.Servers {
		if server.ID == "" {
			return fmt.Errorf("服务器 %d 的ID不能为空", i)
		}

		if server.Addr == "" {
			return fmt.Errorf("服务器 %s 的地址不能为空", server.ID)
		}

		if server.Port <= 0 || server.Port > 65535 {
			return fmt.Errorf("服务器 %s 的端口无效: %d", server.ID, server.Port)
		}
	}

	return nil
}

// AddServer 添加服务器
func (c *Config) AddServer(server Server) error {
	// 检查ID是否已存在
	for _, s := range c.Servers {
		if s.ID == server.ID {
			return fmt.Errorf("服务器ID已存在: %s", server.ID)
		}
	}

	c.Servers = append(c.Servers, server)
	return nil
}

// RemoveServer 删除服务器
func (c *Config) RemoveServer(id string) error {
	for i, s := range c.Servers {
		if s.ID == id {
			c.Servers = append(c.Servers[:i], c.Servers[i+1:]...)
			// 如果删除的是选中的服务器，重置选中服务器
			if c.SelectedServerID == id {
				c.SelectedServerID = ""
			}
			return nil
		}
	}

	return fmt.Errorf("服务器不存在: %s", id)
}

// GetServer 获取服务器
func (c *Config) GetServer(id string) (*Server, error) {
	for i, s := range c.Servers {
		if s.ID == id {
			return &c.Servers[i], nil
		}
	}

	return nil, fmt.Errorf("服务器不存在: %s", id)
}

// SelectServer 选择服务器
func (c *Config) SelectServer(id string) error {
	// 检查服务器是否存在
	found := false
	for i, s := range c.Servers {
		if s.ID == id {
			found = true
			// 取消所有服务器的选中状态
			for j := range c.Servers {
				c.Servers[j].Selected = false
			}
			// 选中当前服务器
			c.Servers[i].Selected = true
			c.SelectedServerID = id
			break
		}
	}

	if !found {
		return fmt.Errorf("服务器不存在: %s", id)
	}

	return nil
}

// GetSelectedServer 获取当前选中的服务器
func (c *Config) GetSelectedServer() (*Server, error) {
	for _, s := range c.Servers {
		if s.ID == c.SelectedServerID {
			return &s, nil
		}
	}

	return nil, fmt.Errorf("没有选中的服务器")
}
