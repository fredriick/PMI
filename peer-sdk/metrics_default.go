//go:build !linux && !darwin && !windows

package peersdk

// getBatteryLevel returns 100 on unsupported platforms.
func (p *PeerSDK) getBatteryLevel() int {
	return 100
}

// isCharging returns true on unsupported platforms.
func (p *PeerSDK) isCharging() bool {
	return true
}

// getCPUUsage returns 10% on unsupported platforms.
func (p *PeerSDK) getCPUUsage() float64 {
	return 10.0
}

// isOnUnmeteredWiFi returns true on unsupported platforms.
func (p *PeerSDK) isOnUnmeteredWiFi() bool {
	return true
}
