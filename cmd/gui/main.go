package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"myproxy.com/p/internal/config"
	"myproxy.com/p/internal/database"
	"myproxy.com/p/internal/logging"
	"myproxy.com/p/internal/ui"
)

func main() {
	// 配置文件路径（用于向后兼容，实际配置存储在数据库）
	configPath := "./config.json"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	// 初始化数据库（必须在加载配置之前初始化）
	dbPath := filepath.Join(filepath.Dir(configPath), "data", "myproxy.db")
	if err := database.InitDB(dbPath); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	defer database.CloseDB()

	// 从数据库加载配置，如果不存在则从 JSON 文件加载并迁移
	cfg, err := loadConfigFromDB(configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 创建应用状态（先创建，logger稍后设置）
	appState := ui.NewAppState(cfg, nil)

	// 启动时从数据库同步服务器列表，保证 UI 有数据
	appState.LoadServersFromDB()

	// 初始化应用（创建Fyne应用和窗口）
	appState.InitApp()

	// 创建主窗口（此时LogsPanel已创建）
	mainWindow := ui.NewMainWindow(appState)

	// 创建日志回调函数，用于实时更新UI（确保日志文件写入和UI显示一致）
	logCallback := func(level, logType, message, logLine string) {
		if appState.LogsPanel != nil {
			// 直接使用完整的日志行，确保格式与文件中的格式完全一致
			appState.LogsPanel.AppendLogLine(logLine)
		}
	}

	// 初始化日志（使用数据库中的配置），并设置UI回调
	logger, err := logging.NewLogger(cfg.LogFile, cfg.LogLevel == "debug", cfg.LogLevel, logCallback)
	if err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}

	// 设置logger到appState
	appState.Logger = logger

	// Logger初始化后，启动日志文件监控（用于监控xray日志等直接从文件写入的日志）
	if appState.LogsPanel != nil {
		appState.LogsPanel.StartLogFileWatcher()
	}

	// 设置窗口内容
	content := mainWindow.Build()
	if content != nil {
		appState.Window.SetContent(content)
	}

	// 创建并设置系统托盘（使用 Fyne 原生 API，不需要单独的 goroutine）
	trayManager := ui.NewTrayManager(appState)
	fmt.Println("开始设置系统托盘...")
	trayManager.SetupTray()
	fmt.Println("系统托盘设置完成")

	// 设置窗口关闭事件，隐藏到托盘而不是退出
	appState.Window.SetCloseIntercept(func() {
		// 保存窗口大小到数据库
		if appState.Window != nil && appState.Window.Canvas() != nil {
			ui.SaveWindowSize(appState.Window.Canvas().Size())
		}
		// 保存布局配置到数据库
		mainWindow.SaveLayoutConfig()
		// 保存应用配置到数据库
		if err := saveConfigToDB(cfg); err != nil {
			log.Printf("保存配置到数据库失败: %v", err)
		}
		// 隐藏窗口而不是关闭（Fyne 会自动处理 Dock 图标点击显示窗口）
		appState.Window.Hide()
	})
	fmt.Println("设置窗口关闭事件")

	// 显示窗口并运行应用
	appState.Window.Show()
	appState.App.Run()
	fmt.Println("应用运行结束")
}

// loadConfigFromDB 从数据库加载配置，如果不存在则从 JSON 文件加载并迁移到数据库。
func loadConfigFromDB(configPath string) (*config.Config, error) {
	// 尝试从数据库加载配置
	logLevel, _ := database.GetAppConfigWithDefault("logLevel", "")
	logFile, _ := database.GetAppConfigWithDefault("logFile", "")
	autoProxyEnabledStr, _ := database.GetAppConfigWithDefault("autoProxyEnabled", "")
	autoProxyPortStr, _ := database.GetAppConfigWithDefault("autoProxyPort", "")

	// 如果数据库中有配置，使用数据库配置
	if logLevel != "" || logFile != "" || autoProxyEnabledStr != "" || autoProxyPortStr != "" {
		cfg := config.DefaultConfig()
		if logLevel != "" {
			cfg.LogLevel = logLevel
		}
		if logFile != "" {
			cfg.LogFile = logFile
		}
		if autoProxyEnabledStr != "" {
			if enabled, err := strconv.ParseBool(autoProxyEnabledStr); err == nil {
				cfg.AutoProxyEnabled = enabled
			}
		}
		if autoProxyPortStr != "" {
			// 允许任意端口值，不进行有效性检查（用户可根据实际情况选择端口）
			if port, err := strconv.Atoi(autoProxyPortStr); err == nil {
				cfg.AutoProxyPort = port
			}
		}
		return cfg, nil
	}

	// 数据库中没有配置，从 JSON 文件加载（向后兼容）
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, err
	}

	// 将配置迁移到数据库
	if err := saveConfigToDB(cfg); err != nil {
		log.Printf("迁移配置到数据库失败: %v", err)
		// 即使迁移失败，也返回配置，保证应用可以启动
	}

	return cfg, nil
}

// saveConfigToDB 将配置保存到数据库。
func saveConfigToDB(cfg *config.Config) error {
	if err := database.SetAppConfig("logLevel", cfg.LogLevel); err != nil {
		return err
	}
	if err := database.SetAppConfig("logFile", cfg.LogFile); err != nil {
		return err
	}
	if err := database.SetAppConfig("autoProxyEnabled", strconv.FormatBool(cfg.AutoProxyEnabled)); err != nil {
		return err
	}
	if err := database.SetAppConfig("autoProxyPort", strconv.Itoa(cfg.AutoProxyPort)); err != nil {
		return err
	}
	return nil
}
