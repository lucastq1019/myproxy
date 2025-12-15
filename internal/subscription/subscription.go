package subscription

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"myproxy.com/p/internal/config"
	"myproxy.com/p/internal/database"
	"myproxy.com/p/internal/server"
)

// SubscriptionManager 订阅管理器
type SubscriptionManager struct {
	serverManager *server.ServerManager
	client        *http.Client
}

// NewSubscriptionManager 创建新的订阅管理器
func NewSubscriptionManager(serverManager *server.ServerManager) *SubscriptionManager {
	return &SubscriptionManager{
		serverManager: serverManager,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchSubscription 从URL获取订阅服务器列表
// label 参数用于为订阅添加标签，如果为空则使用默认标签
func (sm *SubscriptionManager) FetchSubscription(url string, label ...string) ([]config.Server, error) {
	// 发送HTTP请求获取订阅内容
	resp, err := sm.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("获取订阅失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取订阅内容失败: %w", err)
	}

	// 解析订阅内容
	servers, err := sm.parseSubscription(string(body))
	if err != nil {
		return nil, fmt.Errorf("解析订阅失败: %w", err)
	}

	// 保存订阅到数据库
	subscriptionLabel := ""
	if len(label) > 0 && label[0] != "" {
		subscriptionLabel = label[0]
	}

	sub, err := database.AddOrUpdateSubscription(url, subscriptionLabel)
	if err != nil {
		return nil, fmt.Errorf("保存订阅到数据库失败: %w", err)
	}

	// 保存服务器到数据库
	var subscriptionID *int64
	if sub != nil {
		subscriptionID = &sub.ID
	}

	for _, s := range servers {
		if err := database.AddOrUpdateServer(s, subscriptionID); err != nil {
			return nil, fmt.Errorf("保存服务器到数据库失败: %w", err)
		}
	}

	return servers, nil
}

// UpdateSubscription 更新订阅
// label 参数用于更新订阅标签，如果为空则保持原有标签
func (sm *SubscriptionManager) UpdateSubscription(url string, label ...string) error {
	// 获取订阅服务器列表（会自动保存到数据库）
	subscriptionLabel := ""
	if len(label) > 0 && label[0] != "" {
		subscriptionLabel = label[0]
	} else {
		// 如果未提供标签，尝试从数据库获取现有标签
		existingSub, err := database.GetSubscriptionByURL(url)
		if err == nil && existingSub != nil {
			subscriptionLabel = existingSub.Label
		}
	}

	// 获取现有订阅（用于清理旧服务器）
	existingSub, err := database.GetSubscriptionByURL(url)
	if err != nil {
		return fmt.Errorf("获取订阅信息失败: %w", err)
	}

	// 如果存在旧订阅，先清理该订阅下的服务器，避免更新后重复累加
	if existingSub != nil {
		if err := database.DeleteServersBySubscriptionID(existingSub.ID); err != nil {
			return fmt.Errorf("清理旧订阅服务器失败: %w", err)
		}
	}

	// 拉取并保存最新服务器；内部会更新订阅标签并写库
	servers, err := sm.FetchSubscription(url, subscriptionLabel)
	if err != nil {
		return err
	}

	// 再次获取订阅信息（防止标签更新或首次创建）
	sub, err := database.GetSubscriptionByURL(url)
	if err != nil {
		return fmt.Errorf("获取订阅信息失败: %w", err)
	}

	var subscriptionID *int64
	if sub != nil {
		subscriptionID = &sub.ID
	}

	// 更新服务器列表（同时更新内存和数据库）
	for _, s := range servers {
		// 检查服务器是否已存在（从数据库）
		existingServer, err := database.GetServer(s.ID)
		if err == nil {
			// 服务器已存在，保留选中状态和延迟
			s.Selected = existingServer.Selected
			s.Delay = existingServer.Delay
		}

		// 更新数据库中的服务器信息
		if err := database.AddOrUpdateServer(s, subscriptionID); err != nil {
			return fmt.Errorf("更新服务器到数据库失败: %w", err)
		}

		// 同时更新内存中的服务器信息（保持兼容性）
		if err := sm.serverManager.UpdateServer(s); err != nil {
			// 如果内存中不存在，则添加
			if err := sm.serverManager.AddServer(s); err != nil {
				return fmt.Errorf("更新服务器到内存失败: %w", err)
			}
		}
	}

	return nil
}

// parseSubscription 解析订阅内容
func (sm *SubscriptionManager) parseSubscription(content string) ([]config.Server, error) {
	// 尝试解码Base64
	decoded, err := base64.StdEncoding.DecodeString(content)
	if err == nil {
		content = string(decoded)
	}
	fmt.Println(content)

	// 尝试不同的订阅格式

	// 1. 尝试JSON格式
	var jsonServers []struct {
		Name     string `json:"name"`
		Addr     string `json:"addr"`
		Port     int    `json:"port"`
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.Unmarshal([]byte(content), &jsonServers); err == nil {
		// JSON格式解析成功
		servers := make([]config.Server, len(jsonServers))
		for i, js := range jsonServers {
			rawConfig, _ := json.Marshal(js)
			servers[i] = config.Server{
				ID:           server.GenerateServerID(js.Addr, js.Port, js.Username),
				Name:         js.Name,
				Addr:         js.Addr,
				Port:         js.Port,
				Username:     js.Username,
				Password:     js.Password,
				Delay:        0,
				Selected:     false,
				Enabled:      true,
				ProtocolType: "socks5", // JSON格式默认为 SOCKS5
				RawConfig:    string(rawConfig),
			}
		}
		return servers, nil
	}

	// 2. 尝试Clash格式 (每行一个服务器配置)
	lines := strings.Split(content, "\n")
	var servers []config.Server

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 尝试解析Clash格式
		if strings.HasPrefix(line, "- name:") {
			// 多行Clash格式，暂时不支持
			continue
		}

		// 尝试解析VMess格式
		if strings.HasPrefix(line, "vmess://") {
			// 移除前缀
			vmessData := strings.TrimPrefix(line, "vmess://")
			// 解码Base64
			decoded, err := base64.StdEncoding.DecodeString(vmessData)
			if err != nil {
				// 尝试URL安全的Base64解码
				decoded, err = base64.URLEncoding.DecodeString(vmessData)
				if err != nil {
					continue
				}
			}

			// 解析JSON - 包含所有字段
			var vmessConfig struct {
				V    string `json:"v"`    // 版本
				Ps   string `json:"ps"`   // 备注/名称
				Add  string `json:"add"`  // 地址
				Port string `json:"port"` // 端口（字符串类型）
				Id   string `json:"id"`   // UUID
				Aid  string `json:"aid"`  // AlterID（字符串类型，可能是 "0"）
				Net  string `json:"net"`  // 传输协议: tcp, kcp, ws, h2, quic, grpc
				Type string `json:"type"` // 伪装类型: none, http, srtp, utp, wechat-video
				Host string `json:"host"` // 伪装域名
				Path string `json:"path"` // 路径
				Tls  string `json:"tls"`  // TLS: "" 或 "tls"
			}

			decodedStr := string(decoded)

			if err := json.Unmarshal(decoded, &vmessConfig); err != nil {
				fmt.Printf("解析 VMess JSON 失败: %v, 内容: %s\n", err, decodedStr)
				continue
			}

			// 将port转换为整数
			port, err := strconv.Atoi(vmessConfig.Port)
			if err != nil {
				fmt.Printf("解析端口失败: %v, 端口值: %s\n", err, vmessConfig.Port)
				continue
			}

			// 将aid转换为整数
			aid := 0
			if vmessConfig.Aid != "" {
				if aidInt, err := strconv.Atoi(vmessConfig.Aid); err == nil {
					aid = aidInt
				}
			}

			// 生成服务器ID（使用 addr:port:uuid）
			serverID := server.GenerateServerID(vmessConfig.Add, port, vmessConfig.Id)

			// 创建服务器配置，包含所有字段
			s := config.Server{
				ID:           serverID,
				Name:         vmessConfig.Ps,
				Addr:         vmessConfig.Add,
				Port:         port,
				Username:     vmessConfig.Id, // VMess 使用 UUID 作为标识
				Password:     "",
				Delay:        0,
				Selected:     false,
				Enabled:      true,
				ProtocolType: "vmess",
				// VMess 协议字段
				VMessVersion:  vmessConfig.V,
				VMessUUID:     vmessConfig.Id,
				VMessAlterID:  aid,
				VMessSecurity: "auto", // 默认加密方式
				VMessNetwork:  vmessConfig.Net,
				VMessType:     vmessConfig.Type,
				VMessHost:     vmessConfig.Host,
				VMessPath:     vmessConfig.Path,
				VMessTLS:      vmessConfig.Tls,
				// 保存原始配置 JSON
				RawConfig: decodedStr,
			}

			// 如果名称为空，使用地址:端口作为名称
			if s.Name == "" {
				s.Name = fmt.Sprintf("%s:%d", s.Addr, s.Port)
			}

			servers = append(servers, s)
			continue
		}

		// 尝试解析SSR/SS格式
		if strings.HasPrefix(line, "ssr://") || strings.HasPrefix(line, "ss://") {
			// SSR/SS格式，暂时不支持
			continue
		}

		// 尝试解析SOCKS5格式
		// 格式: socks5://username:password@addr:port
		socks5Regex := regexp.MustCompile(`^socks5://(?:([^:]+):([^@]+)@)?([^:]+):(\d+)$`)
		matches := socks5Regex.FindStringSubmatch(line)
		if matches != nil {
			port, _ := strconv.Atoi(matches[4])
			s := config.Server{
				ID:           server.GenerateServerID(matches[3], port, matches[1]),
				Name:         fmt.Sprintf("%s:%d", matches[3], port),
				Addr:         matches[3],
				Port:         port,
				Username:     matches[1],
				Password:     matches[2],
				Delay:        0,
				Selected:     false,
				Enabled:      true,
				ProtocolType: "socks5",
				RawConfig:    line,
			}
			servers = append(servers, s)
			continue
		}

		// 尝试简单格式: addr:port username password
		simpleRegex := regexp.MustCompile(`^([^:]+):(\d+)\s+([^\s]+)\s+([^\s]+)$`)
		matches = simpleRegex.FindStringSubmatch(line)
		if matches != nil {
			port, _ := strconv.Atoi(matches[2])
			s := config.Server{
				ID:           server.GenerateServerID(matches[1], port, matches[3]),
				Name:         fmt.Sprintf("%s:%d", matches[1], port),
				Addr:         matches[1],
				Port:         port,
				Username:     matches[3],
				Password:     matches[4],
				Delay:        0,
				Selected:     false,
				Enabled:      true,
				ProtocolType: "socks5", // 简单格式默认为 SOCKS5
				RawConfig:    line,
			}
			servers = append(servers, s)
		}
	}

	if len(servers) == 0 {
		return nil, fmt.Errorf("不支持的订阅格式")
	}

	return servers, nil
}
