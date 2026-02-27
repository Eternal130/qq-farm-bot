package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"qq-farm-bot/internal/api"
	"qq-farm-bot/internal/bot"
	"qq-farm-bot/internal/config"
	"qq-farm-bot/internal/store"
)

//go:embed all:dist
var embeddedFrontend embed.FS

func main() {
	// Determine base directory
	exe, _ := os.Executable()
	baseDir := filepath.Dir(exe)
	if wd, err := os.Getwd(); err == nil {
		baseDir = wd
	}

	// Load config
	configPath := filepath.Join(baseDir, "config.json")
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		os.Exit(1)
	}
	cfg.ResolvePaths(baseDir)

	// Save default config if not exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg.Save(configPath)
		fmt.Printf("已生成默认配置文件: %s\n", configPath)
	}

	// Init game config
	bot.LoadGameConfig(cfg.GameConfigDir)

	// Init database
	s, err := store.New(cfg.DBPath)
	if err != nil {
		fmt.Printf("初始化数据库失败: %v\n", err)
		os.Exit(1)
	}
	defer s.Close()

	// Clean old logs (keep 7 days)
	s.CleanOldLogs(7)

	// Init bot manager
	mgr := bot.NewManager(s, cfg)

	// Auto start bots
	mgr.AutoStart()

	// Prepare embedded frontend FS (strip "dist" prefix)
	frontendFS, err := fs.Sub(embeddedFrontend, "dist")
	if err != nil {
		fmt.Printf("加载前端资源失败: %v\n", err)
		os.Exit(1)
	}

	// Setup HTTP server
	router := api.SetupRouter(cfg, s, mgr, frontendFS)

	fmt.Printf("========================================\n")
	fmt.Printf("  QQ农场管理后台\n")
	fmt.Printf("  监听地址: %s\n", cfg.Listen)
	fmt.Printf("  管理账号: %s\n", cfg.AdminUser)
	fmt.Printf("  数据目录: %s\n", cfg.DataDir)
	fmt.Printf("========================================\n")

	// Graceful shutdown
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		fmt.Println("\n正在停止所有 Bot...")
		mgr.StopAll()
		os.Exit(0)
	}()

	if err := router.Run(cfg.Listen); err != nil {
		fmt.Printf("HTTP 服务启动失败: %v\n", err)
		os.Exit(1)
	}
}
