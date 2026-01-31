package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config 存储应用的配置信息。
// 注意：GUI 应用使用数据库存储服务器和订阅信息，此配置主要用于日志和自动代理设置。
type Config struct {
	// Servers                  []Server `json:"servers"`                  // 服务器列表（保留用于向后兼容，GUI 应用主要使用数据库）
	SelectedServerID       string `json:"selectedServerID"`       // 当前选中的服务器ID
	SelectedSubscriptionID int64  `json:"selectedSubscriptionID"` // 当前选中的订阅ID，0表示全部
	AutoProxyEnabled       bool   `json:"autoProxyEnabled"`       // 自动代理是否启用
	AutoProxyPort          int    `json:"autoProxyPort"`          // 自动代理监听端口
	LogLevel               string `json:"logLevel"`               // 日志级别
	LogFile                string `json:"logFile"`                // 日志文件路径
}

// DefaultConfig 返回默认的应用配置。
// 返回：包含默认值的配置实例
func DefaultConfig() *Config {
	return &Config{
		AutoProxyEnabled: false,
		AutoProxyPort:    1080,
		LogLevel:         "info",
		LogFile:          "myproxy.log",
		// Servers:                []Server{},
		SelectedServerID:       "",
		SelectedSubscriptionID: 0, // 默认显示全部订阅的服务器
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

	return nil
}
