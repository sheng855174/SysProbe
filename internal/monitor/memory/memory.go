package memory

import (
	"context"
	"encoding/json"
	"sysprobe/internal/config"
	"sysprobe/internal/utils"
	"time"

	"github.com/shirou/gopsutil/v4/mem"
)

type MemoryInfo struct {
	Category  string  `json:"Category"`
	Total     uint64  `json:"Total"`     // bytes
	Used      uint64  `json:"Used"`      // bytes
	Free      uint64  `json:"Free"`      // bytes
	UsedPct   float64 `json:"UsedPct"`   // %
	Timestamp string  `json:"Timestamp"` // RFC3339
}

func Start(ctx context.Context, cfg config.MonitorModule, path string) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				utils.Log.Error("[Memory] goroutine panic: %v", r)
				Start(ctx, cfg, path)
			}
		}()

		ticker := time.NewTicker(time.Duration(cfg.Interval) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				memData := monitorMemory()
				if len(memData) > 0 {
					utils.WriteJSONLine(path, "memory.jsonl", memData)
				}
			case <-ctx.Done():
				utils.Log.Info("[Memory] 收集器已停止")
				return
			}
		}
	}()
}

func monitorMemory() []byte {
	vm, err := mem.VirtualMemory()
	if err != nil {
		utils.Log.Error("[Memory] 無法取得記憶體資訊: %v", err)
		return nil
	}

	data := MemoryInfo{
		Category:  "MEMORY",
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
