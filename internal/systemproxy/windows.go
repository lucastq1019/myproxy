//go:build windows
// +build windows

package systemproxy

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows/registry"
)

// WindowsProxy Windows 平台的代理实现
type WindowsProxy struct {
	proxyHost string
	proxyPort int
}

func newWindowsProxy(host string, port int) *WindowsProxy {
	return &WindowsProxy{
		proxyHost: host,
		proxyPort: port,
	}
}

// ClearSystemProxy 清除 Windows 系统代理设置
// 通过修改注册表实现：HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Internet Settings
func (p *WindowsProxy) ClearSystemProxy() error {
	key, err := registry.OpenKey(
		registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Internet Settings`,
		registry.SET_VALUE,
	)
	if err != nil {
		return fmt.Errorf("打开注册表失败: %v", err)
	}
	defer key.Close()

	// 禁用代理
	if err := key.SetDWordValue("ProxyEnable", 0); err != nil {
		return fmt.Errorf("禁用代理失败: %v", err)
	}

	// 清除代理服务器地址（可选，保留原值也可以）
	// key.DeleteValue("ProxyServer")

	// 通知系统设置已更改（需要发送 WM_SETTINGCHANGE 消息）
	// 在 Go 中可以通过调用 Windows API 实现，但这里简化处理
	// 用户可能需要刷新网络设置或重启浏览器

	return nil
}

// SetSystemProxy 设置 Windows 系统代理
// 通过修改注册表实现
func (p *WindowsProxy) SetSystemProxy(host string, port int) error {
	key, err := registry.OpenKey(
		registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Internet Settings`,
		registry.SET_VALUE,
	)
	if err != nil {
		return fmt.Errorf("打开注册表失败: %v", err)
	}
	defer key.Close()

	// 设置代理服务器地址，格式：host:port
	proxyServer := fmt.Sprintf("%s:%d", host, port)
	if err := key.SetStringValue("ProxyServer", proxyServer); err != nil {
		return fmt.Errorf("设置代理服务器地址失败: %v", err)
	}

	// 启用代理
	if err := key.SetDWordValue("ProxyEnable", 1); err != nil {
		return fmt.Errorf("启用代理失败: %v", err)
	}

	// 设置代理覆盖列表（本地地址不使用代理）
	// 默认值：<local> 表示本地地址不使用代理
	proxyOverride := "<local>"
	if err := key.SetStringValue("ProxyOverride", proxyOverride); err != nil {
		// 这个错误可以忽略，不是必须的
		_ = err
	}

	// 注意：修改注册表后，需要通知系统设置已更改
	// 在 Go 中可以通过调用 Windows API 发送 WM_SETTINGCHANGE 消息
	// 但这里简化处理，用户可能需要刷新网络设置或重启浏览器

	return nil
}

// SetTerminalProxy 设置终端代理（环境变量代理）
// Windows 可以通过设置用户环境变量实现持久化
func (p *WindowsProxy) SetTerminalProxy(host string, port int) error {
	proxyURL := fmt.Sprintf("socks5://%s:%d", host, port)

	// 1. 设置当前进程环境变量（立即生效）
	os.Setenv("HTTP_PROXY", proxyURL)
	os.Setenv("HTTPS_PROXY", proxyURL)
	os.Setenv("http_proxy", proxyURL)
	os.Setenv("https_proxy", proxyURL)
	os.Setenv("ALL_PROXY", proxyURL)
	os.Setenv("all_proxy", proxyURL)

	// 2. 设置用户环境变量（持久化）
	// 通过注册表设置用户环境变量：HKEY_CURRENT_USER\Environment
	envKey, err := registry.OpenKey(
		registry.CURRENT_USER,
		"Environment",
		registry.SET_VALUE,
	)
	if err != nil {
		// 如果无法打开注册表，只设置当前进程环境变量
		return nil
	}
	defer envKey.Close()

	// 设置用户环境变量（持久化）
	_ = envKey.SetStringValue("HTTP_PROXY", proxyURL)
	_ = envKey.SetStringValue("HTTPS_PROXY", proxyURL)
	_ = envKey.SetStringValue("http_proxy", proxyURL)
	_ = envKey.SetStringValue("https_proxy", proxyURL)
	_ = envKey.SetStringValue("ALL_PROXY", proxyURL)
	_ = envKey.SetStringValue("all_proxy", proxyURL)

	// 注意：修改用户环境变量后，需要广播 WM_SETTINGCHANGE 消息
	// 新打开的终端/程序会自动读取新的环境变量
	// 当前已打开的终端需要重新加载环境变量

	return nil
}

// ClearTerminalProxy 清除终端代理设置
func (p *WindowsProxy) ClearTerminalProxy() error {
	// 1. 清除当前进程环境变量
	os.Unsetenv("HTTP_PROXY")
	os.Unsetenv("HTTPS_PROXY")
	os.Unsetenv("http_proxy")
	os.Unsetenv("https_proxy")
	os.Unsetenv("ALL_PROXY")
	os.Unsetenv("all_proxy")

	// 2. 从用户环境变量中删除（持久化清除）
	envKey, err := registry.OpenKey(
		registry.CURRENT_USER,
		"Environment",
		registry.SET_VALUE,
	)
	if err != nil {
		// 如果无法打开注册表，只清除当前进程环境变量
		return nil
	}
	defer envKey.Close()

	// 删除用户环境变量
	_ = envKey.DeleteValue("HTTP_PROXY")
	_ = envKey.DeleteValue("HTTPS_PROXY")
	_ = envKey.DeleteValue("http_proxy")
	_ = envKey.DeleteValue("https_proxy")
	_ = envKey.DeleteValue("ALL_PROXY")
	_ = envKey.DeleteValue("all_proxy")

	return nil
}

func (p *WindowsProxy) GetCurrentProxyMode() ProxyMode {
	if os.Getenv("HTTP_PROXY") != "" || os.Getenv("http_proxy") != "" {
		return ProxyModeTerminal
	}
	return ProxyModeNone
}
