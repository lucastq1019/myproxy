package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

// TrayManager 管理系统托盘
type TrayManager struct {
	appState *AppState
	app      fyne.App
	window   fyne.Window
}

// NewTrayManager 创建系统托盘管理器
func NewTrayManager(appState *AppState) *TrayManager {
	return &TrayManager{
		appState: appState,
		app:      appState.App,
		window:   appState.Window,
	}
}

// SetupTray 设置系统托盘（使用 Fyne 原生系统托盘 API）
func (tm *TrayManager) SetupTray() {
	// 检查应用是否支持桌面扩展（系统托盘需要）
	if desk, ok := tm.app.(desktop.App); ok {
		fmt.Println("应用支持桌面扩展，开始设置托盘图标...")
		
		// 创建托盘图标
		icon := createTrayIconResource(tm.appState)
		if icon == nil {
			fmt.Println("警告: 创建托盘图标失败")
			return
		}
		fmt.Println("托盘图标创建成功")
		
		// 设置托盘图标
		desk.SetSystemTrayIcon(icon)
		fmt.Println("托盘图标已设置")
		
		// 创建托盘菜单
		menu := fyne.NewMenu("SOCKS5 代理客户端",
			fyne.NewMenuItem("显示窗口", func() {
				tm.window.Show()
				tm.window.RequestFocus()
			}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("退出", func() {
				tm.quit()
			}),
		)
		
		// 设置托盘菜单
		desk.SetSystemTrayMenu(menu)
		fmt.Println("托盘菜单已设置")
	} else {
		// 如果不支持桌面扩展，记录警告
		fmt.Println("错误: 应用不支持桌面扩展，无法显示系统托盘")
		if tm.appState.Logger != nil {
			tm.appState.Logger.Error("应用不支持桌面扩展，无法显示系统托盘")
		}
	}
}


// quit 退出应用
func (tm *TrayManager) quit() {
	// 停止日志监控
	if tm.appState.LogsPanel != nil {
		tm.appState.LogsPanel.Stop()
	}
	
	// 保存布局配置
	if tm.appState.MainWindow != nil {
		tm.appState.MainWindow.SaveLayoutConfig()
	}
	
	// 退出应用
	tm.app.Quit()
}
