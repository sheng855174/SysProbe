package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"sysprobe/internal/config"
	"sysprobe/internal/monitor"
	"sysprobe/internal/service"
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
	err = utils.InitLogger(cfg.Log)
	if err != nil {
		utils.Log.Error("failed to init logger: %v", err)
	}
	utils.Log.Info("Config and logger initialized")

	// init uuid
	uuidInfo, _ := utils.InitUUID(cfg.Monitor.Data)
	utils.Log.Info("UUID: %s", uuidInfo.UUID)

	// 建立 contex
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 取得 HostInf
	service.StartHostUpdater(ctx, 15*time.Minute)

	// 載入 Monitor
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
