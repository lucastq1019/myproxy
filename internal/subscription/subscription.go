package subscription

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"myproxy.com/p/internal/database"
	"myproxy.com/p/internal/server"
)

// ServerParser 服务器配置解析器接口
type ServerParser interface {
	// Parse 解析服务器配置字符串，返回服务器配置和错误
	Parse(content string) (*database.Node, error)
}

// VMessParser VMess协议解析器
type VMessParser struct{}

// Parse 解析VMess协议
func (p *VMessParser) Parse(content string) (*database.Node, error) {
	// 移除前缀
	vmessData := strings.TrimPrefix(content, "vmess://")
	// 解码Base64
	decoded, err := base64.StdEncoding.DecodeString(vmessData)
	if err != nil {
		// 尝试URL安全的Base64解码
		decoded, err = base64.URLEncoding.DecodeString(vmessData)
		if err != nil {
			return nil, err
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
		return nil, err
	}

	// 将port转换为整数
	port, err := strconv.Atoi(vmessConfig.Port)
	if err != nil {
		return nil, err
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
	s := &database.Node{
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

	return s, nil
}

// SSConfig SS协议配置
type SSConfig struct {
	Cipher     string
	Password   string
	Addr       string
	Port       int
	Plugin     string
	PluginOpts string
}

// SSParser SS协议解析器
type SSParser struct{}

// Parse 解析SS协议
func (p *SSParser) Parse(content string) (*database.Node, error) {
	// 移除前缀
	ssData := strings.TrimPrefix(content, "ss://")

	// 处理可能的备注部分
	ssDataWithoutRemark := ssData
	if idx := strings.Index(ssData, "#"); idx != -1 {
		ssDataWithoutRemark = ssData[:idx]
	}

	// 找到 @ 符号，将字符串分为两部分
	base64Part, addrPortPart, found := strings.Cut(ssDataWithoutRemark, "@")
	var cipher, password, cipherPasswdPart string

	if !found {
		// 如果没有 @ 符号，说明整个部分都是 Base64 编码的
		// 解码Base64 - 处理可能的填充问题
		base64Str := ssDataWithoutRemark
		// 确保Base64字符串的长度是4的倍数，必要时添加填充字符
		for len(base64Str)%4 != 0 {
			base64Str += "="
		}
		decoded, err := base64.StdEncoding.DecodeString(base64Str)
		if err != nil {
			// 尝试URL安全的Base64解码
			decoded, err = base64.URLEncoding.DecodeString(base64Str)
			if err != nil {
				return nil, err
			}
		}

		ssStr := string(decoded)
		// 解析内部结构：cipher:password@addr:port
		cipherPasswdPart, addrPortPart, found = strings.Cut(ssStr, "@")
		if !found {
			return nil, fmt.Errorf("invalid SS format: missing @ separator in decoded string")
		}
		cipher, password, found = strings.Cut(cipherPasswdPart, ":")
		if !found {
			return nil, fmt.Errorf("invalid SS format: missing cipher:password")
		}
	} else {
		// 解码Base64
		decoded, err := base64.StdEncoding.DecodeString(base64Part)
		if err != nil {
			// 尝试URL安全的Base64解码
			decoded, err = base64.URLEncoding.DecodeString(base64Part)
			if err != nil {
				return nil, err
			}
		}

		ssStr := string(decoded)
		// 解析SS配置
		// 格式：cipher:password
		cipher, password, found = strings.Cut(ssStr, ":")
		if !found {
			return nil, fmt.Errorf("invalid SS format: missing cipher:password")
		}
	}

	// 解析地址和端口，以及可能的参数
	addrPort, pluginPart, _ := strings.Cut(addrPortPart, "?")
	addr, portStr, found := strings.Cut(addrPort, ":")
	if !found {
		return nil, fmt.Errorf("invalid SS format: missing addr:port")
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid SS port: %w", err)
	}

	// 解析插件参数
	var plugin, pluginOpts string
	if pluginPart != "" {
		pluginParams := strings.Split(pluginPart, "&")
		for _, param := range pluginParams {
			key, value, _ := strings.Cut(param, "=")
			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)

			if key == "plugin" {
				plugin = value
			} else if key == "plugin-opts" {
				pluginOpts = value
			}
		}
	}

	// 生成服务器ID
	serverID := server.GenerateServerID(addr, port, password)

	// 创建服务器配置
	s := &database.Node{
		ID:           serverID,
		Name:         fmt.Sprintf("%s:%d", addr, port),
		Addr:         addr,
		Port:         port,
		Username:     password, // SS使用密码作为标识
		Password:     password,
		Delay:        0,
		Selected:     false,
		Enabled:      true,
		ProtocolType: "ss",
		// SS 协议字段
		SSMethod:     cipher,
		SSPlugin:     plugin,
		SSPluginOpts: pluginOpts,
		// 保存原始配置
		RawConfig: content,
	}

	// 设置名称（从备注中获取）
	if idx := strings.Index(ssData, "#"); idx != -1 {
		remark := ssData[idx+1:]
		if decodedRemark, err := url.QueryUnescape(remark); err == nil {
			s.Name = decodedRemark
		} else {
			s.Name = remark
		}
	}

	return s, nil
}

// TrojanConfig Trojan协议配置
type TrojanConfig struct {
	Password      string
	Addr          string
	Port          int
	SNI           string
	Alpn          string
	AllowInsecure bool
}

// TrojanParser Trojan协议解析器
type TrojanParser struct{}

// Parse 解析Trojan协议
func (p *TrojanParser) Parse(content string) (*database.Node, error) {
	// 移除前缀
	trojanData := strings.TrimPrefix(content, "trojan://")

	// 处理可能的备注部分
	trojanDataWithoutRemark := trojanData
	name := ""
	if idx := strings.Index(trojanData, "#"); idx != -1 {
		trojanDataWithoutRemark = trojanData[:idx]
		name = trojanData[idx+1:]
		// 解码备注
		if decodedName, err := url.QueryUnescape(name); err == nil {
			name = decodedName
		}
	}

	// 解析密码和地址端口部分，以及可能的参数
	// 格式：password@addr:port?param1=value1&param2=value2
	passwordAddrPart, paramPart, _ := strings.Cut(trojanDataWithoutRemark, "?")

	// 解析密码和地址端口
	password, addrPort, found := strings.Cut(passwordAddrPart, "@")
	if !found {
		return nil, fmt.Errorf("invalid Trojan format: missing @ separator")
	}

	// 解析地址和端口
	addr, portStr, found := strings.Cut(addrPort, ":")
	if !found {
		return nil, fmt.Errorf("invalid Trojan format: missing addr:port")
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid Trojan port: %w", err)
	}

	// 解析参数部分
	var sni, alpn string
	allowInsecure := false

	if paramPart != "" {
		params := strings.Split(paramPart, "&")
		for _, param := range params {
			key, value, _ := strings.Cut(param, "=")
			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)

			switch key {
			case "sni":
				sni = value
			case "alpn":
				alpn = value
			case "allowInsecure":
				allowInsecure = value == "1" || strings.ToLower(value) == "true"
			}
		}
	}

	// 生成服务器ID
	serverID := server.GenerateServerID(addr, port, password)

	// 创建服务器配置
	s := &database.Node{
		ID:           serverID,
		Name:         name,
		Addr:         addr,
		Port:         port,
		Username:     password, // Trojan使用密码作为标识
		Password:     password,
		Delay:        0,
		Selected:     false,
		Enabled:      true,
		ProtocolType: "trojan",
		// Trojan 协议字段
		TrojanPassword:      password,
		TrojanSNI:           sni,
		TrojanAlpn:          alpn,
		TrojanAllowInsecure: allowInsecure,
		// 保存原始配置
		RawConfig: content,
	}

	// 如果名称为空，使用地址:端口作为名称
	if s.Name == "" {
		s.Name = fmt.Sprintf("%s:%d", s.Addr, s.Port)
	}

	return s, nil
}

// SOCKS5Parser SOCKS5协议解析器
type SOCKS5Parser struct{}

// Parse 解析SOCKS5协议
func (p *SOCKS5Parser) Parse(content string) (*database.Node, error) {
	socks5Regex := regexp.MustCompile(`^socks5://(?:([^:]+):([^@]+)@)?([^:]+):(\d+)$`)
	matches := socks5Regex.FindStringSubmatch(content)
	if matches == nil {
		return nil, fmt.Errorf("invalid SOCKS5 format")
	}

	username := matches[1]
	password := matches[2]
	addr := matches[3]
	portStr := matches[4]

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid SOCKS5 port: %w", err)
	}

	// 生成服务器ID
	serverID := server.GenerateServerID(addr, port, username)

	// 创建服务器配置
	s := &database.Node{
		ID:           serverID,
		Name:         fmt.Sprintf("%s:%d", addr, port),
		Addr:         addr,
		Port:         port,
		Username:     username,
		Password:     password,
		Delay:        0,
		Selected:     false,
		Enabled:      true,
		ProtocolType: "socks5",
		RawConfig:    content,
	}

	return s, nil
}

