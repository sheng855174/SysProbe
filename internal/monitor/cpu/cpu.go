package cpu

import (
	"context"
	"runtime"
	"sysprobe/internal/config"
	"sysprobe/internal/utils"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/load"
)

func Start(ctx context.Context, cfg config.MonitorModule) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				utils.Log.Error("[CPU] goroutine panic: %v", r)
				// 可以選擇重新啟動 goroutine
				Start(ctx, cfg)
			}
		}()

		ticker := time.NewTicker(time.Duration(cfg.Interval) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				utils.Log.Info("[CPU] 收集 CPU 使用率中...")
				monitorCPU()
			case <-ctx.Done():
				utils.Log.Info("[CPU] 收集器已停止")
				return
			}
		}
	}()
}

func monitorCPU() {
	// --- CPU 核心數 ---
	counts, err := cpu.Counts(true)
	if err != nil {
		utils.Log.Error("failed to cpu: %v", err)
	}
	utils.Log.Info("CPU 核心數(含超執行緒): %d", counts)

	// --- CPU 型號資訊 ---
	info, err := cpu.Info()
	if err != nil {
		utils.Log.Error("failed to cpu: %v", err)
	}
	for _, ci := range info {
		utils.Log.Info("CPU 型號: %s, 速度: %.2fMHz\n", ci.ModelName, ci.Mhz)
	}

	// --- CPU 總體使用率 ---
	percent, err := cpu.Percent(time.Second, false)
	if err != nil {
		utils.Log.Error("failed to cpu: %v", err)
	}
	utils.Log.Info("CPU 使用率(整體): %.2f%%\n", percent[0])

	// --- 每核心使用率 ---
	perCore, err := cpu.Percent(time.Second, true)
	if err != nil {
		utils.Log.Error("failed to cpu: %v", err)
	}
	for i, p := range perCore {
		utils.Log.Info("CPU 核心 %d 使用率: %.2f%%\n", i, p)
	}

	// --- Load Average（Windows 無此功能） ---
	if runtime.GOOS == "windows" {
		utils.Log.Info("Load Average: Windows 不支援（略過）")
	} else {
		l, err := load.Avg()
		if err == nil {
			utils.Log.Info("Load Avg: 1m=%.2f  5m=%.2f  15m=%.2f\n",
				l.Load1, l.Load5, l.Load15)
		}
	}

	// --- CPU Times（user/system/idle）---
	times, err := cpu.Times(false)
	if err == nil && len(times) > 0 {
		utils.Log.Info("CPU Times:")
		utils.Log.Info("  User:   %.2fs\n", times[0].User)
		utils.Log.Info("  System: %.2fs\n", times[0].System)
		utils.Log.Info("  Idle:   %.2fs\n", times[0].Idle)
		utils.Log.Info("  Nice:   %.2fs\n", times[0].Nice)
		utils.Log.Info("  IOWait: %.2fs\n", times[0].Iowait)
		utils.Log.Info("  IRQ:    %.2fs\n", times[0].Irq)
	}

}
