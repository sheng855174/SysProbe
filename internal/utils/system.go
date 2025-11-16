package utils

import (
	"net"
	"os"
)

type HostInfo struct {
	Hostname string   `json:"hostname"`
	IPs      []string `json:"ips"`
}

func GetHostInfo() HostInfo {
	host := HostInfo{}

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
