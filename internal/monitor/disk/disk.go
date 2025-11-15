package disk

import (
	"context"
	"sysprobe/internal/config"
	"sysprobe/internal/utils"
	"time"

	"github.com/shirou/gopsutil/disk"
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

		var prevIOCounters map[string]disk.IOCountersStat
		intervalMs := float64(cfg.Interval)

		for {
			select {
			case <-ticker.C:
				prevIOCounters = monitorDisk(prevIOCounters, intervalMs)
			case <-ctx.Done():
				utils.Log.Debug("[Disk] 收集器已停止")
				return
			}
		}
	}()
}

func monitorDisk(prevIO map[string]disk.IOCountersStat, intervalMs float64) map[string]disk.IOCountersStat {
	// 1️⃣ 分割區資訊
	partitions, err := disk.Partitions(false)
	if err != nil {
		utils.Log.Error("[Disk] 無法讀取分割區: %v", err)
	} else {
		for _, p := range partitions {
			usage, err := disk.Usage(p.Mountpoint)
			if err != nil {
				utils.Log.Error("[Disk] 無法讀取使用狀態: %v", err)
				continue
			}
			utils.Log.Debug(
				"[Disk] 分割區 %s (掛載 %s, FsType %s, ReadOnly=%v): Total=%vGB, Used=%vGB, Free=%vGB, Usage=%.2f%%",
				p.Device,
				p.Mountpoint,
				p.Fstype,
				p.Opts == "ro",
				usage.Total/1024/1024/1024,
				usage.Used/1024/1024/1024,
				usage.Free/1024/1024/1024,
				usage.UsedPercent,
			)
		}
	}

	// 2️⃣ I/O 統計
	ioCounters, err := disk.IOCounters()
	if err != nil {
		utils.Log.Error("[Disk] 無法讀取 I/O 統計: %v", err)
		return prevIO
	}

	for name, counter := range ioCounters {
		var readRate, writeRate uint64
		var busyRatio float64

		if prevIO != nil {
			if prev, ok := prevIO[name]; ok {
				readRate = counter.ReadBytes - prev.ReadBytes
				writeRate = counter.WriteBytes - prev.WriteBytes
				if intervalMs > 0 {
					busyRatio = float64(counter.IoTime-prev.IoTime) / float64(intervalMs) * 100
				}
			}
		}

		utils.Log.Debug(
			"[Disk] %s: Read=%vB, Write=%vB, ReadCount=%v, WriteCount=%v, ReadTime=%vms, WriteTime=%vms, IoTime=%vms, ReadRate=%vB/s, WriteRate=%vB/s, Busy=%.2f%%",
			name,
			counter.ReadBytes,
			counter.WriteBytes,
			counter.ReadCount,
			counter.WriteCount,
			counter.ReadTime,
			counter.WriteTime,
			counter.IoTime,
			readRate,
			writeRate,
			busyRatio,
		)
	}

	return ioCounters
}
