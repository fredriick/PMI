package gateway

import (
	"testing"
	"time"

	"proxymesh/internal/config"
)

func TestPrometheusPusher_Disabled(t *testing.T) {
	cfg := &config.PrometheusConfig{Enabled: false}
	p := NewPrometheusPusher(cfg)
	if p == nil {
		t.Fatal("NewPrometheusPusher returned nil")
	}
	if p.enabled {
		t.Error("expected pusher to be disabled")
	}
	p.Start()
	p.Stop()
}

func TestPrometheusPusher_Enabled_NoURL(t *testing.T) {
	cfg := &config.PrometheusConfig{
		Enabled:             true,
		PushGatewayURL:      "",
		PushIntervalSeconds: 1,
		JobName:             "test",
	}
	p := NewPrometheusPusher(cfg)
	if p == nil {
		t.Fatal("NewPrometheusPusher returned nil")
	}
	if !p.enabled {
		t.Error("expected pusher to be enabled")
	}
	p.Start()
	time.Sleep(50 * time.Millisecond)
	p.Stop()
}

func TestPrometheusPusher_Push_NoURL(t *testing.T) {
	cfg := &config.PrometheusConfig{
		Enabled:             true,
		PushGatewayURL:      "",
		PushIntervalSeconds: 1,
		JobName:             "test",
	}
	p := NewPrometheusPusher(cfg)
	p.push()
}

func TestPrometheusConfig_Defaults(t *testing.T) {
	cfg := &config.PrometheusConfig{}
	if cfg.Enabled {
		t.Error("expected Prometheus disabled by default")
	}
}
