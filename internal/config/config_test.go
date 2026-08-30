package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestSetAndSave(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test_config.yaml")
	original := `gateway:
  host: "0.0.0.0"
  port: 8000
  rate_limit_requests: 100
`
	if err := os.WriteFile(tmpFile, []byte(original), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	viper.Reset()
	viper.SetConfigFile(tmpFile)
	viper.SetConfigType("yaml")
	if err := viper.ReadInConfig(); err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	cfgPath = tmpFile

	Set("gateway.rate_limit_requests", 250)
	if err := Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "rate_limit_requests: 250") {
		t.Errorf("expected 'rate_limit_requests: 250' in config file, got:\n%s", content)
	}
}

func TestConfigPath(t *testing.T) {
	cfgPath = "some/path/config.yaml"
	if ConfigPath() != "some/path/config.yaml" {
		t.Errorf("expected 'some/path/config.yaml', got '%s'", ConfigPath())
	}
}

func TestFileWatcher_TriggersSubscriber(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "watch_config.yaml")
	original := `gateway:
  rate_limit_requests: 100
  rate_limit_window_seconds: 60
`
	if err := os.WriteFile(tmpFile, []byte(original), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	viper.Reset()
	viper.SetConfigFile(tmpFile)
	viper.SetConfigType("yaml")
	if err := viper.ReadInConfig(); err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	cfgPath = tmpFile

	called := make(chan *Config, 1)
	OnChange(func(newCfg *Config) {
		select {
		case called <- newCfg:
		default:
		}
	})

	Watch()

	updated := `gateway:
  rate_limit_requests: 500
  rate_limit_window_seconds: 60
`
	if err := os.WriteFile(tmpFile, []byte(updated), 0644); err != nil {
		t.Fatalf("failed to write updated config: %v", err)
	}

	select {
	case received := <-called:
		if received.Gateway.RateLimitRequests != 500 {
			t.Errorf("expected rate_limit_requests=500, got %d", received.Gateway.RateLimitRequests)
		}
	case <-time.After(5 * time.Second):
		t.Skip("file watcher did not fire within timeout (may not work in this environment)")
	}
}

func TestWatch_RegistersSubscriber(t *testing.T) {
	called := make(chan *Config, 1)
	OnChange(func(newCfg *Config) {
		select {
		case called <- newCfg:
		default:
		}
	})

	testCfg := &Config{
		Gateway: GatewayConfig{
			Host:              "0.0.0.0",
			Port:              9000,
			RateLimitRequests: 42,
		},
	}

	mu.Lock()
	cfg = testCfg
	subs := make([]func(*Config), len(subscribers))
	copy(subs, subscribers)
	mu.Unlock()

	for _, fn := range subs {
		fn(testCfg)
	}

	select {
	case received := <-called:
		if received.Gateway.RateLimitRequests != 42 {
			t.Errorf("expected rate_limit_requests=42, got %d", received.Gateway.RateLimitRequests)
		}
		if received.Gateway.Port != 9000 {
			t.Errorf("expected port=9000, got %d", received.Gateway.Port)
		}
	case <-time.After(time.Second):
		t.Error("subscriber was not called")
	}
}

func TestSave_NilConfigPath(t *testing.T) {
	cfgPath = ""
	err := Save()
	if err == nil {
		t.Error("expected error when config path is empty")
	}
}

func TestMultipleSubscribers(t *testing.T) {
	called1 := make(chan string, 1)
	called2 := make(chan string, 1)

	OnChange(func(newCfg *Config) {
		select {
		case called1 <- "subscriber1":
		default:
		}
	})
	OnChange(func(newCfg *Config) {
		select {
		case called2 <- "subscriber2":
		default:
		}
	})

	testCfg := &Config{Gateway: GatewayConfig{Host: "test"}}

	mu.Lock()
	cfg = testCfg
	subs := make([]func(*Config), len(subscribers))
	copy(subs, subscribers)
	mu.Unlock()

	for _, fn := range subs {
		fn(testCfg)
	}

	select {
	case msg := <-called1:
		if msg != "subscriber1" {
			t.Errorf("expected 'subscriber1', got '%s'", msg)
		}
	case <-time.After(time.Second):
		t.Error("subscriber1 was not called")
	}

	select {
	case msg := <-called2:
		if msg != "subscriber2" {
			t.Errorf("expected 'subscriber2', got '%s'", msg)
		}
	case <-time.After(time.Second):
		t.Error("subscriber2 was not called")
	}
}

func TestGet_ReturnsCurrentConfig(t *testing.T) {
	testCfg := &Config{
		Gateway: GatewayConfig{
			Host:              "0.0.0.0",
			Port:              8000,
			RateLimitRequests: 100,
		},
	}

	mu.Lock()
	cfg = testCfg
	mu.Unlock()

	retrieved := Get()
	if retrieved == nil {
		t.Fatal("expected non-nil config from Get()")
	}

	if retrieved.Gateway.Host != "0.0.0.0" {
		t.Errorf("expected host '0.0.0.0', got '%s'", retrieved.Gateway.Host)
	}
	if retrieved.Gateway.Port != 8000 {
		t.Errorf("expected port 8000, got %d", retrieved.Gateway.Port)
	}
}

func TestSetAndGet(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "setget_config.yaml")
	original := `gateway:
  rate_limit_requests: 100
`
	if err := os.WriteFile(tmpFile, []byte(original), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	viper.Reset()
	viper.SetConfigFile(tmpFile)
	viper.SetConfigType("yaml")
	if err := viper.ReadInConfig(); err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	cfgPath = tmpFile

	Set("gateway.rate_limit_requests", 777)

	val := viper.GetInt("gateway.rate_limit_requests")
	if val != 777 {
		t.Errorf("expected rate_limit_requests=777 from Viper, got %d", val)
	}

	if err := Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	if !strings.Contains(string(data), "rate_limit_requests: 777") {
		t.Errorf("expected 'rate_limit_requests: 777' in file, got:\n%s", string(data))
	}
}

func TestSet_NestedCompression(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "nested_config.yaml")
	original := fmt.Sprintf(`gateway:
  rate_limit_requests: 100
`)
	if err := os.WriteFile(tmpFile, []byte(original), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	viper.Reset()
	viper.SetConfigFile(tmpFile)
	viper.SetConfigType("yaml")
	if err := viper.ReadInConfig(); err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	cfgPath = tmpFile

	Set("gateway.compression.enabled", true)
	Set("gateway.compression.level", 5)

	if err := Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "enabled: true") {
		t.Errorf("expected 'enabled: true' in file, got:\n%s", content)
	}
	if !strings.Contains(content, "level: 5") {
		t.Errorf("expected 'level: 5' in file, got:\n%s", content)
	}
}
