package network

import (
	"context"
	"strings"
	"sysprobe/internal/config"
	"sysprobe/internal/utils"
	"time"

	"github.com/shirou/gopsutil/v4/net"
)

func Start(ctx context.Context, cfg config.MonitorModule) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				utils.Log.Error("[Network] goroutine panic: %v", r)
				Start(ctx, cfg) // restart
			}
		}()

		ticker := time.NewTicker(time.Duration(cfg.Interval) * time.Second)
		defer ticker.Stop()

		var prevStats map[string]net.IOCountersStat
		intervalSec := float64(cfg.Interval)

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

// ----------------------------
// ❗過濾多餘網卡（跨平台）
// ----------------------------
func isSkipInterface(name string) bool {
	n := strings.ToLower(name)

	skipPrefixes := []string{
		// Windows
		"loopback", "isatap", "teredo", "virtualbox", "vmware",
		"npcap", "bluetooth", "hyper-v", "vethernet", "local area connection",

		// Linux
		"lo", "docker", "cni", "veth", "br-", "kube", "flannel",
	}

	for _, p := range skipPrefixes {
		if strings.HasPrefix(n, strings.ToLower(p)) {
			return true
		}
	}

	return false
}

// ----------------------------
// 📡 monitorNet 主流程
// ----------------------------
func monitorNet(prev map[string]net.IOCountersStat, intervalSec float64) map[string]net.IOCountersStat {
	utils.Log.Debug("[Network] 收集網路資訊中...")

	// 1️⃣ 取得所有 NIC 流量
	stats, err := net.IOCounters(true)
	if err != nil {
		utils.Log.Error("[Network] 無法取得網路統計: %v", err)
		return prev
	}

	// 2️⃣ 統計 TCP 連線狀態
	tcpState := make(map[string]int)
	conns, err := net.Connections("tcp")
	if err != nil {
		utils.Log.Error("[Network] 無法取得連線: %v", err)
	} else {
		for _, c := range conns {
			tcpState[c.Status]++
		}
	}

	// 3️⃣ 處理每個 NIC（過濾過後）
	for _, s := range stats {

		// 🚫 過濾不必要 NIC
		if isSkipInterface(s.Name) {
			continue
		}

		// 📊 計算每秒速率
		var txRate, rxRate, txPPS, rxPPS float64
		if prev != nil {
			if p, ok := prev[s.Name]; ok {
				txRate = float64(s.BytesSent-p.BytesSent) / intervalSec
				rxRate = float64(s.BytesRecv-p.BytesRecv) / intervalSec
				txPPS = float64(s.PacketsSent-p.PacketsSent) / intervalSec
				rxPPS = float64(s.PacketsRecv-p.PacketsRecv) / intervalSec
			}
		}

		// ✔ 輸出 NIC 資料 + Summary TCP 狀態
		utils.Log.Debug(
			"[Network] IF=%s | Tx=%.0fB/s Rx=%.0fB/s | TxPPS=%.1f RxPPS=%.1f | TCP=%v",
			s.Name,
			txRate, rxRate,
			txPPS, rxPPS,
			tcpState,
		)
	}

	// 4️⃣ 下次計算需要 diff → 存起來
	newPrev := make(map[string]net.IOCountersStat)
	for _, s := range stats {
		newPrev[s.Name] = s
	}

	return newPrev
}
