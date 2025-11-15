package network

import (
	"context"
	"sysprobe/internal/config"
	"sysprobe/internal/utils"
	"time"
)

func Start(ctx context.Context, cfg config.MonitorModule) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				utils.Log.Error("[Network] goroutine panic: %v", r)
				// 可以選擇重新啟動 goroutine
				Start(ctx, cfg)
			}
		}()

		ticker := time.NewTicker(time.Duration(cfg.Interval) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				utils.Log.Info("[Network] 收集網路流量中...")
			case <-ctx.Done():
				utils.Log.Info("[Network] 收集器已停止")
				return
			}
		}
	}()
}
