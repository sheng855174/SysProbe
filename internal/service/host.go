package service

import (
	"context"
	"sync"
	"sysprobe/internal/utils"
	"time"
)

var (
	currentHost utils.HostInfo
	hostMutex   sync.RWMutex
)

// StartHostUpdater 啟動 ticker，每 interval 更新 HostInfo
func StartHostUpdater(ctx context.Context, interval time.Duration) {
	// 先抓一次
	updateHost()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				updateHost()
			case <-ctx.Done():
				return
			}
		}
	}()
}

// updateHost 更新全域 hostInfo
func updateHost() {
	info := utils.GetHostInfo()

	hostMutex.Lock()
	currentHost = info
	hostMutex.Unlock()

	utils.Log.Debug("[HostInfo] Updated Hostname=%s IPs=%v", info.Hostname, info.IPs)
}

// GetCurrentHost 提供其他模組取得最新 hostInfo
func GetCurrentHost() utils.HostInfo {
	hostMutex.RLock()
	defer hostMutex.RUnlock()
	return currentHost
}
