//go:build windows

package peersdk

import (
	"os/exec"
	"strconv"
	"strings"
)

func (p *PeerSDK) getBatteryLevel() int {
	out, err := exec.Command("wmic", "path", "Win32_Battery", "get", "EstimatedChargeRemaining", "/value").Output()
	if err != nil {
		return 100
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "EstimatedChargeRemaining=") {
			val := strings.TrimPrefix(line, "EstimatedChargeRemaining=")
			if level, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
				return level
			}
		}
	}
	return 100
}

func (p *PeerSDK) isCharging() bool {
	out, err := exec.Command("wmic", "path", "Win32_Battery", "get", "BatteryStatus", "/value").Output()
	if err != nil {
		return true
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "BatteryStatus=") {
			val := strings.TrimPrefix(line, "BatteryStatus=")
			status, _ := strconv.Atoi(strings.TrimSpace(val))
			return status == 2 || status == 6
		}
	}
	return true
}

func (p *PeerSDK) getCPUUsage() float64 {
	out, err := exec.Command("wmic", "cpu", "get", "loadpercentage", "/value").Output()
	if err != nil {
		return 10.0
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "LoadPercentage=") {
			val := strings.TrimPrefix(line, "LoadPercentage=")
			if pct, err := strconv.ParseFloat(strings.TrimSpace(val), 64); err == nil {
				return pct
			}
		}
	}
	return 10.0
}

func (p *PeerSDK) isOnUnmeteredWiFi() bool {
	return true
}
