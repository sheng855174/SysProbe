package disk

import (
	"context"
	"strings"
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
				Start(ctx, cfg)
			}
		}()

		ticker := time.NewTicker(time.Duration(cfg.Interval) * time.Second)
		defer ticker.Stop()

		var prev map[string]disk.IOCountersStat
		intervalMs := float64(cfg.Interval)

		for {
			select {
			case <-ticker.C:
				prev = monitorDisk(prev, intervalMs)
			case <-ctx.Done():
				utils.Log.Debug("[Disk] 收集器已停止")
				return
			}
		}
	}()
}

func monitorDisk(prev map[string]disk.IOCountersStat, intervalMs float64) map[string]disk.IOCountersStat {
	partitions, _ := disk.Partitions(false)
	ioCounters, _ := disk.IOCounters()

	type DiskInfo struct {
		Part  *disk.PartitionStat
		Usage *disk.UsageStat
		IO    *disk.IOCountersStat
	}

	disks := make(map[string]*DiskInfo) // key = Mountpoint

	// 1️⃣ 塞分割區
	for _, p := range partitions {
		// 過濾 Linux 的 loop、tmpfs、overlay
		if p.Fstype == "tmpfs" || p.Fstype == "overlay" || strings.HasPrefix(p.Device, "/dev/loop") {
			continue
		}

		usage, _ := disk.Usage(p.Mountpoint)

		disks[p.Mountpoint] = &DiskInfo{
			Part:  &p,
			Usage: usage,
		}
	}

	// 2️⃣ 塞 IO counters → 嘗試比對 Mountpoint
	for _, d := range disks {
		// Windows: Device = "C:" → IOCounters key = "C:"
		if io, ok := ioCounters[d.Part.Device]; ok {
			dd := io
			d.IO = &dd
			continue
		}

		// Linux: Device = "/dev/sda1" → IOCounters key = "sda" or "sda1"
		dev := d.Part.Device
		if strings.HasPrefix(dev, "/dev/") {
			dev = strings.TrimPrefix(dev, "/dev/")
		}

		if io, ok := ioCounters[dev]; ok {
			dd := io
			d.IO = &dd
			continue
		}

		// 有些 Linux key 是整 disk：sda、nvme0n1
		for key := range ioCounters {
			if strings.HasPrefix(dev, key) {
				dd := ioCounters[key]
				d.IO = &dd
				break
			}
		}
	}

	// 3️⃣ 輸出
	for mount, d := range disks {
		var readRate, writeRate uint64
		var busyRatio float64

		if prev != nil && d.IO != nil {
			if p, ok := prev[d.Part.Device]; ok {
				readRate = d.IO.ReadBytes - p.ReadBytes
				writeRate = d.IO.WriteBytes - p.WriteBytes
				if intervalMs > 0 {
					busyRatio = float64(d.IO.IoTime-p.IoTime) / intervalMs * 100
				}
			}
		}

		utils.Log.Debug(
			"[Disk] Mount=%s Dev=%s Fs=%s Total=%vGB Used=%vGB Free=%vGB Used=%.2f%% | "+
				"ReadRate=%vB/s WriteRate=%vB/s Busy=%.2f%%",
			mount,
			d.Part.Device,
			d.Part.Fstype,
			d.Usage.Total/1024/1024/1024,
			d.Usage.Used/1024/1024/1024,
			d.Usage.Free/1024/1024/1024,
			d.Usage.UsedPercent,
			readRate,
			writeRate,
			busyRatio,
		)
	}

	return ioCounters
}
