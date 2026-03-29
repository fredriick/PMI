package peersdk

import (
	"net"
	"os"
	"runtime"
	"strconv"
	"time"
)

// getLocalIP returns the first non-loopback IPv4 address.
func (p *PeerSDK) getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() {
			continue
		}
		if ipNet.IP.To4() != nil {
			return ipNet.IP.String()
		}
	}

	return "127.0.0.1"
}

// GetSystemInfo returns a map of system information.
func (p *PeerSDK) GetSystemInfo() map[string]string {
	hostname, _ := os.Hostname()
	return map[string]string{
		"os":       runtime.GOOS,
		"arch":     runtime.GOARCH,
		"hostname": hostname,
		"cpus":     strconv.Itoa(runtime.NumCPU()),
		"time":     time.Now().Format(time.RFC3339),
	}
}
