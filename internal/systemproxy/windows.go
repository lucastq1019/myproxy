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
	// 终端环境变量代理使用的 NO_PROXY，避免环回与本地服务误走代理
	windowsTerminalNoProxy = "localhost,127.0.0.1,::1"
	// windowsProxyOverrideDefault 与 v2rayN 等工具常用绕过列表一致（分号分隔），含 <local> 以匹配「不对本地地址使用代理」。
	// 参考：2dust/v2rayN ServiceLib/Handler/SysProxy 及 WinINet INTERNET_PER_CONN_PROXY_BYPASS。
	windowsProxyOverrideDefault = "localhost;127.*;10.*;172.16.*;172.17.*;172.18.*;172.19.*;172.20.*;172.21.*;172.22.*;172.23.*;172.24.*;172.25.*;172.26.*;172.27.*;172.28.*;172.29.*;172.30.*;172.31.*;192.168.*;<local>"
)

// windowsProxyServerRegistry 生成 WinINet/IE 注册表 ProxyServer 字符串。
// 须使用「协议=http://主机:端口」形式；仅写 http=host:port 时部分环境会解析失败，表现为仍沿用 v2rayN 等先前写入的配置。
func windowsProxyServerRegistry(host string, port int) string {
	return fmt.Sprintf("http=http://%s:%d;https=http://%s:%d;socks=%s:%d", host, port, host, port, host, port)
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

	// 禁用代理，并清空与 v2rayN 等工具共用的键，避免仅关 ProxyEnable 仍残留旧 ProxyServer/PAC
	if err := key.SetDWordValue("ProxyEnable", 0); err != nil {
		return fmt.Errorf("禁用代理失败: %v", err)
	}
	if err := key.SetStringValue("ProxyServer", ""); err != nil {
		return fmt.Errorf("清除 ProxyServer 失败: %v", err)
	}
	if err := key.SetStringValue("ProxyOverride", ""); err != nil {
		return fmt.Errorf("清除 ProxyOverride 失败: %v", err)
	}
	if err := key.DeleteValue("AutoConfigURL"); err != nil && !isRegistryNotExist(err) {
		return fmt.Errorf("清除 AutoConfigURL 失败: %v", err)
	}

	return notifyWindowsProxyChanged()
}

func isRegistryNotExist(err error) bool {
	if err == nil {
		return false
	}
	return err == registry.ErrNotExist
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

	// 先关闭 PAC，否则 AutoConfigURL 与手动代理并存时行为依赖系统实现，易表现为仍走旧配置（如与 v2rayN 混用后）
	if err := key.DeleteValue("AutoConfigURL"); err != nil && !isRegistryNotExist(err) {
		return fmt.Errorf("清除 AutoConfigURL 失败: %v", err)
	}

	proxyServer := windowsProxyServerRegistry(host, port)
	if err := key.SetStringValue("ProxyServer", proxyServer); err != nil {
		return fmt.Errorf("设置代理服务器地址失败: %v", err)
	}

	if err := key.SetStringValue("ProxyOverride", windowsProxyOverrideDefault); err != nil {
		return fmt.Errorf("设置 ProxyOverride 失败: %v", err)
	}

	// 最后启用，避免中间状态读到不完整配置
	if err := key.SetDWordValue("ProxyEnable", 1); err != nil {
		return fmt.Errorf("启用代理失败: %v", err)
	}

	return notifyWindowsProxyChanged()
}

// SetTerminalProxy 设置终端代理（环境变量代理）
// Windows 可以通过设置用户环境变量实现持久化
func (p *WindowsProxy) SetTerminalProxy(host string, port int, proxyType string) error {
	proxyURL := TerminalProxyURL(host, port, proxyType)

	// 1. 设置当前进程环境变量（立即生效）
	os.Setenv("HTTP_PROXY", proxyURL)
	os.Setenv("HTTPS_PROXY", proxyURL)
	os.Setenv("http_proxy", proxyURL)
	os.Setenv("https_proxy", proxyURL)
	os.Setenv("ALL_PROXY", proxyURL)
	os.Setenv("all_proxy", proxyURL)
	os.Setenv("NO_PROXY", windowsTerminalNoProxy)
	os.Setenv("no_proxy", windowsTerminalNoProxy)

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
	_ = envKey.SetStringValue("NO_PROXY", windowsTerminalNoProxy)
	_ = envKey.SetStringValue("no_proxy", windowsTerminalNoProxy)

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
	os.Unsetenv("NO_PROXY")
	os.Unsetenv("no_proxy")

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
	_ = envKey.DeleteValue("NO_PROXY")
	_ = envKey.DeleteValue("no_proxy")

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
	var result uintptr
	ret, _, callErr := procSendMessageTimeout.Call(
		uintptr(hwndBroadcast),
		uintptr(wmSettingChange),
		0,
		uintptr(unsafe.Pointer(targetPtr)),
		uintptr(smtoAbortIfHung),
		uintptr(5000),
		uintptr(unsafe.Pointer(&result)),
	)
	if ret == 0 {
		if callErr != syscall.Errno(0) {
			return fmt.Errorf("广播 Windows 设置变更失败: %v", callErr)
		}
		return fmt.Errorf("广播 Windows 设置变更失败")
	}
	return nil
}
