package xray

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	// 导入所有 xray-core 组件，注册必要的处理器
	_ "github.com/xtls/xray-core/main/distro/all"

	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/infra/conf"
	"myproxy.com/p/internal/model"
)

// LogCallback 定义日志回调函数类型
// 参数：level (日志级别，如 "INFO", "ERROR"), message (日志消息)
type LogCallback func(level, message string)

// logWriter 是一个自定义的日志写入器，用于拦截 xray 的日志输出
type logWriter struct {
	callback LogCallback
	buffer   []byte
	mu       sync.Mutex
}

// NewLogWriter 创建新的日志写入器
func NewLogWriter(callback LogCallback) *logWriter {
	return &logWriter{
		callback: callback,
		buffer:   make([]byte, 0, 1024),
	}
}

// SetCallback 设置日志回调函数
func (lw *logWriter) SetCallback(callback LogCallback) {
	lw.mu.Lock()
	defer lw.mu.Unlock()
	lw.callback = callback
}

// Write 实现 io.Writer 接口
func (lw *logWriter) Write(p []byte) (n int, err error) {
	lw.mu.Lock()
	defer lw.mu.Unlock()

	// 将数据添加到缓冲区
	lw.buffer = append(lw.buffer, p...)

	// 按行处理日志
	for {
		// 查找换行符
		newlineIndex := -1
		for i, b := range lw.buffer {
			if b == '\n' {
				newlineIndex = i
				break
			}
		}

		// 如果没有找到换行符，等待更多数据
		if newlineIndex == -1 {
			break
		}

		// 提取一行日志
		line := string(lw.buffer[:newlineIndex])
		lw.buffer = lw.buffer[newlineIndex+1:]

		// 处理日志行
		if strings.TrimSpace(line) != "" {
			lw.processLogLine(line)
		}
	}

	return len(p), nil
}

// processLogLine 处理单行日志，解析级别并调用回调
func (lw *logWriter) processLogLine(line string) {
	if lw.callback == nil {
		return
	}

	// 移除可能的回车符
	line = strings.TrimRight(line, "\r\n")

	// 过滤掉无意义的频繁日志
	if lw.shouldFilterLog(line) {
		return
	}

	// 解析日志级别（xray-core 的日志格式通常包含级别信息）
	level := "INFO"
	message := line

	// 尝试解析常见的日志格式
	upperLine := strings.ToUpper(line)
	if strings.Contains(upperLine, "[ERROR]") || strings.Contains(upperLine, " ERROR ") {
		level = "ERROR"
	} else if strings.Contains(upperLine, "[WARN]") || strings.Contains(upperLine, " WARN ") {
		level = "WARN"
	} else if strings.Contains(upperLine, "[DEBUG]") || strings.Contains(upperLine, " DEBUG ") {
		level = "DEBUG"
	} else if strings.Contains(upperLine, "[INFO]") || strings.Contains(upperLine, " INFO ") {
		level = "INFO"
	}

	// 调用回调函数
	lw.callback(level, message)
}

// shouldFilterLog 判断是否应该过滤掉这条日志
// 过滤掉频繁出现且无意义的日志，减少日志噪音
func (lw *logWriter) shouldFilterLog(line string) bool {
	// 过滤规则：匹配频繁出现的无意义日志模式
	filterPatterns := []string{
		"proxy/socks: Not Socks request, try to parse as HTTP request",
		"proxy/http: request to Method [CONNECT]",
		"app/dispatcher: default route for",
		"transport/internet/tcp: dialing TCP to",
		"transport/internet: dialing to",
	}

	upperLine := strings.ToUpper(line)
	for _, pattern := range filterPatterns {
		if strings.Contains(upperLine, strings.ToUpper(pattern)) {
			return true
		}
	}

	return false
}

// XrayInstance 封装 xray-core 实例
type XrayInstance struct {
	instance    *core.Instance
	ctx         context.Context
	cancel      context.CancelFunc
	isRunning   bool        // 运行状态
	port        int         // 监听端口
	logWriter   *logWriter  // 日志写入器
	logCallback LogCallback // 日志回调函数
}

// NewXrayInstanceFromJSON 从 JSON 配置创建 xray-core 实例
func NewXrayInstanceFromJSON(configJSON []byte) (*XrayInstance, error) {
	return NewXrayInstanceFromJSONWithCallback(configJSON, nil)
}

