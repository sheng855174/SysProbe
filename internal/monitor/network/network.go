package network

import (
	"context"
	"sysprobe/internal/utils"
	"time"
)

func Start(ctx context.Context) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				utils.Log.Error("[Network] goroutine panic: %v", r)
				// 可以選擇重新啟動 goroutine
				Start(ctx)
			}
		}()

		ticker := time.NewTicker(7 * time.Second)
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
