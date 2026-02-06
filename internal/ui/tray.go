package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

// TrayManager 管理系统托盘
type TrayManager struct {
	appState           *AppState
	app                fyne.App
	window             fyne.Window
	proxyModeMenuItems [2]*fyne.MenuItem // 系统代理模式菜单项（清除、系统）
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
		tm.createTrayMenu(desk)
		fmt.Println("托盘菜单已设置")
	} else {
		// 如果不支持桌面扩展，记录警告
		fmt.Println("错误: 应用不支持桌面扩展，无法显示系统托盘")
		if tm.appState.Logger != nil {
			tm.appState.Logger.Error("应用不支持桌面扩展，无法显示系统托盘")
		}
	}
}

// createTrayMenu 创建托盘菜单
func (tm *TrayManager) createTrayMenu(desk desktop.App) {
	// 创建系统代理模式菜单项（如果尚未创建）
	if tm.proxyModeMenuItems[0] == nil {
		tm.proxyModeMenuItems[0] = fyne.NewMenuItem(SystemProxyModeClear.ShortString(), func() {
			if tm.appState != nil && tm.appState.MainWindow != nil {
				_ = tm.appState.MainWindow.SetSystemProxyMode(SystemProxyModeClear)
				// SetSystemProxyMode 内部会调用 RefreshProxyModeMenu，这里不需要再次调用
			}
		})
		tm.proxyModeMenuItems[1] = fyne.NewMenuItem(SystemProxyModeAuto.ShortString(), func() {
			if tm.appState != nil && tm.appState.MainWindow != nil {
				_ = tm.appState.MainWindow.SetSystemProxyMode(SystemProxyModeAuto)
				// SetSystemProxyMode 内部会调用 RefreshProxyModeMenu，这里不需要再次调用
			}
		})
	}

	// 更新菜单项的选中状态
	tm.updateProxyModeMenuCheckedState()

	// 创建托盘菜单
	menu := fyne.NewMenu("SOCKS5 代理客户端",
		fyne.NewMenuItem("显示窗口", func() {
			tm.window.Show()
			tm.window.RequestFocus()
		}),
		fyne.NewMenuItemSeparator(),
		tm.proxyModeMenuItems[0], // 清除代理
		tm.proxyModeMenuItems[1], // 系统代理
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("退出", func() {
			tm.quit()
		}),
	)

	// 设置托盘菜单
	desk.SetSystemTrayMenu(menu)
}

// RefreshProxyModeMenu 刷新系统代理模式菜单的选中状态（公共方法）
func (tm *TrayManager) RefreshProxyModeMenu() {
	tm.refreshProxyModeMenu()
}

// updateProxyModeMenuCheckedState 更新菜单项的选中状态（不刷新菜单）
func (tm *TrayManager) updateProxyModeMenuCheckedState() {
	if tm.appState == nil || tm.appState.MainWindow == nil {
		return
	}

	// 获取当前系统代理模式
	currentMode := tm.appState.MainWindow.GetCurrentSystemProxyMode()

	// 更新菜单项的选中状态
	for i, item := range tm.proxyModeMenuItems {
		if item == nil {
			continue
		}
		switch i {
		case 0: // 清除代理
			item.Checked = (currentMode == SystemProxyModeClear)
		case 1: // 系统代理
			item.Checked = (currentMode == SystemProxyModeAuto)
		}
	}
}

// refreshProxyModeMenu 刷新系统代理模式菜单的选中状态（内部方法）
func (tm *TrayManager) refreshProxyModeMenu() {
	if tm.appState == nil || tm.appState.MainWindow == nil {
		return
	}

	// 获取当前系统代理模式
	currentMode := tm.appState.MainWindow.GetCurrentSystemProxyMode()

	// 检查是否有状态变化
	needRefresh := false
	for i, item := range tm.proxyModeMenuItems {
		if item == nil {
			continue
		}
		var shouldBeChecked bool
		switch i {
		case 0: // 清除代理
			shouldBeChecked = (currentMode == SystemProxyModeClear)
		case 1: // 系统代理
			shouldBeChecked = (currentMode == SystemProxyModeAuto)
		}
		if item.Checked != shouldBeChecked {
			needRefresh = true
			break // 发现变化就退出循环
		}
	}

	// 只有在状态变化时才刷新托盘菜单（需要重新设置菜单才能更新选中状态）
	if needRefresh {
		if desk, ok := tm.app.(desktop.App); ok {
			tm.createTrayMenu(desk)
		}
	}
}

// quit 退出应用
func (tm *TrayManager) quit() {
	// 停止日志监控
	if tm.appState.MainWindow != nil && tm.appState.MainWindow.logsPanel != nil {
		tm.appState.MainWindow.logsPanel.Stop()
	}

	// 保存布局配置
	if tm.appState.MainWindow != nil {
		tm.appState.MainWindow.SaveLayoutConfig()
	}

	// 退出应用
	tm.app.Quit()
}
