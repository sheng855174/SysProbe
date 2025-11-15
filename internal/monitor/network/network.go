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
	utils.Log.Debug("[Network] 收集網路流量中...")

	// 1️⃣ 收集介面流量統計
	stats, err := net.IOCounters(true)
	if err != nil {
		utils.Log.Error("[Network] 無法取得網路統計: %v", err)
		return prev
	}

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

		utils.Log.Debug(
			"[Network] %s: BytesSent=%v, BytesRecv=%v, TxRate=%.2fB/s, RxRate=%.2fB/s, "+
				"PacketsSent=%v, PacketsRecv=%v, TxPPS=%.2f, RxPPS=%.2f, "+
				"ErrIn=%v, ErrOut=%v, DropIn=%v, DropOut=%v",
			s.Name,
			s.BytesSent,
			s.BytesRecv,
			txRate,
			rxRate,
			s.PacketsSent,
			s.PacketsRecv,
			txPPS,
			rxPPS,
			s.Errin,
			s.Errout,
			s.Dropin,
			s.Dropout,
		)
	}

	// 2️⃣ 統計 TCP Port 狀態總數
	conns, err := net.Connections("tcp")
	if err != nil {
		utils.Log.Error("[Network] 無法取得連線: %v", err)
	} else {
		statusCount := make(map[string]int)
		for _, c := range conns {
			statusCount[c.Status]++
		}
		for status, count := range statusCount {
			utils.Log.Debug("[Network-Port] Status=%s, Count=%d", status, count)
		}
	}

	// 3️⃣ 回傳本次 stats 作為下次計算差值
	newPrev := make(map[string]net.IOCountersStat)
	for _, s := range stats {
		newPrev[s.Name] = s
	}
	return newPrev
}
