package service

import (
	"context"
	"net"
	"os"
	"sync"
	"sysprobe/internal/utils"
	"time"
)

type HostInfo struct {
	UUID     string   `json:"UUID"`
	Hostname string   `json:"Hostname"`
	IPs      []string `json:"Ips"`
}

// HostUpdater 封裝 HostInfo 更新邏輯
type HostUpdater struct {
	info HostInfo
	mu   sync.RWMutex
	ctx  context.Context
}

func GetHostInfo(uuid string) HostInfo {
	host := HostInfo{}
	host.UUID = uuid

	// 主機名稱
	name, err := os.Hostname()
	if err == nil {
		host.Hostname = name
	}

	// IP 列表
	ifaces, err := net.Interfaces()
	if err != nil {
		return host
	}

	for _, iface := range ifaces {
		// 排除沒 UP 的網卡
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil {
				continue
			}

			// 只留 IPv4 & 非 loopback
			if ip.To4() != nil && !ip.IsLoopback() {
				host.IPs = append(host.IPs, ip.String())
			}
		}
	}

	return host
}

// NewHostUpdater 建立 HostUpdater 並啟動定時更新
func NewHostUpdater(ctx context.Context, interval time.Duration, uuid string) *HostUpdater {
	h := &HostUpdater{
		ctx: ctx,
	}
	// 先抓一次 host info
	h.info.UUID = uuid
	h.update(uuid)

	go h.run(interval, uuid)
	return h
}

// run 啟動 ticker，定期更新 HostInfo
func (h *HostUpdater) run(interval time.Duration, uuid string) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.update(uuid)
		case <-h.ctx.Done():
			return
		}
	}
}

// update 取得最新 HostInfo 並寫入 struct
func (h *HostUpdater) update(uuid string) {
	info := GetHostInfo(uuid)

	h.mu.Lock()
	h.info = info
	h.mu.Unlock()

	utils.Log.Debug("[HostInfo] Updated Hostname=%s IPs=%v", info.Hostname, info.IPs)
}

// Get 提供其他函數取得最新 HostInfo
func (h *HostUpdater) Get() HostInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.info
}
