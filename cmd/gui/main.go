package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"myproxy.com/p/internal/database"
	"myproxy.com/p/internal/ui"
)

func main() {
	// 初始化数据库
	if err := initDatabase(); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	defer database.CloseDB()

	// 创建应用工厂
	factory := ui.NewApplicationFactory()

	// 创建应用状态
	appState, err := factory.CreateAppState()
	if err != nil {
		log.Fatalf("创建应用状态失败: %v", err)
	}

	// 统一初始化应用（包括 Fyne 应用、主窗口、日志、托盘等）
	if err := factory.InitializeApplication(appState); err != nil {
		log.Fatalf("应用启动失败: %v", err)
	}

	// 显示窗口并运行应用
	appState.Run()
}

// initDatabase 初始化数据库和默认配置
func initDatabase() error {
	// 获取工作目录（用于确定数据库路径）
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取工作目录失败: %w", err)
	}

	// 初始化数据库（使用硬编码路径，避免暴露配置）
	dbPath := filepath.Join(workDir, "data", "myproxy.db")
	if err := database.InitDB(dbPath); err != nil {
		return fmt.Errorf("初始化数据库失败: %w", err)
	}

	// 初始化默认配置（如果不存在则写入硬编码默认值）
	if err := database.InitDefaultConfig(); err != nil {
		log.Printf("初始化默认配置失败: %v", err)
		// 继续执行，不影响应用启动
	}

	return nil
}
