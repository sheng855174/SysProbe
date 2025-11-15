package disk

import (
	"context"
	"strings"
	"sysprobe/internal/config"
	"sysprobe/internal/utils"
	"time"

	"github.com/shirou/gopsutil/v4/disk"
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

		var prevIO map[string]disk.IOCountersStat
		intervalMs := float64(cfg.Interval * 1000)

		for {
			select {
			case <-ticker.C:
				prevIO = monitorDisk(prevIO, intervalMs)
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

	disks := make(map[string]*DiskInfo) // key = Device or Mount

	// 1️⃣ 塞 partition + usage
	for _, p := range partitions {
		if p.Fstype == "tmpfs" || p.Fstype == "overlay" || strings.HasPrefix(p.Device, "/dev/loop") {
			continue
		}

		usage, err := disk.Usage(p.Mountpoint)
		if err != nil {
			continue
		}

		disks[p.Device] = &DiskInfo{
			Part:  &p,
			Usage: usage,
		}
	}

	// 2️⃣ 塞 I/O
	for key, io := range ioCounters {
		// 找對應 partition
		for dev, d := range disks {
			match := false
			if strings.EqualFold(dev, key) {
				match = true
			} else if strings.HasPrefix(dev, "/dev/") && strings.HasPrefix(key, dev[5:]) { // /dev/sda1 -> sda1
				match = true
			} else if strings.HasPrefix(dev, key) { // sda -> sda1
				match = true
			}
			if match {
				dd := io
				d.IO = &dd
				break
			}
		}
	}

	// 3️⃣ 輸出
	for _, d := range disks {
		if d.Part == nil || d.Usage == nil || d.IO == nil {
			continue
		}

		readRate, writeRate := uint64(0), uint64(0)
		busy := float64(0)

		if prev != nil {
			if prevIO, ok := prev[d.Part.Device]; ok {
				readRate = d.IO.ReadBytes - prevIO.ReadBytes
				writeRate = d.IO.WriteBytes - prevIO.WriteBytes
				if intervalMs > 0 {
					busy = float64(d.IO.IoTime-prevIO.IoTime) / intervalMs * 100
				}
			}
		}

		utils.Log.Debug(
			"[Disk] Mount=%s Dev=%s Fs=%s Total=%dGB Used=%dGB Free=%dGB Usage=%.2f%% | ReadRate=%dB/s WriteRate=%dB/s Busy=%.2f%%",
			d.Part.Mountpoint,
			d.Part.Device,
			d.Part.Fstype,
			d.Usage.Total/1024/1024/1024,
			d.Usage.Used/1024/1024/1024,
			d.Usage.Free/1024/1024/1024,
			d.Usage.UsedPercent,
			readRate,
			writeRate,
			busy,
		)
	}

	return ioCounters
}
