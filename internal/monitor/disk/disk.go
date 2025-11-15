package disk

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
				utils.Log.Error("[Disk] goroutine panic: %v", r)
				// 可以選擇重新啟動 goroutine
				Start(ctx, cfg)
			}
		}()

		ticker := time.NewTicker(time.Duration(cfg.Interval) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				utils.Log.Debug("[Disk] 收集硬碟資訊中...")
			case <-ctx.Done():
				utils.Log.Debug("[Disk] 收集器已停止")
				return
			}
		}
	}()
}
