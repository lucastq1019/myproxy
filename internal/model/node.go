package model

// Node 表示一个代理服务器的配置信息。
type Node struct {
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
	
	// Trojan 协议字段
	TrojanPassword    string `json:"trojan_password,omitempty"`   // Trojan 密码
	TrojanSNI         string `json:"trojan_sni,omitempty"`        // Trojan SNI
	TrojanAlpn        string `json:"trojan_alpn,omitempty"`       // Trojan ALPN
	TrojanAllowInsecure bool  `json:"trojan_allow_insecure,omitempty"` // Trojan 是否允许不安全连接
	
	// 原始配置 JSON（用于存储完整的协议配置，便于未来扩展）
	RawConfig        string `json:"raw_config,omitempty"`        // 原始配置 JSON 字符串
}

