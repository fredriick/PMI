package peersdk

import (
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// getBatteryLevel reads battery level from /sys/class/power_supply on Linux.
func (p *PeerSDK) getBatteryLevel() int {
	if runtime.GOOS != "linux" {
		return 100
	}

	capacity, err := os.ReadFile("/sys/class/power_supply/BAT0/capacity")
	if err != nil {
		return 100
	}

	level, err := strconv.Atoi(strings.TrimSpace(string(capacity)))
	if err != nil {
		return 100
	}
	return level
}

// isCharging reads charging status from /sys/class/power_supply on Linux.
func (p *PeerSDK) isCharging() bool {
	if runtime.GOOS != "linux" {
		return true
	}

	status, err := os.ReadFile("/sys/class/power_supply/BAT0/status")
	if err != nil {
		return true
	}

	return strings.TrimSpace(string(status)) == "Charging"
}

// getCPUUsage calculates CPU usage by reading /proc/stat twice.
func (p *PeerSDK) getCPUUsage() float64 {
	if runtime.GOOS != "linux" {
		return 10.0
	}

	idle1, total1 := readCPUStat()
	time.Sleep(500 * time.Millisecond)
	idle2, total2 := readCPUStat()

	idleDelta := float64(idle2 - idle1)
	totalDelta := float64(total2 - total1)

	if totalDelta == 0 {
		return 0
	}

	usage := 100.0 * (1.0 - idleDelta/totalDelta)
	if usage < 0 {
		return 0
	}
	return usage
}

func readCPUStat() (idle, total uint64) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, 0
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 {
		return 0, 0
	}

	fields := strings.Fields(lines[0])
	if len(fields) < 5 || fields[0] != "cpu" {
		return 0, 0
	}

	var vals []uint64
	for _, f := range fields[1:] {
		v, _ := strconv.ParseUint(f, 10, 64)
		vals = append(vals, v)
	}

	total = 0
	for _, v := range vals {
		total += v
	}
	if len(vals) > 3 {
		idle = vals[3]
	}

	return idle, total
}

// isOnUnmeteredWiFi checks if the connection is unmetered.
// On Linux, we check for WiFi interface type and assume unmetered for wired.
func (p *PeerSDK) isOnUnmeteredWiFi() bool {
	if runtime.GOOS != "linux" {
		return true
	}

	interfaces, err := net.Interfaces()
	if err != nil {
		return true
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		typePath := "/sys/class/net/" + iface.Name + "/type"
		typeData, err := os.ReadFile(typePath)
		if err != nil {
			continue
		}

		ifaceType := strings.TrimSpace(string(typeData))
		if ifaceType == "1" {
			return true
		}
	}

	return true
}

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