// SimpleParser 简单格式解析器
type SimpleParser struct{}

// Parse 解析简单格式
func (p *SimpleParser) Parse(content string) (*database.Node, error) {
	simpleRegex := regexp.MustCompile(`^([^:]+):(\d+)\s+([^\s]+)\s+([^\s]+)$`)
	matches := simpleRegex.FindStringSubmatch(content)
	if matches == nil {
		return nil, fmt.Errorf("invalid simple format")
	}

	addr := matches[1]
	portStr := matches[2]
	username := matches[3]
	password := matches[4]

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid simple port: %w", err)
	}

	// 生成服务器ID
	serverID := server.GenerateServerID(addr, port, username)

	// 创建服务器配置
	s := &database.Node{
		ID:           serverID,
		Name:         fmt.Sprintf("%s:%d", addr, port),
		Addr:         addr,
		Port:         port,
		Username:     username,
		Password:     password,
		Delay:        0,
		Selected:     false,
		Enabled:      true,
		ProtocolType: "socks5", // 简单格式默认为 SOCKS5
		RawConfig:    content,
	}

	return s, nil
}

// SubscriptionManager 订阅管理器
type SubscriptionManager struct {
	serverManager *server.ServerManager
	client        *http.Client
	parsers       map[string]ServerParser  // 服务器配置解析器映射，key为协议前缀
	subscriptions []*database.Subscription // 订阅列表
}

