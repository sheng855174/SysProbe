package memory

import (
	"context"
	"encoding/json"
	"sysprobe/internal/config"
	"sysprobe/internal/service"
	"sysprobe/internal/utils"
	"time"

	"github.com/shirou/gopsutil/v4/mem"
)

const Category = "MEMORY"

type MemoryInfo struct {
	Host      service.HostInfo `json:"Host"`
	Category  string           `json:"Category"`
	Total     uint64           `json:"Total"`     // bytes
	Used      uint64           `json:"Used"`      // bytes
	Free      uint64           `json:"Free"`      // bytes
	UsedPct   float64          `json:"UsedPct"`   // %
	Timestamp string           `json:"Timestamp"` // RFC3339
}

func Start(ctx context.Context, cfg config.MonitorConfig, host *service.HostUpdater) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				utils.Log.Error("[Memory] goroutine panic: %v", r)
				Start(ctx, cfg, host)
			}
		}()

		logger := utils.GetLogger(cfg.Data+"/"+Category, Category, cfg.Days)
		ticker := time.NewTicker(time.Duration(cfg.Memory.Interval) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				memData := monitorMemory(host)
				if len(memData) > 0 {
					logger.Write(memData)
				}
			case <-ctx.Done():
				utils.Log.Info("[Memory] 收集器已停止")
				return
			}
		}
	}()
}

func monitorMemory(host *service.HostUpdater) []byte {
	vm, err := mem.VirtualMemory()
	if err != nil {
		utils.Log.Error("[Memory] 無法取得記憶體資訊: %v", err)
		return nil
	}

	data := MemoryInfo{
		Host:      host.Get(),
		Category:  Category,
		Total:     vm.Total,
		Used:      vm.Used,
		Free:      vm.Available,
		UsedPct:   vm.UsedPercent,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	b, _ := json.Marshal(data)
	utils.Log.Debug("%s", string(b))
	return b
}
