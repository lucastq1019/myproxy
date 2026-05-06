package systemproxy

import "fmt"

// TerminalProxyURL 根据设置中的 proxyType 生成写入 HTTP_PROXY 等变量的 URL。
//   - socks5：SOCKS5
//   - http：HTTP 代理（CONNECT，访问 HTTPS 站点时仍用此 URL，scheme 为 http://）
//   - https_tls：客户端与代理之间走 TLS，URL 为 https://（需代理端支持；本地默认入站为明文时请用 http）
//   - 兼容：历史配置里曾用 "https" 表示 HTTP CONNECT，仍映射为 http://
func TerminalProxyURL(host string, port int, proxyType string) string {
	if proxyType == "" {
		proxyType = "socks5"
	}
	scheme := "socks5"
	switch proxyType {
	case "http", "https":
		scheme = "http"
	case "https_tls":
		scheme = "https"
	case "socks5", "socks":
		scheme = "socks5"
	}
	return fmt.Sprintf("%s://%s:%d", scheme, host, port)
}