// NewSubscriptionManager 创建新的订阅管理器
func NewSubscriptionManager(serverManager *server.ServerManager) *SubscriptionManager {
	// 注册所有支持的解析器
	parsers := make(map[string]ServerParser)
	parsers["vmess://"] = &VMessParser{}
	parsers["ss://"] = &SSParser{}
	parsers["trojan://"] = &TrojanParser{}
	parsers["socks5://"] = &SOCKS5Parser{}

	sm := &SubscriptionManager{
		serverManager: serverManager,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		parsers: parsers,
	}

	// 初始化时从数据库加载订阅列表
	sm.LoadSubscriptionsFromDB()

	return sm
}

// LoadSubscriptionsFromDB 从数据库加载订阅列表到内存
func (sm *SubscriptionManager) LoadSubscriptionsFromDB() error {
	subscriptions, err := database.GetAllSubscriptions()
	if err != nil {
		return fmt.Errorf("加载订阅列表失败: %w", err)
	}

	sm.subscriptions = subscriptions
	return nil
}

// GetSubscriptions 获取所有订阅列表
func (sm *SubscriptionManager) GetSubscriptions() []*database.Subscription {
	return sm.subscriptions
}

// FetchSubscription 从URL获取订阅服务器列表
// label 参数用于为订阅添加标签，如果为空则使用默认标签
func (sm *SubscriptionManager) FetchSubscription(url string, label ...string) ([]database.Node, error) {
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

	// 更新内存中的订阅列表
	if err := sm.LoadSubscriptionsFromDB(); err != nil {
		return nil, fmt.Errorf("更新订阅列表失败: %w", err)
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

	// 更新内存中的订阅列表
	if err := sm.LoadSubscriptionsFromDB(); err != nil {
		return fmt.Errorf("更新订阅列表失败: %w", err)
	}

	return nil
}

// UpdateSubscriptionByID 根据订阅 ID 更新订阅。
// 该方法会先获取订阅信息，然后拉取最新的订阅内容并更新。
// 参数：
//   - id: 订阅 ID
//
// 返回：错误（如果有）
func (sm *SubscriptionManager) UpdateSubscriptionByID(id int64) error {
	// 根据 ID 获取订阅信息
	sub, err := database.GetSubscriptionByID(id)
	if err != nil {
		return fmt.Errorf("获取订阅信息失败: %w", err)
	}
	if sub == nil {
		return fmt.Errorf("订阅不存在")
	}
	
	// 调用 UpdateSubscription 更新订阅（会拉取最新内容）
	return sm.UpdateSubscription(sub.URL, sub.Label)
}

// parseSubscription 解析订阅内容
func (sm *SubscriptionManager) parseSubscription(content string) ([]database.Node, error) {
	// 尝试解码Base64
	decoded, err := base64.StdEncoding.DecodeString(content)
	if err == nil {
		content = string(decoded)
	}

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
		servers := make([]database.Node, len(jsonServers))
		for i, js := range jsonServers {
			rawConfig, _ := json.Marshal(js)
			servers[i] = database.Node{
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
	var servers []database.Node

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

		// 使用注册的解析器解析服务器配置
		var parsedServer *database.Node

		// 直接根据前缀获取解析器
		// 查找字符串中第一个 "://" 出现的位置
		if idx := strings.Index(line, "://"); idx != -1 {
			// 提取前缀（包括 "://"）
			prefix := line[:idx+3]
			// 从 map 中获取对应的解析器
			if parser, ok := sm.parsers[prefix]; ok {
				parsedServer, err = parser.Parse(line)
			}
		}

		// 如果没有找到解析器或解析失败，尝试使用 SimpleParser
		if parsedServer == nil {
			simpleParser := &SimpleParser{}
			parsedServer, err = simpleParser.Parse(line)
		}

		// 如果解析成功，添加到服务器列表
		if parsedServer != nil {
			servers = append(servers, *parsedServer)
		}
	}

	if len(servers) == 0 {
		return nil, fmt.Errorf("不支持的订阅格式")
	}

	return servers, nil
}
