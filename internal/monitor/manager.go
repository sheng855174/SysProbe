package monitor

import (
	"context"
	"sysprobe/internal/config"
	"sysprobe/internal/monitor/cpu"
	"sysprobe/internal/monitor/disk"
	"sysprobe/internal/monitor/memory"
	"sysprobe/internal/monitor/network"
	"sysprobe/internal/utils"
)

func LoadMonitor(ctx context.Context, cfg config.MonitorConfig) {
	utils.Log.Info("Monitor Manager starting...")
	if cfg.CPU.Enable {
		utils.Log.Info("→ Starting CPU monitor")
		cpu.Start(ctx, cfg)
	}
	if cfg.Memory.Enable {
		utils.Log.Info("→ Starting CPU monitor")
		memory.Start(ctx, cfg)
	}
	if cfg.Disk.Enable {
		utils.Log.Info("→ Starting Disk monitor")
		disk.Start(ctx, cfg)
	}
	if cfg.Net.Enable {
		utils.Log.Info("→ Starting Network monitor")
		network.Start(ctx, cfg)
	}
	utils.Log.Info("Monitor started")
}
