package network

import (
	"context"
	"sysprobe/internal/config"
	"sysprobe/internal/utils"
	"time"

	"github.com/shirou/gopsutil/net"
)

func Start(ctx context.Context, cfg config.MonitorModule) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				utils.Log.Error("[Network] goroutine panic: %v", r)
				// 可以選擇重新啟動 goroutine
				Start(ctx, cfg)
			}
		}()

		ticker := time.NewTicker(time.Duration(cfg.Interval) * time.Second)
		defer ticker.Stop()

		var prevStats map[string]net.IOCountersStat
		intervalSec := float64(cfg.Interval) / 1000.0

		for {
			select {
			case <-ticker.C:
				prevStats = monitorNet(prevStats, intervalSec)
			case <-ctx.Done():
				utils.Log.Debug("[Network] 收集器已停止")
				return
			}
		}
	}()
}

func monitorNet(prev map[string]net.IOCountersStat, intervalSec float64) map[string]net.IOCountersStat {
	utils.Log.Debug("[Network] 收集網路資訊中...")

	// 1️⃣ 收集介面流量統計
	stats, err := net.IOCounters(true)
	if err != nil {
		utils.Log.Error("[Network] 無法取得網路統計: %v", err)
		return prev
	}

	// 2️⃣ 統計一次 TCP 連線狀態（全系統）
	tcpState := make(map[string]int)
	conns, err := net.Connections("tcp")
	if err != nil {
		utils.Log.Error("[Network] 無法取得連線: %v", err)
	} else {
		for _, c := range conns {
			tcpState[c.Status]++
		}
	}

	// 🔄3️⃣ 每張網卡一起輸出（整合 TCP 狀態）
	for _, s := range stats {
		var txRate, rxRate, txPPS, rxPPS float64

		if prev != nil {
			if p, ok := prev[s.Name]; ok {
				txRate = float64(s.BytesSent-p.BytesSent) / intervalSec
				rxRate = float64(s.BytesRecv-p.BytesRecv) / intervalSec
				txPPS = float64(s.PacketsSent-p.PacketsSent) / intervalSec
				rxPPS = float64(s.PacketsRecv-p.PacketsRecv) / intervalSec
			}
		}

		// 🔹輸出格式整合：網卡資訊 + TCP 狀態摘要
		utils.Log.Debug(
			"[Network] IF=%s | Tx=%.2fB/s, Rx=%.2fB/s | TxPPS=%.2f, RxPPS=%.2f | "+
				"Err(in/out)=%v/%v | Drop(in/out)=%v/%v | TCP=%v",
			s.Name,
			txRate,
			rxRate,
			txPPS,
			rxPPS,
			s.Errin, s.Errout,
			s.Dropin, s.Dropout,
			tcpState,
		)
	}

	// 4️⃣ 回傳本次 stats 作為下次的 prev
	newPrev := make(map[string]net.IOCountersStat)
	for _, s := range stats {
		newPrev[s.Name] = s
	}
	return newPrev
}
