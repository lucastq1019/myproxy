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
	if err := initDatabase(); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	defer database.CloseDB()

	appState := ui.NewAppState()
	if err := appState.Startup(); err != nil {
		log.Fatalf("应用启动失败: %v", err)
	}
	appState.Run()
}

func initDatabase() error {
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取工作目录失败: %w", err)
	}

	dbPath := filepath.Join(workDir, "data", "myproxy.db")
	if err := database.InitDB(dbPath); err != nil {
		return fmt.Errorf("初始化数据库失败: %w", err)
	}
	if err := database.InitDefaultConfig(); err != nil {
		log.Printf("初始化默认配置失败: %v", err)
	}

	return nil
}
