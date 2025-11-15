package monitor

import (
	"context"
	"sysprobe/internal/config"
	"sysprobe/internal/monitor/cpu"
	"sysprobe/internal/monitor/disk"
	"sysprobe/internal/monitor/network"
	"sysprobe/internal/utils"
)

func LoadMonitor(ctx context.Context, cfg config.MonitorConfig) {
	utils.Log.Info("Monitor Manager starting...")
	if cfg.CPU.Enable {
		utils.Log.Info("→ Starting CPU monitor")
		cpu.Start(ctx, cfg.CPU, cfg.Data)
	}
	if cfg.Disk.Enable {
		utils.Log.Info("→ Starting Disk monitor")
		disk.Start(ctx, cfg.Disk, cfg.Data)
	}
	if cfg.Net.Enable {
		utils.Log.Info("→ Starting Network monitor")
		network.Start(ctx, cfg.Net, cfg.Data)
	}
	utils.Log.Info("Monitor started")
}
