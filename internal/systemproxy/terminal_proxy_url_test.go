package systemproxy

import "testing"

func TestTerminalProxyURL(t *testing.T) {
	tests := []struct {
		proxyType string
		want      string
	}{
		{"", "socks5://127.0.0.1:10808"},
		{"socks5", "socks5://127.0.0.1:10808"},
		{"socks", "socks5://127.0.0.1:10808"},
		{"https", "http://127.0.0.1:10808"},
		{"http", "http://127.0.0.1:10808"},
		{"https_tls", "https://127.0.0.1:10808"},
		{"unknown", "socks5://127.0.0.1:10808"},
	}
	for _, tt := range tests {
		if got := TerminalProxyURL("127.0.0.1", 10808, tt.proxyType); got != tt.want {
			t.Errorf("TerminalProxyURL(..., %q) = %q, want %q", tt.proxyType, got, tt.want)
		}
	}
}
