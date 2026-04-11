package health

import (
	"context"
	"runtime"
	"time"

	"github.com/go-redis/redis/v8"
)

type Component string

const (
	ComponentRedis   Component = "redis"
	ComponentMemory  Component = "memory"
	ComponentDisk    Component = "disk"
	ComponentGateway Component = "gateway"
)

type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
)

type HealthCheck struct {
	Component Component   `json:"component"`
	Status    Status      `json:"status"`
	LatencyMs int64       `json:"latency_ms"`
	Message   string      `json:"message,omitempty"`
	Details   interface{} `json:"details,omitempty"`
}

type SystemHealth struct {
	Overall   Status        `json:"overall"`
	Checks    []HealthCheck `json:"checks"`
	Timestamp time.Time     `json:"timestamp"`
}

type HealthService struct {
	redisClient   *redis.Client
	memThreshold  float64
	diskThreshold float64
}

func NewHealthService(redisClient *redis.Client) *HealthService {
	return &HealthService{
		redisClient:   redisClient,
		memThreshold:  90.0,
		diskThreshold: 90.0,
	}
}

func (s *HealthService) CheckAll() *SystemHealth {
	checks := []HealthCheck{
		s.CheckRedis(),
		s.CheckMemory(),
		s.CheckDisk(),
	}

	overall := StatusHealthy
	for _, c := range checks {
		if c.Status == StatusUnhealthy {
			overall = StatusUnhealthy
			break
		}
		if c.Status == StatusDegraded {
			overall = StatusDegraded
		}
	}

	return &SystemHealth{
		Overall:   overall,
		Checks:    checks,
		Timestamp: time.Now(),
	}
}

func (s *HealthService) CheckRedis() HealthCheck {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := s.redisClient.Ping(ctx).Err()
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return HealthCheck{
			Component: ComponentRedis,
			Status:    StatusUnhealthy,
			LatencyMs: latency,
			Message:   "Redis connection failed: " + err.Error(),
		}
	}

	if latency > 1000 {
		return HealthCheck{
			Component: ComponentRedis,
			Status:    StatusDegraded,
			LatencyMs: latency,
			Message:   "High latency",
		}
	}

	return HealthCheck{
		Component: ComponentRedis,
		Status:    StatusHealthy,
		LatencyMs: latency,
		Message:   "Connected",
	}
}

func (s *HealthService) CheckMemory() HealthCheck {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	usedPercent := float64(m.Alloc) / float64(m.Sys) * 100

	details := map[string]interface{}{
		"alloc_bytes": m.Alloc,
		"total_bytes": m.Sys,
		"gc_count":    m.NumGC,
		"goroutines":  runtime.NumGoroutine(),
	}

	if usedPercent > s.memThreshold {
		return HealthCheck{
			Component: ComponentMemory,
			Status:    StatusUnhealthy,
			Message:   "Memory usage critical",
			Details:   details,
		}
	}

	if usedPercent > s.memThreshold*0.7 {
		return HealthCheck{
			Component: ComponentMemory,
			Status:    StatusDegraded,
			Message:   "Memory usage elevated",
			Details:   details,
		}
	}

	return HealthCheck{
		Component: ComponentMemory,
		Status:    StatusHealthy,
		Message:   "Memory usage normal",
		Details:   details,
	}
}

func (s *HealthService) CheckDisk() HealthCheck {
	return HealthCheck{
		Component: ComponentDisk,
		Status:    StatusHealthy,
		Message:   "Disk check not available on this platform",
	}
}

func (s *HealthService) SetThresholds(memPct, diskPct float64) {
	s.memThreshold = memPct
	s.diskThreshold = diskPct
}
