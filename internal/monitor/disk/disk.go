package disk

import (
	"context"
	"encoding/json"
	"strings"
	"sysprobe/internal/config"
	"sysprobe/internal/utils"
	"time"

	"github.com/shirou/gopsutil/v4/disk"
)

// DiskPartition 對應 JSON 中的每個分割區
type DiskPartition struct {
	Name      string  `json:"Name"`
	Mount     string  `json:"Mount"`
	Fs        string  `json:"Fs"`
	Total     uint64  `json:"Total"`     // GB
	Used      uint64  `json:"Used"`      // GB
	Free      uint64  `json:"Free"`      // GB
	Usage     float64 `json:"Usage"`     // %
	ReadRate  uint64  `json:"ReadRate"`  // B/s
	WriteRate uint64  `json:"WriteRate"` // B/s
	Busy      float64 `json:"Busy"`      // %
}

// DiskInfoJSON 對應整個 JSON 結構
type DiskInfoJSON struct {
	Category   string          `json:"Category"`
	Partitions []DiskPartition `json:"Partitions"`
}

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

	// 1️⃣ 收集 partitions
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

	// 2️⃣ 加入 IO Counters，避免 Linux 分割區對不上
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

	var partitionsJSON []DiskPartition

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

		mount, fs := "-", "-"
		total, used, free := uint64(0), uint64(0), uint64(0)
		usage := 0.0

		if d.Part != nil && d.Usage != nil {
			mount = d.Part.Mountpoint
			fs = d.Part.Fstype
			total = d.Usage.Total / 1024 / 1024 / 1024
			used = d.Usage.Used / 1024 / 1024 / 1024
			free = d.Usage.Free / 1024 / 1024 / 1024
			usage = d.Usage.UsedPercent
		}

		partitionsJSON = append(partitionsJSON, DiskPartition{
			Name:      d.Name,
			Mount:     mount,
			Fs:        fs,
			Total:     total,
			Used:      used,
			Free:      free,
			Usage:     usage,
			ReadRate:  readRate,
			WriteRate: writeRate,
			Busy:      busy,
		})
	}

	data := DiskInfoJSON{
		Category:   "DISK",
		Partitions: partitionsJSON,
	}

	// 一行 JSON 輸出
	b, _ := json.Marshal(data)
	utils.Log.Debug("%s", string(b))

	return ioCounters
}
