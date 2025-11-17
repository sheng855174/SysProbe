package monitor

import (
	"context"
	"sysprobe/internal/config"
	"sysprobe/internal/monitor/cpu"
	"sysprobe/internal/monitor/disk"
	"sysprobe/internal/monitor/memory"
	"sysprobe/internal/monitor/network"
	"sysprobe/internal/service"
	"sysprobe/internal/utils"
)

func LoadMonitor(ctx context.Context, cfg config.MonitorConfig, host *service.HostUpdater) {
	utils.Log.Info("Monitor Manager starting...")
	if cfg.CPU.Enable {
		utils.Log.Info("→ Starting CPU monitor")
		cpu.Start(ctx, cfg, host)
	}
	if cfg.Memory.Enable {
		utils.Log.Info("→ Starting CPU monitor")
		memory.Start(ctx, cfg, host)
	}
	if cfg.Disk.Enable {
		utils.Log.Info("→ Starting Disk monitor")
		disk.Start(ctx, cfg, host)
	}
	if cfg.Net.Enable {
		utils.Log.Info("→ Starting Network monitor")
		network.Start(ctx, cfg, host)
	}
	utils.Log.Info("Monitor started")
}
