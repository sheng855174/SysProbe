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
		Name  string
		Part  *disk.PartitionStat
		Usage *disk.UsageStat
		IO    *disk.IOCountersStat
	}

	disks := make(map[string]*DiskInfo)

	// 1️⃣ 抓 partitions
	for _, p := range partitions {
		if p.Fstype == "tmpfs" || p.Fstype == "overlay" || strings.HasPrefix(p.Device, "/dev/loop") {
			continue
		}
		usage, err := disk.Usage(p.Mountpoint)
		if err != nil {
			continue
		}
		disks[p.Device] = &DiskInfo{
			Name:  p.Device,
			Part:  &p,
			Usage: usage,
		}
	}

	// 2️⃣ 把原始磁碟也加進來，避免 Linux 分割區對不上 IOCounters
	for key, io := range ioCounters {
		if _, ok := disks[key]; !ok {
			disks[key] = &DiskInfo{
				Name: key,
				IO:   &io,
			}
		} else {
			disks[key].IO = &io
		}
	}

	// 3️⃣ 輸出
	for _, d := range disks {
		if d.Usage == nil {
			continue
		}

		readRate, writeRate := uint64(0), uint64(0)
		busy := float64(0)

		if prev != nil && d.IO != nil {
			if prevIO, ok := prev[d.Name]; ok {
				readRate = d.IO.ReadBytes - prevIO.ReadBytes
				writeRate = d.IO.WriteBytes - prevIO.WriteBytes
				if intervalMs > 0 {
					busy = float64(d.IO.IoTime-prevIO.IoTime) / intervalMs * 100
				}
			}
		}

		mount, fs, total, used, free, usage := "-", "-", uint64(0), uint64(0), uint64(0), 0.0
		if d.Part != nil && d.Usage != nil {
			mount = d.Part.Mountpoint
			fs = d.Part.Fstype
			total = d.Usage.Total / 1024 / 1024 / 1024
			used = d.Usage.Used / 1024 / 1024 / 1024
			free = d.Usage.Free / 1024 / 1024 / 1024
			usage = d.Usage.UsedPercent
		}

		utils.Log.Debug(
			"[Disk] Name=%s Mount=%s Fs=%s Total=%dGB Used=%dGB Free=%dGB Usage=%.2f%% | ReadRate=%dB/s WriteRate=%dB/s Busy=%.2f%%",
			d.Name,
			mount,
			fs,
			total,
			used,
			free,
			usage,
			readRate,
			writeRate,
			busy,
		)
	}

	return ioCounters
}