// NewXrayInstanceFromJSONWithCallback 从 JSON 配置创建 xray-core 实例，并设置日志回调
func NewXrayInstanceFromJSONWithCallback(configJSON []byte, logCallback LogCallback) (*XrayInstance, error) {
	var config conf.Config
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return nil, fmt.Errorf("Xray: 解析配置失败: %w", err)
	}

	pbConfig, err := config.Build()
	if err != nil {
		return nil, fmt.Errorf("Xray: 构建配置失败: %w", err)
	}

	instance, err := core.New(pbConfig)
	if err != nil {
		return nil, fmt.Errorf("Xray: 创建实例失败: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// 创建日志写入器（虽然当前未直接使用，但保留以备将来扩展）
	logWriter := NewLogWriter(logCallback)

	xi := &XrayInstance{
		instance:    instance,
		ctx:         ctx,
		cancel:      cancel,
		isRunning:   false,
		port:        0,
		logWriter:   logWriter,
		logCallback: logCallback,
	}

	return xi, nil
}

// SetLogCallback 设置日志回调函数
func (xi *XrayInstance) SetLogCallback(callback LogCallback) {
	xi.logCallback = callback
	if xi.logWriter != nil {
		xi.logWriter.SetCallback(callback)
	}
}

// Start 启动 xray-core 实例
func (xi *XrayInstance) Start() error {
	if xi.isRunning {
		return fmt.Errorf("Xray: xray实例已经在运行")
	}
	if err := xi.instance.Start(); err != nil {
		return fmt.Errorf("Xray: 启动失败: %w", err)
	}
	xi.isRunning = true
	return nil
}

// Stop 停止 xray-core 实例
func (xi *XrayInstance) Stop() error {
	if !xi.isRunning {
		return nil // 已经停止，直接返回
	}
	xi.isRunning = false
	xi.cancel()
	if xi.instance != nil {
		xi.instance.Close()
	}
	return nil
}

// IsRunning 检查 xray 实例是否在运行
func (xi *XrayInstance) IsRunning() bool {
	return xi.isRunning && xi.instance != nil
}

// SetPort 设置监听端口
func (xi *XrayInstance) SetPort(port int) {
	xi.port = port
}

// GetPort 获取监听端口
func (xi *XrayInstance) GetPort() int {
	return xi.port
}

// GetInstance 获取底层 xray-core 实例（用于高级操作）
func (xi *XrayInstance) GetInstance() *core.Instance {
	return xi.instance
}

// CreateOutboundFromServer 根据服务器配置创建 xray 出站配置
func CreateOutboundFromServer(server *model.Node) (map[string]interface{}, error) {
	var outbound map[string]interface{}

	switch server.ProtocolType {
	case "socks5":
		// 创建 SOCKS5 出站配置
		socksConfig := map[string]interface{}{
			"auth": "noauth",
			"servers": []map[string]interface{}{
				{
					"address": server.Addr,
					"port":    server.Port,
				},
			},
		}

		if server.Username != "" && server.Password != "" {
			socksConfig["auth"] = "password"
			socksConfig["accounts"] = []map[string]string{
				{
					"user": server.Username,
					"pass": server.Password,
				},
			}
		}

		outbound = map[string]interface{}{
			"tag":      "proxy",
			"protocol": "socks",
			"settings": socksConfig,
		}

	case "vmess":
		// 创建 VMess 出站配置
		vmessConfig := map[string]interface{}{
			"vnext": []map[string]interface{}{
				{
					"address": server.Addr,
					"port":    server.Port,
					"users": []map[string]interface{}{
						{
							"id":       server.VMessUUID,
							"alterId":  server.VMessAlterID,
							"security": getVMessSecurity(server.VMessSecurity),
						},
					},
				},
			},
		}

		// 构建 streamSettings（传输协议配置）
		streamSettings := buildVMessStreamSettings(server)

		outbound = map[string]interface{}{
			"tag":            "proxy",
			"protocol":       "vmess",
			"settings":       vmessConfig,
			"streamSettings": streamSettings,
		}

	case "ss":
		// 创建 Shadowsocks 出站配置
		ssConfig := map[string]interface{}{
			"servers": []map[string]interface{}{
				{
					"address":  server.Addr,
					"port":     server.Port,
					"method":   server.SSMethod,
					"password": server.Password,
				},
			},
		}

		// 构建 streamSettings（传输协议配置）
		streamSettings := buildSSStreamSettings(server)

		outbound = map[string]interface{}{
			"tag":            "proxy",
			"protocol":       "shadowsocks",
			"settings":       ssConfig,
			"streamSettings": streamSettings,
		}

		// 添加插件配置（如果有）
		if server.SSPlugin != "" {
			ssConfig["servers"].([]map[string]interface{})[0]["plugin"] = server.SSPlugin
			if server.SSPluginOpts != "" {
				ssConfig["servers"].([]map[string]interface{})[0]["plugin_opts"] = server.SSPluginOpts
			}
		}

	case "trojan":
		// 创建 Trojan 出站配置
		// 默认使用 TLS
		security := "tls"
		tlsSettings := map[string]interface{}{
			"allowInsecure": server.TrojanAllowInsecure,
		}

		// 设置 SNI
		if server.TrojanSNI != "" {
			tlsSettings["serverName"] = server.TrojanSNI
		}

		// 设置 ALPN
		if server.TrojanAlpn != "" {
			// ALPN 应该是字符串数组
			alpnArray := []string{}
			for _, alpn := range strings.Split(server.TrojanAlpn, ",") {
				if alpn = strings.TrimSpace(alpn); alpn != "" {
					alpnArray = append(alpnArray, alpn)
				}
			}
			if len(alpnArray) > 0 {
				tlsSettings["alpn"] = alpnArray
			}
		}

		streamSettings := map[string]interface{}{
			"security":    security,
			"tlsSettings": tlsSettings,
		}

		trojanConfig := map[string]interface{}{
			"servers": []map[string]interface{}{
				{
					"address":  server.Addr,
					"port":     server.Port,
					"password": server.Password,
				},
			},
		}

		outbound = map[string]interface{}{
			"tag":            "proxy",
			"protocol":       "trojan",
			"settings":       trojanConfig,
			"streamSettings": streamSettings,
		}

	default:
		return nil, fmt.Errorf("Xray: 不支持的协议类型: %s", server.ProtocolType)
	}

	return outbound, nil
}

// getVMessSecurity 获取 VMess 加密方式，默认为 "auto"
func getVMessSecurity(security string) string {
	if security == "" {
		return "auto"
	}
	return security
}

// buildVMessStreamSettings 构建 VMess 传输协议配置
func buildVMessStreamSettings(server *model.Node) map[string]interface{} {
	streamSettings := map[string]interface{}{
		"network": getVMessNetwork(server.VMessNetwork),
	}

	// 根据传输协议类型设置不同的配置
	switch server.VMessNetwork {
	case "ws", "websocket":
		wsSettings := map[string]interface{}{}
		if server.VMessHost != "" {
			wsSettings["host"] = server.VMessHost
		}
		if server.VMessPath != "" {
			wsSettings["path"] = server.VMessPath
		}
		if len(wsSettings) > 0 {
			streamSettings["wsSettings"] = wsSettings
		}

	case "h2", "http":
		h2Settings := map[string]interface{}{}
		if server.VMessHost != "" {
			h2Settings["host"] = []string{server.VMessHost}
		}
		if server.VMessPath != "" {
			h2Settings["path"] = server.VMessPath
		}
		if len(h2Settings) > 0 {
			streamSettings["httpSettings"] = h2Settings
		}

	case "grpc":
		grpcSettings := map[string]interface{}{}
		if server.VMessPath != "" {
			grpcSettings["serviceName"] = server.VMessPath
		}
		if len(grpcSettings) > 0 {
			streamSettings["grpcSettings"] = grpcSettings
		}
	}

	// TLS 配置
	if server.VMessTLS == "tls" {
		tlsSettings := map[string]interface{}{
			"allowInsecure": false,
		}
		if server.VMessHost != "" {
			tlsSettings["serverName"] = server.VMessHost
		}
		streamSettings["security"] = "tls"
		streamSettings["tlsSettings"] = tlsSettings
	}

	return streamSettings
}

// getVMessNetwork 获取 VMess 传输协议，默认为 "tcp"
func getVMessNetwork(network string) string {
	if network == "" {
		return "tcp"
	}
	return network
}

// buildSSStreamSettings 构建 Shadowsocks 传输协议配置
func buildSSStreamSettings(server *model.Node) map[string]interface{} {
	// 默认使用 tcp
	network := "tcp"
	streamSettings := map[string]interface{}{
		"network": network,
	}

	// 目前 Shadowsocks 主要使用 tcp
	// 如果需要更复杂的配置，可以根据实际需求扩展

	return streamSettings
}

// RoutingOptions 路由相关配置（直连列表、直连列表是否走代理等）。
type RoutingOptions struct {
	DirectRoutes        []string // 用户配置的直连列表（domain:xxx 或 ip/cidr）
	DirectRoutesUseProxy bool    // true：直连列表走代理；false：走直连
}

// CreateXrayConfig 创建完整的 xray 配置。
// 参数：
//   - localPort: 本地 SOCKS5 监听端口（默认 10808）
//   - server: 服务器配置，用于创建出站配置
//   - logFilePath: 日志文件路径（可选，为空则不设置）
//   - routing: 路由选项（可选，nil 则仅使用内置规则）
func CreateXrayConfig(localPort int, server *model.Node, logFilePath string, routing *RoutingOptions) ([]byte, error) {
	if localPort == 0 {
		localPort = 10808
	}

	// 创建入站配置（本地 SOCKS5 服务器）
	inbound := map[string]interface{}{
		"tag":      "socks-in",
		"port":     localPort,
		"protocol": "socks",
		"settings": map[string]interface{}{
			"auth": "noauth",
			"udp":  true,
		},
	}

	// 创建出站配置
	outbound, err := CreateOutboundFromServer(server)
	if err != nil {
		return nil, fmt.Errorf("Xray: 创建出站配置失败: %w", err)
	}

	// 创建直连出站配置
	directOutbound := map[string]interface{}{
		"tag":      "direct",
		"protocol": "freedom",
		"settings": map[string]interface{}{},
	}

	// 构建日志配置
	logConfig := map[string]interface{}{
		"loglevel": "warning",
	}
	if logFilePath != "" {
		logConfig["error"] = logFilePath
		logConfig["access"] = logFilePath
	}

	// 构建路由规则（含用户直连列表与是否走代理）
	rules := buildRoutingRules(routing)

	// 构建完整配置
	config := map[string]interface{}{
		"log": logConfig,
		"inbounds": []interface{}{
			inbound,
		},
		"outbounds": []interface{}{
			outbound,
			directOutbound,
		},
		"routing": map[string]interface{}{
			"rules": rules,
			"domainStrategy": "AsIs",
		},
	}

	return json.MarshalIndent(config, "", "  ")
}

// buildRoutingRules 构建路由规则。
// 顺序：本地直连 -> 用户直连列表（根据 directRoutesUseProxy 走直连或代理）-> 国内直连 -> 默认代理。
func buildRoutingRules(routing *RoutingOptions) []interface{} {
	rules := []interface{}{}

	// 1. 本地地址直连
	localRule := map[string]interface{}{
		"type": "field",
		"ip": []string{
			"127.0.0.0/8",
			"10.0.0.0/8",
			"172.16.0.0/12",
			"192.168.0.0/16",
			"fc00::/7",
			"fe80::/10",
		},
		"outboundTag": "direct",
	}
	rules = append(rules, localRule)

	// 2. 用户直连列表：走直连或走代理（直连列表中的地址也可以走代理）
	if routing != nil && len(routing.DirectRoutes) > 0 {
		domains, ips := splitDirectRoutes(routing.DirectRoutes)
		if len(domains) > 0 || len(ips) > 0 {
			r := map[string]interface{}{"type": "field"}
			if len(domains) > 0 {
				r["domain"] = domains
			}
			if len(ips) > 0 {
				r["ip"] = ips
			}
			if routing.DirectRoutesUseProxy {
				r["outboundTag"] = "proxy"
			} else {
				r["outboundTag"] = "direct"
			}
			rules = append(rules, r)
		}
	}

	// 3. 国内域名直连
	chinaRule := map[string]interface{}{
		"type": "field",
		"domain": []string{
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
			"domain:sohu.com",
			"domain:优酷.com",
			"domain:土豆.com",
			"domain:爱奇艺.com",
			"domain:央视.com",
			"domain:小米.com",
			"domain:华为.com",
			"domain:OPPO.com",
			"domain:vivo.com",
			"domain:美团.com",
			"domain:大众点评.com",
			"domain:高德地图.com",
			"domain:携程.com",
			"domain:艺龙.com",
			"domain:同程.com",
			"domain:去哪儿.com",
			"domain:考拉.com",
			"domain:苏宁.com",
			"domain:国美.com.cn",
			"domain:天猫.com",
			"domain:阿里巴巴.com",
			"domain:百度.com",
			"domain:腾讯.com",
			"domain:微信.com",
			"domain:淘宝.com",
			"domain:京东.com",
			"domain:阿里云.com",
			"domain:网易.com",
			"domain:新浪.com",
			"domain:搜狐.com",
		},
		"outboundTag": "direct",
	}
	rules = append(rules, chinaRule)

	// 添加常见的国内 IP 白名单
	chinaIPRule := map[string]interface{}{
		"type": "field",
		"ip": []string{
			"202.102.128.0/16",
			"202.106.0.0/16",
			"202.108.0.0/16",
			"202.112.0.0/16",
			"202.114.0.0/16",
			"202.117.0.0/16",
			"202.118.0.0/16",
			"202.119.0.0/16",
			"202.120.0.0/16",
			"202.121.0.0/16",
			"202.122.0.0/16",
			"202.123.0.0/16",
			"202.124.0.0/16",
			"202.125.0.0/16",
			"202.126.0.0/16",
			"202.127.0.0/16",
			"210.22.0.0/16",
			"210.24.0.0/16",
			"210.25.0.0/16",
			"210.26.0.0/16",
			"210.27.0.0/16",
			"210.28.0.0/16",
			"210.29.0.0/16",
			"210.30.0.0/16",
			"210.31.0.0/16",
			"210.51.0.0/16",
			"210.72.0.0/16",
			"210.73.0.0/16",
			"210.74.0.0/16",
			"210.75.0.0/16",
			"210.76.0.0/16",
			"210.77.0.0/16",
			"210.78.0.0/16",
			"210.79.0.0/16",
			"210.80.0.0/16",
			"210.81.0.0/16",
			"210.82.0.0/16",
			"210.83.0.0/16",
			"210.84.0.0/16",
			"210.85.0.0/16",
			"210.86.0.0/16",
			"210.87.0.0/16",
			"210.88.0.0/16",
			"210.89.0.0/16",
			"210.90.0.0/16",
			"210.91.0.0/16",
			"210.92.0.0/16",
			"210.93.0.0/16",
			"210.94.0.0/16",
			"210.95.0.0/16",
			"210.96.0.0/16",
			"210.97.0.0/16",
			"210.98.0.0/16",
			"210.99.0.0/16",
			"211.64.0.0/16",
			"211.65.0.0/16",
			"211.66.0.0/16",
			"211.67.0.0/16",
			"211.68.0.0/16",
			"211.69.0.0/16",
			"211.70.0.0/16",
			"211.71.0.0/16",
			"211.72.0.0/16",
			"211.73.0.0/16",
			"211.74.0.0/16",
			"211.75.0.0/16",
			"211.76.0.0/16",
			"211.77.0.0/16",
			"211.78.0.0/16",
			"211.79.0.0/16",
			"211.80.0.0/16",
			"211.81.0.0/16",
			"211.82.0.0/16",
			"211.83.0.0/16",
			"211.84.0.0/16",
			"211.85.0.0/16",
			"211.86.0.0/16",
			"211.87.0.0/16",
			"211.88.0.0/16",
			"211.89.0.0/16",
			"211.90.0.0/16",
			"211.91.0.0/16",
			"211.92.0.0/16",
			"211.93.0.0/16",
			"211.94.0.0/16",
			"211.95.0.0/16",
			"211.96.0.0/16",
			"211.97.0.0/16",
			"211.98.0.0/16",
			"211.99.0.0/16",
		},
		"outboundTag": "direct",
	}
	rules = append(rules, chinaIPRule)

	// 4. 默认走代理（匹配所有剩余流量）
	rules = append(rules, map[string]interface{}{
		"type":        "field",
		"outboundTag": "proxy",
	})

	return rules
}

// splitDirectRoutes 将直连规则拆分为 domain 与 ip 列表（xray 规则格式）。
func splitDirectRoutes(routes []string) (domains, ips []string) {
	for _, r := range routes {
		s := strings.TrimSpace(r)
		if s == "" {
			continue
		}
		if strings.HasPrefix(s, "domain:") || strings.HasPrefix(s, "geosite:") ||
			strings.HasPrefix(s, "regexp:") || strings.HasPrefix(s, "full:") {
			domains = append(domains, s)
		} else {
			ips = append(ips, s)
		}
	}
	return domains, ips
}
