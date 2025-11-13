package cpu

import (
	"context"
	"sysprobe/internal/utils"
	"time"
)

func Start(ctx context.Context) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				utils.Log.Error("[CPU] goroutine panic: %v", r)
				// 可以選擇重新啟動 goroutine
				Start(ctx)
			}
		}()

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				utils.Log.Info("[CPU] 收集 CPU 使用率中...")
			case <-ctx.Done():
				utils.Log.Info("[CPU] 收集器已停止")
				return
			}
		}
	}()
}
