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
	// 取得 partitions
	partitions, err := disk.Partitions(false)
	if err != nil {
		utils.Log.Error("[Disk] 無法讀取分割區: %v", err)
	}

	// 取得 I/O counters
	ioCounters, err := disk.IOCounters()
	if err != nil {
		utils.Log.Error("[Disk] 無法讀取 I/O 統計: %v", err)
		return prevIO
	}

	// 合併資料：map["C:"] -> {Partition + IOCounter}
	type DiskInfo struct {
		Part  *disk.PartitionStat
		Usage *disk.UsageStat
		IO    *disk.IOCountersStat
	}

	disks := make(map[string]*DiskInfo)

	// 塞 partition 與 usage
	for _, p := range partitions {
		usage, _ := disk.Usage(p.Mountpoint)
		disks[p.Device] = &DiskInfo{
			Part:  &p,
			Usage: usage,
		}
	}

	// 塞 I/O
	for name, io := range ioCounters {
		if disks[name] == nil {
			disks[name] = &DiskInfo{}
		}
		disks[name].IO = &io
	}

	// 印出合併後的完整資訊
	for name, d := range disks {
		if d.Part == nil && d.IO == nil {
			continue
		}

		// 計算速率
		var readRate, writeRate uint64
		var busyRatio float64

		if prevIO != nil {
			if prev, ok := prevIO[name]; ok && d.IO != nil {
				readRate = d.IO.ReadBytes - prev.ReadBytes
				writeRate = d.IO.WriteBytes - prev.WriteBytes
				if intervalMs > 0 {
					busyRatio = float64(d.IO.IoTime-prev.IoTime) / intervalMs * 100
				}
			}
		}

		utils.Log.Debug(
			"[Disk] Name=%s | Mount=%v, FsType=%v, RO=%v | Total=%vGB, Used=%vGB, Free=%vGB, Usage=%.2f%% | "+
				"Read=%vB, Write=%vB | ReadCnt=%v, WriteCnt=%v | ReadTime=%vms, WriteTime=%vms, IoTime=%vms | "+
				"ReadRate=%vB/s, WriteRate=%vB/s | Busy=%.2f%%",

			name,

			// Partition/usage
			getOrNil(func() interface{} { return d.Part.Mountpoint }),
			getOrNil(func() interface{} { return d.Part.Fstype }),
			getOrNil(func() interface{} { return d.Part.Opts == "ro" }),

			getOrNil(func() interface{} { return d.Usage.Total / 1024 / 1024 / 1024 }),
			getOrNil(func() interface{} { return d.Usage.Used / 1024 / 1024 / 1024 }),
			getOrNil(func() interface{} { return d.Usage.Free / 1024 / 1024 / 1024 }),
			getOrNil(func() interface{} { return d.Usage.UsedPercent }),

			// IO
			getOrNil(func() interface{} { return d.IO.ReadBytes }),
			getOrNil(func() interface{} { return d.IO.WriteBytes }),
			getOrNil(func() interface{} { return d.IO.ReadCount }),
			getOrNil(func() interface{} { return d.IO.WriteCount }),
			getOrNil(func() interface{} { return d.IO.ReadTime }),
			getOrNil(func() interface{} { return d.IO.WriteTime }),
			getOrNil(func() interface{} { return d.IO.IoTime }),

			readRate,
			writeRate,
			busyRatio,
		)
	}

	return ioCounters
}

// 如果資料不存在避免 panic
func getOrNil(f func() interface{}) interface{} {
	defer func() {
		recover()
	}()
	return f()
}
