package monitor

import (
	"context"
	"sysprobe/internal/monitor/cpu"
	"sysprobe/internal/monitor/disk"
	"sysprobe/internal/monitor/network"
	"sysprobe/internal/utils"
)

type Config struct {
	CPU  bool
	Disk bool
	Net  bool
}

func LoadMonitor(ctx context.Context, cfg Config) {
	utils.Log.Info("Monitor Manager starting...")
	if cfg.CPU {
		utils.Log.Info("→ Starting CPU monitor")
		cpu.Start(ctx)
	}
	if cfg.Disk {
		utils.Log.Info("→ Starting Disk monitor")
		disk.Start(ctx)
	}
	if cfg.Net {
		utils.Log.Info("→ Starting Network monitor")
		network.Start(ctx)
	}
	utils.Log.Info("Monitor started")
}
