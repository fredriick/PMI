//go:build linux

package peersdk

import (
	"os"
	"strconv"
	"strings"
	"time"
)

func (p *PeerSDK) getBatteryLevel() int {
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

func (p *PeerSDK) isCharging() bool {
	status, err := os.ReadFile("/sys/class/power_supply/BAT0/status")
	if err != nil {
		return true
	}
	return strings.TrimSpace(string(status)) == "Charging"
}

func (p *PeerSDK) getCPUUsage() float64 {
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

func (p *PeerSDK) isOnUnmeteredWiFi() bool {
	entries, err := os.ReadDir("/sys/class/net/")
	if err != nil {
		return true
	}
	for _, entry := range entries {
		if entry.Name() == "lo" {
			continue
		}
		typePath := "/sys/class/net/" + entry.Name() + "/type"
		typeData, err := os.ReadFile(typePath)
		if err != nil {
			continue
		}
		if strings.TrimSpace(string(typeData)) == "1" {
			return true
		}
	}
	return true
}
