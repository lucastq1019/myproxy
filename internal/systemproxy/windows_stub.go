//go:build !windows

package systemproxy

// newWindowsProxy 仅在 Windows 构建中由 windows.go 提供真实实现；其它 GOOS 需占位以满足 platform.go 的编译期引用。
func newWindowsProxy(host string, port int) PlatformProxy {
	_ = host
	_ = port
	return newUnsupportedProxy("windows")
}
