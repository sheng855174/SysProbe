package network

import (
	"context"
	"encoding/json"
	"strings"
	"sysprobe/internal/config"
	"sysprobe/internal/utils"
	"time"

	stdnet "net" // 標準 library net，用於取得 MAC/IP

	gopsnet "github.com/shirou/gopsutil/v4/net" // gopsutil 的 net，用於流量/連線統計
)

// NetworkInterface 對應每個 NIC 的 JSON
type NetworkInterface struct {
	Name      string         `json:"Name"`
	IP        string         `json:"IP"`  // 優先 IPv4，若無則 IPv6
	MAC       string         `json:"MAC"` // MAC address
	Tx        uint64         `json:"Tx"`  // B/s
	Rx        uint64         `json:"Rx"`  // B/s
	TxPPS     float64        `json:"TxPPS"`
	RxPPS     float64        `json:"RxPPS"`
	TCP       map[string]int `json:"TCP"` // TCP 狀態統計
	Timestamp string         `json:"Timestamp"`
}

// NetworkJSON 對應整個 JSON
type NetworkJSON struct {
	Category   string             `json:"Category"`
	Interfaces []NetworkInterface `json:"Interfaces"`
}

func Start(ctx context.Context, cfg config.MonitorConfig) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				utils.Log.Error("[Network] goroutine panic: %v", r)
				Start(ctx, cfg) // restart
			}
		}()

		logger := utils.GetLogger(cfg.Data+"/net", "net", cfg.Days)
		ticker := time.NewTicker(time.Duration(cfg.Net.Interval) * time.Second)
		defer ticker.Stop()

		var prevStats map[string]gopsnet.IOCountersStat
		intervalSec := float64(cfg.Net.Interval)

		for {
			select {
			case <-ticker.C:
				var netwrokData []byte
				prevStats, netwrokData = monitorNet(prevStats, intervalSec)
				if len(netwrokData) > 0 {
					logger.Write(netwrokData)
				}
			case <-ctx.Done():
				utils.Log.Debug("[Network] 收集器已停止")
				return
			}
		}
	}()
}

// 過濾多餘網卡
func isSkipInterface(name string) bool {
	n := strings.ToLower(name)
	skipPrefixes := []string{
		"loopback", "isatap", "teredo", "virtualbox", "vmware",
		"npcap", "bluetooth", "hyper-v", "vethernet", "local area connection",
		"lo", "docker", "cni", "veth", "br-", "kube", "flannel",
	}
	for _, p := range skipPrefixes {
		if strings.HasPrefix(n, strings.ToLower(p)) {
			return true
		}
	}
	return false
}

// 主流程：收集網卡資料並輸出 JSON（IPv4 優先）
func monitorNet(prev map[string]gopsnet.IOCountersStat, intervalSec float64) (map[string]gopsnet.IOCountersStat, []byte) {
	// 1️⃣ 取得所有 NIC 流量（gopsutil）
	stats, err := gopsnet.IOCounters(true)
	if err != nil {
		utils.Log.Error("[Network] 無法取得網路統計: %v", err)
		return prev, nil
	}

	// 2️⃣ 統計 TCP 連線狀態（gopsutil）
	tcpState := make(map[string]int)
	conns, err := gopsnet.Connections("tcp")
	if err != nil {
		utils.Log.Error("[Network] 無法取得連線: %v", err)
	} else {
		for _, c := range conns {
			tcpState[c.Status]++
		}
	}

	// 3️⃣ 取得標準 library 的 interfaces（用來拿 MAC / IP）
	stdIfaces, _ := stdnet.Interfaces()

	var interfaces []NetworkInterface

	for _, s := range stats {
		// 過濾不必要的 NIC
		if isSkipInterface(s.Name) {
			continue
		}

		// 計算每秒速率（若有 prev）
		var txRate, rxRate, txPPS, rxPPS float64
		if prev != nil {
			if p, ok := prev[s.Name]; ok {
				txRate = float64(s.BytesSent-p.BytesSent) / intervalSec
				rxRate = float64(s.BytesRecv-p.BytesRecv) / intervalSec
				txPPS = float64(s.PacketsSent-p.PacketsSent) / intervalSec
				rxPPS = float64(s.PacketsRecv-p.PacketsRecv) / intervalSec
			}
		}

		// 找對應 Interface 取得 MAC 與 IP（IPv4 優先）
		macAddr := ""
		ipAddr := ""
		for _, iface := range stdIfaces {
			if iface.Name != s.Name {
				continue
			}

			// MAC
			macAddr = iface.HardwareAddr.String()

			// 取所有 addr，並決定 IPv4 或 IPv6（IPv4 優先）
			addrs, _ := iface.Addrs()
			var ipv4Addr, ipv6Addr string
			for _, a := range addrs {
				// a.String() 可能會帶上 /mask，先移除
				ip := strings.Split(a.String(), "/")[0]
				if strings.Contains(ip, ":") {
					// IPv6
					if ipv6Addr == "" {
						ipv6Addr = ip
					}
				} else if ip != "" {
					// IPv4
					if ipv4Addr == "" {
						ipv4Addr = ip
					}
				}
			}
			if ipv4Addr != "" {
				ipAddr = ipv4Addr
			} else {
				ipAddr = ipv6Addr
			}
			break
		}

		interfaces = append(interfaces, NetworkInterface{
			Name:      s.Name,
			IP:        ipAddr,
			MAC:       macAddr,
			Tx:        uint64(txRate),
			Rx:        uint64(rxRate),
			TxPPS:     txPPS,
			RxPPS:     rxPPS,
			TCP:       tcpState,
			Timestamp: time.Now().Format(time.RFC3339),
		})
	}

	// 4️⃣ 整理成 JSON 並一行輸出
	data := NetworkJSON{
		Category:   "NETWORK",
		Interfaces: interfaces,
	}
	b, _ := json.Marshal(data)
	s := string(b)
	utils.Log.Debug("%s", s)

	// 5️⃣ 準備下一輪 diff
	newPrev := make(map[string]gopsnet.IOCountersStat)
	for _, s := range stats {
		newPrev[s.Name] = s
	}
	return newPrev, b
}
