package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"sysprobe/internal/config"
	"sysprobe/internal/monitor"
	"sysprobe/internal/utils"
	"time"
)

func main() {
	// 載入 config.yml
	cfg, err := config.Load("config.yml")
	if err != nil {
		utils.Log.Error("failed to load config: %v", err)
	}

	// 初始化 logger
	err = utils.InitLogger(utils.LogConfig{
		Path:       cfg.Log.Path,
		MaxSizeMB:  cfg.Log.MaxSizeMB,
		MaxAge:     cfg.Log.MaxAge,
		MaxBackups: cfg.Log.MaxBackups,
	})
	if err != nil {
		utils.Log.Error("failed to init logger: %v", err)
	}
	utils.Log.Info("Config and logger initialized")

	// 載入 Monitor
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	monitor.LoadMonitor(ctx, cfg.Monitor)

	utils.Log.Info("Service initialized successfully")

	// 捕捉 Ctrl+C / SIGTERM，用來優雅關閉
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigs
	utils.Log.Info("Received signal %v, shutting down...", sig)
	cancel()
	time.Sleep(5 * time.Second)
	utils.Log.Info("Service exited gracefully")

}
