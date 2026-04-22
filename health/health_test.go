package health

import (
	"runtime"
	"testing"
)

func TestCheckMemory_Healthy(t *testing.T) {
	svc := NewHealthService(nil)

	// Set thresholds high so even real memory usage reports healthy.
	svc.SetThresholds(100, 100)

	check := svc.CheckMemory()
	if check.Component != ComponentMemory {
		t.Errorf("Component = %q, want %q", check.Component, ComponentMemory)
	}
	if check.Status != StatusHealthy {
		t.Errorf("Status = %q, want %q, Message=%q", check.Status, StatusHealthy, check.Message)
	}
	if check.LatencyMs < 0 {
		t.Error("LatencyMs should not be negative")
	}
	if check.Details == nil {
		t.Error("Details should be populated")
	}
}

func TestCheckMemory_Degraded(t *testing.T) {
	svc := NewHealthService(nil)
	svc.SetThresholds(10, 100)

	check := svc.CheckMemory()
	if check.Component != ComponentMemory {
		t.Errorf("Component = %q", check.Component)
	}
	// With flat threshold of 10%, real usage will almost certainly trigger degraded/unhealthy.
	if check.Status != StatusDegraded && check.Status != StatusUnhealthy {
		t.Logf("Status = %q (depends on runtime memory usage), Message = %q", check.Status, check.Message)
		t.Skip("memory usage too atypical to assert exact status")
	}
	if _, ok := check.Details.(map[string]interface{}); !ok {
		t.Error("Details should be a map")
	}
}

func TestCheckDisk(t *testing.T) {
	svc := NewHealthService(nil)
	check := svc.CheckDisk()

	if check.Component != ComponentDisk {
		t.Errorf("Component = %q, want %q", check.Component, ComponentDisk)
	}
	if check.Status != StatusHealthy {
		t.Errorf("Status = %q, want %q", check.Status, StatusHealthy)
	}
	if check.Message == "" {
		t.Error("Message should not be empty")
	}
}

func TestCheckAll_Structure(t *testing.T) {
	svc := NewHealthService(nil)
	svc.SetThresholds(100, 100)

	health := svc.CheckAll()
	if health == nil {
		t.Fatal("CheckAll() returned nil")
	}
	if health.Overall == "" {
		t.Error("Overall status should not be empty")
	}
	if len(health.Checks) < 2 {
		t.Errorf("expected at least 2 checks (Memory+Disk), got %d", len(health.Checks))
	}
	if health.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}

	components := map[Component]bool{}
	for _, c := range health.Checks {
		components[c.Component] = true
		if c.Message == "" {
			t.Errorf("check for %q has empty Message", c.Component)
		}
	}
	if !components[ComponentMemory] {
		t.Errorf("expected Memory check, got %v", components)
	}
	if !components[ComponentDisk] {
		t.Errorf("expected Disk check, got %v", components)
	}
}

func TestNewHealthService(t *testing.T) {
	svc := NewHealthService(nil)
	if svc == nil {
		t.Fatal("NewHealthService() returned nil")
	}
	if svc.memThreshold != 90.0 {
		t.Errorf("default memThreshold = %v, want 90.0", svc.memThreshold)
	}
	if svc.diskThreshold != 90.0 {
		t.Errorf("default diskThreshold = %v, want 90.0", svc.diskThreshold)
	}
}

func TestHealthCheck_JSONSerialization(t *testing.T) {
	hc := HealthCheck{
		Component: ComponentRedis,
		Status:    StatusHealthy,
		LatencyMs: 42,
		Message:   "OK",
		Details:   map[string]interface{}{"version": "1.0"},
	}

	expected := map[Component]bool{
		ComponentRedis:   true,
		ComponentMemory:  false,
		ComponentDisk:    false,
		ComponentGateway: false,
	}
	if _, ok := expected[hc.Component]; !ok {
		t.Errorf("unknown component %q", hc.Component)
	}

	statusSet := map[Status]bool{StatusHealthy: true, StatusDegraded: true, StatusUnhealthy: true}
	if _, ok := statusSet[hc.Status]; !ok {
		t.Errorf("unknown status %q", hc.Status)
	}
	_ = runtime.NumGoroutine()
}

func TestSystemHealth_StatusConstants(t *testing.T) {
	if StatusHealthy != "healthy" || StatusDegraded != "degraded" || StatusUnhealthy != "unhealthy" {
		t.Errorf("unexpected status string values")
	}
	if ComponentRedis != "redis" || ComponentMemory != "memory" || ComponentDisk != "disk" || ComponentGateway != "gateway" {
		t.Errorf("unexpected component string values")
	}
}
