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
				monitorCPU()
			case <-ctx.Done():
				utils.Log.Info("[CPU] 收集器已停止")
				return
			}
		}
	}()
}

func monitorCPU() {
	utils.Log.Debug("[CPU] 收集 CPU 使用率中...")

	// --- CPU 核心數 ---
	counts, err := cpu.Counts(true)
	if err != nil {
		utils.Log.Error("failed to cpu: %v", err)
	}
	utils.Log.Debug("CPU 核心數(含超執行緒): %d", counts)

	// --- CPU 型號資訊 ---
	info, err := cpu.Info()
	if err != nil {
		utils.Log.Error("failed to cpu: %v", err)
	}
	for _, ci := range info {
		utils.Log.Debug("CPU 型號: %s, 速度: %.2fMHz", ci.ModelName, ci.Mhz)
	}

	// --- CPU 總體使用率（單行輸出）---
	percent, err := cpu.Percent(time.Second, false)
	if err != nil {
		utils.Log.Error("failed to cpu: %v", err)
	}
	// 不加 \n → 就不會自己換行
	utils.Log.Debug("CPU 使用率(整體): %.2f%%", percent[0])

	// --- 每核心使用率（每行印一個 core）---
	perCore, err := cpu.Percent(time.Second, true)
	if err != nil {
		utils.Log.Error("failed to cpu: %v", err)
	}
	for i, p := range perCore {
		utils.Log.Debug("  Core[%d]: %.2f%%", i, p)
	}

	// --- Load Average ---
	if runtime.GOOS == "windows" {
		utils.Log.Debug("Load Average: Windows 不支援（略過）")
	} else {
		l, err := load.Avg()
		if err == nil {
			utils.Log.Debug("Load Avg: 1m=%.2f 5m=%.2f 15m=%.2f",
				l.Load1, l.Load5, l.Load15)
		}
	}

	// --- CPU Times ---
	times, err := cpu.Times(false)
	if err == nil && len(times) > 0 {
		utils.Log.Debug("CPU Times: User=%.2fs System=%.2fs Idle=%.2fs Nice=%.2fs IOWait=%.2fs IRQ=%.2fs",
			times[0].User,
			times[0].System,
			times[0].Idle,
			times[0].Nice,
			times[0].Iowait,
			times[0].Irq,
		)
	}
}
