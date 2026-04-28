//go:build windows
// +build windows

package systemproxy

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

// WindowsProxy Windows 平台的代理实现
type WindowsProxy struct {
	proxyHost string
	proxyPort int
}

var (
	wininetDLL             = syscall.NewLazyDLL("wininet.dll")
	user32DLL              = syscall.NewLazyDLL("user32.dll")
	procInternetSetOptionW = wininetDLL.NewProc("InternetSetOptionW")
	procSendMessageTimeout = user32DLL.NewProc("SendMessageTimeoutW")
)

const (
	internetOptionSettingsChanged = 39
	internetOptionRefresh         = 37
	hwndBroadcast                 = 0xffff
	wmSettingChange               = 0x001A
	smtoAbortIfHung               = 0x0002
)

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

	return notifyWindowsProxyChanged()
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

	// 设置代理服务器地址，格式：socks=host:port
	// 注意：在Windows中，需要指定代理类型为socks，否则默认使用HTTP代理
	proxyServer := fmt.Sprintf("socks=%s:%d", host, port)
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

	return notifyWindowsProxyChanged()
}

// SetTerminalProxy 设置终端代理（环境变量代理）
// Windows 可以通过设置用户环境变量实现持久化
func (p *WindowsProxy) SetTerminalProxy(host string, port int, proxyType string) error {
	if proxyType == "" {
		proxyType = "socks5"
	}
	proxyURL := fmt.Sprintf("%s://%s:%d", proxyType, host, port)

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

	return notifyWindowsEnvironmentChanged()
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

	return notifyWindowsEnvironmentChanged()
}

func (p *WindowsProxy) GetCurrentProxyMode() ProxyMode {
	key, err := registry.OpenKey(
		registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Internet Settings`,
		registry.QUERY_VALUE,
	)
	if err == nil {
		defer key.Close()
		if enabled, _, err := key.GetIntegerValue("ProxyEnable"); err == nil && enabled != 0 {
			if proxyServer, _, err := key.GetStringValue("ProxyServer"); err == nil && proxyServer != "" {
				return ProxyModeAuto
			}
		}
	}
	if os.Getenv("HTTP_PROXY") != "" || os.Getenv("http_proxy") != "" {
		return ProxyModeTerminal
	}
	return ProxyModeNone
}

func notifyWindowsProxyChanged() error {
	if err := internetSetOption(internetOptionSettingsChanged); err != nil {
		return err
	}
	if err := internetSetOption(internetOptionRefresh); err != nil {
		return err
	}
	if err := broadcastSettingChange("Software\\Microsoft\\Windows\\CurrentVersion\\Internet Settings"); err != nil {
		return err
	}
	return nil
}

func notifyWindowsEnvironmentChanged() error {
	return broadcastSettingChange("Environment")
}

func internetSetOption(option uintptr) error {
	ret, _, callErr := procInternetSetOptionW.Call(0, option, 0, 0)
	if ret == 0 {
		if callErr != syscall.Errno(0) {
			return fmt.Errorf("刷新 Windows 代理设置失败: %v", callErr)
		}
		return fmt.Errorf("刷新 Windows 代理设置失败")
	}
	return nil
}

func broadcastSettingChange(target string) error {
	targetPtr, err := syscall.UTF16PtrFromString(target)
	if err != nil {
		return fmt.Errorf("编码 Windows 设置变更消息失败: %w", err)
	}
	ret, _, callErr := procSendMessageTimeout.Call(
		uintptr(hwndBroadcast),
		uintptr(wmSettingChange),
		0,
		uintptr(unsafe.Pointer(targetPtr)),
		uintptr(smtoAbortIfHung),
		uintptr(5000),
		0,
	)
	if ret == 0 {
		if callErr != syscall.Errno(0) {
			return fmt.Errorf("广播 Windows 设置变更失败: %v", callErr)
		}
		return fmt.Errorf("广播 Windows 设置变更失败")
	}
	return nil
}
