//go:build darwin

package peersdk

import (
	"os/exec"
	"strconv"
	"strings"
)

func (p *PeerSDK) getBatteryLevel() int {
	out, err := exec.Command("pmset", "-g", "batt").Output()
	if err != nil {
		return 100
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "%") {
			idx := strings.Index(line, "%")
			start := idx
			for start > 0 && line[start-1] >= '0' && line[start-1] <= '9' {
				start--
			}
			if level, err := strconv.Atoi(line[start:idx]); err == nil {
				return level
			}
		}
	}
	return 100
}

func (p *PeerSDK) isCharging() bool {
	out, err := exec.Command("pmset", "-g", "batt").Output()
	if err != nil {
		return true
	}
	return strings.Contains(string(out), "AC Power") || strings.Contains(string(out), "charging")
}

func (p *PeerSDK) getCPUUsage() float64 {
	out, err := exec.Command("top", "-l", "1", "-n", "0").Output()
	if err != nil {
		return 10.0
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "CPU usage") {
			if idx := strings.Index(line, "idle"); idx > 0 {
				end := idx - 1
				start := end
				for start > 0 && (line[start-1] >= '0' && line[start-1] <= '9' || line[start-1] == '.') {
					start--
				}
				if idle, err := strconv.ParseFloat(line[start:end], 64); err == nil {
					return 100.0 - idle
				}
			}
		}
	}
	return 10.0
}

func (p *PeerSDK) isOnUnmeteredWiFi() bool {
	return true
}
