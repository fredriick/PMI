package gateway

import (
	"testing"
	"time"
)

func TestNodePoolConfig_ApplyToPool(t *testing.T) {
	npc := &NodePoolConfig{
		MaxIdle:     10,
		MaxActive:   100,
		IdleTimeout: 5 * time.Minute,
		Wait:        true,
		MaxConnLife: 30 * time.Minute,
	}
	npc.ApplyToPool(nil)
}

func TestPoolConfigManager_New(t *testing.T) {
	pcm := NewPoolConfigManager()
	if pcm == nil {
		t.Fatal("NewPoolConfigManager returned nil")
	}
	if pcm.configs == nil {
		t.Error("expected configs map to be initialized")
	}
}

func TestPoolConfigManager_SetAndGetConfig(t *testing.T) {
	pcm := NewPoolConfigManager()
	cfg := NodePoolConfig{
		MaxIdle:     20,
		MaxActive:   200,
		IdleTimeout: 10 * time.Minute,
		Wait:        false,
		MaxConnLife: 60 * time.Minute,
	}
	pcm.SetConfig("node-1", cfg)

	got := pcm.GetConfig("node-1")
	if got.MaxIdle != 20 {
		t.Errorf("expected MaxIdle 20, got %d", got.MaxIdle)
	}
	if got.MaxActive != 200 {
		t.Errorf("expected MaxActive 200, got %d", got.MaxActive)
	}
}

func TestPoolConfigManager_GetConfig_NotFound(t *testing.T) {
	pcm := NewPoolConfigManager()
	cfg := pcm.GetConfig("unknown")
	if cfg.MaxIdle != 0 {
		t.Errorf("expected zero value for unknown node, got %+v", cfg)
	}
}

func TestPoolConfigManager_SetDefault(t *testing.T) {
	pcm := NewPoolConfigManager()
	cfg := NodePoolConfig{MaxIdle: 5}
	pcm.SetDefault(cfg)

	got := pcm.GetConfig("_default")
	if got.MaxIdle != 5 {
		t.Errorf("expected MaxIdle 5, got %d", got.MaxIdle)
	}
}

func TestPoolConfigManager_ListConfig(t *testing.T) {
	pcm := NewPoolConfigManager()
	pcm.SetConfig("a", NodePoolConfig{MaxIdle: 1})
	pcm.SetConfig("b", NodePoolConfig{MaxIdle: 2})

	if len(pcm.configs) != 2 {
		t.Errorf("expected 2 configs, got %d", len(pcm.configs))
	}
}
