package cpu

import (
	"context"
	"encoding/json"
	"runtime"
	"sysprobe/internal/config"
	"sysprobe/internal/utils"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/load"
)

type CPUInfo struct {
	Category    string      `json:"Category"`
	CoreCount   int         `json:"CoreCount"`
	CpuModel    string      `json:"CpuModel"`
	CpuMHz      float64     `json:"CpuMHz"`
	CpuUsage    float64     `json:"CpuUsage"`
	CoreUsage   []float64   `json:"CoreUsage"`
	LoadAverage interface{} `json:"LoadAverage"`
	CpuTime     struct {
		User   float64 `json:"User"`
		System float64 `json:"System"`
		Idle   float64 `json:"Idle"`
		Nice   float64 `json:"Nice"`
		IOWait float64 `json:"IOWait"`
		IRQ    float64 `json:"IRQ"`
	} `json:"CpuTime"`
}

func Start(ctx context.Context, cfg config.MonitorModule) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				utils.Log.Error("[CPU] goroutine panic: %v", r)
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
	counts, _ := cpu.Counts(true)

	info, _ := cpu.Info()
	var model string
	var mhz float64
	if len(info) > 0 {
		model = info[0].ModelName
		mhz = info[0].Mhz
	}

	percent, _ := cpu.Percent(time.Second, false)
	perCore, _ := cpu.Percent(time.Second, true)

	var loadAvg interface{} = nil
	if runtime.GOOS != "windows" {
		if l, err := load.Avg(); err == nil {
			loadAvg = []float64{l.Load1, l.Load5, l.Load15}
		}
	}

	times, _ := cpu.Times(false)
	var cpuTimes struct {
		User   float64 `json:"User"`
		System float64 `json:"System"`
		Idle   float64 `json:"Idle"`
		Nice   float64 `json:"Nice"`
		IOWait float64 `json:"IOWait"`
		IRQ    float64 `json:"IRQ"`
	}
	if len(times) > 0 {
		cpuTimes.User = times[0].User
		cpuTimes.System = times[0].System
		cpuTimes.Idle = times[0].Idle
		cpuTimes.Nice = times[0].Nice
		cpuTimes.IOWait = times[0].Iowait
		cpuTimes.IRQ = times[0].Irq
	}

	data := CPUInfo{
		Category:    "CPU",
		CoreCount:   counts,
		CpuModel:    model,
		CpuMHz:      mhz,
		CpuUsage:    percent[0],
		CoreUsage:   perCore,
		LoadAverage: loadAvg,
		CpuTime:     cpuTimes,
	}

	// 一行 JSON 輸出
	b, _ := json.Marshal(data)
	utils.Log.Debug("%s", string(b))
}
