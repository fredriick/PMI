package health

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
)

func TestHealthService_New(t *testing.T) {
	hs := NewHealthService(nil)
	if hs == nil {
		t.Fatal("NewHealthService returned nil")
	}
	if hs.memThreshold != 90.0 {
		t.Errorf("expected memThreshold 90, got %f", hs.memThreshold)
	}
	if hs.diskThreshold != 90.0 {
		t.Errorf("expected diskThreshold 90, got %f", hs.diskThreshold)
	}
}

func TestHealthService_CheckAll_NoRedis(t *testing.T) {
	hs := NewHealthService(nil)
	result := hs.CheckAll()
	if result == nil {
		t.Fatal("expected non-nil SystemHealth")
	}
	if result.Overall == "" {
		t.Error("expected non-empty overall status")
	}
	if len(result.Checks) < 2 {
		t.Errorf("expected at least 2 checks (memory, disk), got %d", len(result.Checks))
	}
}

func TestHealthService_CheckMemory(t *testing.T) {
	hs := NewHealthService(nil)
	check := hs.CheckMemory()
	if check.Component != ComponentMemory {
		t.Errorf("expected ComponentMemory, got %s", check.Component)
	}
	if check.Status == "" {
		t.Error("expected non-empty status")
	}
	if check.Details == nil {
		t.Error("expected non-nil details")
	}
}

func TestHealthService_CheckDisk(t *testing.T) {
	hs := NewHealthService(nil)
	check := hs.CheckDisk()
	if check.Component != ComponentDisk {
		t.Errorf("expected ComponentDisk, got %s", check.Component)
	}
	if check.Status != StatusHealthy {
		t.Errorf("expected StatusHealthy, got %s", check.Status)
	}
}

func TestHealthService_SetThresholds(t *testing.T) {
	hs := NewHealthService(nil)
	hs.SetThresholds(80.0, 70.0)
	if hs.memThreshold != 80.0 {
		t.Errorf("expected memThreshold 80, got %f", hs.memThreshold)
	}
	if hs.diskThreshold != 70.0 {
		t.Errorf("expected diskThreshold 70, got %f", hs.diskThreshold)
	}
}

func TestHealthService_CheckAll_WithRedis(t *testing.T) {
	client := redisClientHealth(t)
	defer client.Close()

	hs := NewHealthService(client)
	result := hs.CheckAll()
	if result == nil {
		t.Fatal("expected non-nil SystemHealth")
	}
	if len(result.Checks) < 3 {
		t.Errorf("expected at least 3 checks (redis, memory, disk), got %d", len(result.Checks))
	}
}

func TestHealthService_CheckRedis(t *testing.T) {
	client := redisClientHealth(t)
	defer client.Close()

	hs := NewHealthService(client)
	check := hs.CheckRedis()
	if check.Component != ComponentRedis {
		t.Errorf("expected ComponentRedis, got %s", check.Component)
	}
	if check.Status != StatusHealthy {
		t.Errorf("expected StatusHealthy, got %s", check.Status)
	}
}

func TestComponent_Constants(t *testing.T) {
	if ComponentRedis != "redis" {
		t.Errorf("expected 'redis', got %s", ComponentRedis)
	}
	if ComponentMemory != "memory" {
		t.Errorf("expected 'memory', got %s", ComponentMemory)
	}
	if ComponentDisk != "disk" {
		t.Errorf("expected 'disk', got %s", ComponentDisk)
	}
	if ComponentGateway != "gateway" {
		t.Errorf("expected 'gateway', got %s", ComponentGateway)
	}
}

func TestStatus_Constants(t *testing.T) {
	if StatusHealthy != "healthy" {
		t.Errorf("expected 'healthy', got %s", StatusHealthy)
	}
	if StatusDegraded != "degraded" {
		t.Errorf("expected 'degraded', got %s", StatusDegraded)
	}
	if StatusUnhealthy != "unhealthy" {
		t.Errorf("expected 'unhealthy', got %s", StatusUnhealthy)
	}
}

func TestSystemHealth_Struct(t *testing.T) {
	sh := SystemHealth{
		Overall:   StatusHealthy,
		Checks:    []HealthCheck{},
		Timestamp: time.Now(),
	}
	if sh.Overall != StatusHealthy {
		t.Error("unexpected Overall")
	}
	if sh.Checks == nil {
		t.Error("expected non-nil Checks")
	}
}

func TestHealthCheck_Struct(t *testing.T) {
	hc := HealthCheck{
		Component: ComponentRedis,
		Status:    StatusHealthy,
		LatencyMs: 5,
		Message:   "Connected",
		Details:   map[string]interface{}{"key": "value"},
	}
	if hc.Component != ComponentRedis {
		t.Error("unexpected Component")
	}
	if hc.LatencyMs != 5 {
		t.Error("unexpected LatencyMs")
	}
}

func redisClientHealth(t *testing.T) *redis.Client {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379", DB: 15})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	_ = rdb.FlushDB(context.Background()).Err()
	return rdb
}
