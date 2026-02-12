package ui

import (
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

// getSystemProxyModeFromAppState 从 AppState（ConfigService）读取系统代理模式，与主窗口共用同一数据源。
func getSystemProxyModeFromAppState(a *AppState) SystemProxyMode {
	if a == nil || a.ConfigService == nil {
		return SystemProxyModeClear
	}
	s := a.ConfigService.GetSystemProxyMode()
	if s == "" {
		return SystemProxyModeClear
	}
	return ParseSystemProxyMode(s)
}

// SetupTray 设置系统托盘（使用 Fyne 原生系统托盘 API）
func (tm *TrayManager) SetupTray() {
	if desk, ok := tm.app.(desktop.App); ok {
		icon := createTrayIconResource(tm.appState)
		if icon == nil {
			tm.appState.SafeLogger.Warn("创建托盘图标失败")
			return
		}
		desk.SetSystemTrayIcon(icon)
		tm.createTrayMenu(desk)
	} else {
		tm.appState.SafeLogger.Warn("应用不支持桌面扩展，无法显示系统托盘")
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

	// 创建关闭代理菜单项
	closeProxyMenuItem := fyne.NewMenuItem("关闭代理", func() {
		if tm.appState != nil && tm.appState.MainWindow != nil {
			// 停止Xray实例
			tm.appState.MainWindow.StopProxy()
			// 清除系统代理
			if tm.appState.MainWindow != nil {
				_ = tm.appState.MainWindow.SetSystemProxyMode(SystemProxyModeClear)
			}
		}
	})

	// 创建托盘菜单
	menu := fyne.NewMenu("SOCKS5 代理客户端",
		fyne.NewMenuItem("显示窗口", func() {
			tm.window.Show()
			tm.window.RequestFocus()
		}),
		fyne.NewMenuItemSeparator(),
		closeProxyMenuItem, // 关闭代理（停止Xray）
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

// updateProxyModeMenuCheckedState 从 AppState（ConfigService）读取系统代理模式，更新菜单选中状态。
func (tm *TrayManager) updateProxyModeMenuCheckedState() {
	if tm.appState == nil || tm.appState.ConfigService == nil {
		return
	}
	currentMode := getSystemProxyModeFromAppState(tm.appState)

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

// refreshProxyModeMenu 根据 AppState 当前状态刷新托盘代理模式菜单。
func (tm *TrayManager) refreshProxyModeMenu() {
	if tm.appState == nil || tm.appState.ConfigService == nil {
		return
	}
	currentMode := getSystemProxyModeFromAppState(tm.appState)

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
