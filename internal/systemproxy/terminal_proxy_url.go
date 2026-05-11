package systemproxy

import "fmt"

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
