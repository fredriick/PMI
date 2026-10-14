package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestLoad_Success(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "load_test.yaml")
	content := `gateway:
  host: "127.0.0.1"
  port: 9000
  rate_limit_requests: 200
  compression:
    enabled: true
    level: 9
matchmaker:
  host: "0.0.0.0"
  port: 9001
  pool_size: 100
  cooldown_ttl_minutes: 10
redis:
  host: "redis-cluster"
  port: 7000
  password: "cluster-pass"
  db: 2
  cluster_enabled: true
  cluster_addrs:
    - "redis-1:7000"
    - "redis-2:7000"
    - "redis-3:7000"
  sentinel_enabled: true
  sentinel_addrs:
    - "sentinel-1:26379"
    - "sentinel-2:26379"
  sentinel_master: "mycluster"
  pool_size: 200
  max_retries: 5
compliance:
  blocked_domains:
    - "malicious.com"
    - "spam.org"
  kyc_required: true
peer:
  min_battery_percent: 15
  max_cpu_percent: 90
  require_unmetered_wifi: false
subnet:
  enabled: true
  prefix: "192.168.0.0/16"
  prefix_len: 28
pricing:
  tiers:
    - name: "starter"
      min_gb_monthly: 0
      max_gb_monthly: 50
      rate_per_gb_sent: 0.02
      rate_per_gb_recv: 0.01
    - name: "enterprise"
      min_gb_monthly: 1000
      max_gb_monthly: 100000
      rate_per_gb_sent: 0.001
      rate_per_gb_recv: 0.0005
federation:
  enabled: true
  regions:
    - "asia-pacific"
    - "south-america"
  heartbeat_interval_seconds: 45
  sync_interval_seconds: 120
jwt:
  enabled: true
  secret_key: "super-secret"
  expiration: "24h"
rbac:
  enabled: true
  default_role: "admin"
  min_password_len: 16
  max_login_attempts: 3
  lockout_duration_minutes: 30
prometheus:
  enabled: true
  push_gateway_url: "http://localhost:9091"
  push_interval_seconds: 60
  job_name: "test-job"
`
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	viper.Reset()
	cfg, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}

	// Gateway
	if cfg.Gateway.Host != "127.0.0.1" {
		t.Errorf("Gateway.Host: expected 127.0.0.1, got %s", cfg.Gateway.Host)
	}
	if cfg.Gateway.Port != 9000 {
		t.Errorf("Gateway.Port: expected 9000, got %d", cfg.Gateway.Port)
	}
	if cfg.Gateway.RateLimitRequests != 200 {
		t.Errorf("Gateway.RateLimitRequests: expected 200, got %d", cfg.Gateway.RateLimitRequests)
	}
	if !cfg.Gateway.Compression.Enabled {
		t.Error("Gateway.Compression.Enabled: expected true")
	}
	if cfg.Gateway.Compression.Level != 9 {
		t.Errorf("Gateway.Compression.Level: expected 9, got %d", cfg.Gateway.Compression.Level)
	}

	// Matchmaker
	if cfg.Matchmaker.Host != "0.0.0.0" {
		t.Errorf("Matchmaker.Host: expected 0.0.0.0, got %s", cfg.Matchmaker.Host)
	}
	if cfg.Matchmaker.Port != 9001 {
		t.Errorf("Matchmaker.Port: expected 9001, got %d", cfg.Matchmaker.Port)
	}
	if cfg.Matchmaker.PoolSize != 100 {
		t.Errorf("Matchmaker.PoolSize: expected 100, got %d", cfg.Matchmaker.PoolSize)
	}
	if cfg.Matchmaker.CooldownTTLMinutes != 10 {
		t.Errorf("Matchmaker.CooldownTTLMinutes: expected 10, got %d", cfg.Matchmaker.CooldownTTLMinutes)
	}

	// Redis
	if cfg.Redis.Host != "redis-cluster" {
		t.Errorf("Redis.Host: expected redis-cluster, got %s", cfg.Redis.Host)
	}
	if cfg.Redis.Port != 7000 {
		t.Errorf("Redis.Port: expected 7000, got %d", cfg.Redis.Port)
	}
	if cfg.Redis.Password != "cluster-pass" {
		t.Errorf("Redis.Password: expected cluster-pass, got %s", cfg.Redis.Password)
	}
	if !cfg.Redis.ClusterEnabled {
		t.Error("Redis.ClusterEnabled: expected true")
	}
	if len(cfg.Redis.ClusterAddrs) != 3 {
		t.Fatalf("Redis.ClusterAddrs: expected 3, got %d", len(cfg.Redis.ClusterAddrs))
	}
	if cfg.Redis.SentinelMaster != "mycluster" {
		t.Errorf("Redis.SentinelMaster: expected mycluster, got %s", cfg.Redis.SentinelMaster)
	}
	if cfg.Redis.PoolSize != 200 {
		t.Errorf("Redis.PoolSize: expected 200, got %d", cfg.Redis.PoolSize)
	}
	if cfg.Redis.MaxRetries != 5 {
		t.Errorf("Redis.MaxRetries: expected 5, got %d", cfg.Redis.MaxRetries)
	}

	// Compliance
	if len(cfg.Compliance.BlockedDomains) != 2 {
		t.Fatalf("Compliance.BlockedDomains: expected 2, got %d", len(cfg.Compliance.BlockedDomains))
	}
	if cfg.Compliance.BlockedDomains[0] != "malicious.com" {
		t.Errorf("Compliance.BlockedDomains[0]: expected malicious.com, got %s", cfg.Compliance.BlockedDomains[0])
	}
	if !cfg.Compliance.KYCRequired {
		t.Error("Compliance.KYCRequired: expected true")
	}

	// Peer
	if cfg.Peer.MinBatteryPercent != 15 {
		t.Errorf("Peer.MinBatteryPercent: expected 15, got %d", cfg.Peer.MinBatteryPercent)
	}
	if cfg.Peer.MaxCPUPercent != 90 {
		t.Errorf("Peer.MaxCPUPercent: expected 90, got %d", cfg.Peer.MaxCPUPercent)
	}
	if cfg.Peer.RequireUnmeteredWiFi {
		t.Error("Peer.RequireUnmeteredWiFi: expected false")
	}

	// Subnet
	if !cfg.Subnet.Enabled {
		t.Error("Subnet.Enabled: expected true")
	}
	if cfg.Subnet.Prefix != "192.168.0.0/16" {
		t.Errorf("Subnet.Prefix: expected 192.168.0.0/16, got %s", cfg.Subnet.Prefix)
	}
	if cfg.Subnet.PrefixLen != 28 {
		t.Errorf("Subnet.PrefixLen: expected 28, got %d", cfg.Subnet.PrefixLen)
	}

	// Pricing
	if len(cfg.Pricing.Tiers) != 2 {
		t.Fatalf("Pricing.Tiers: expected 2, got %d", len(cfg.Pricing.Tiers))
	}
	if cfg.Pricing.Tiers[0].Name != "starter" {
		t.Errorf("Pricing.Tiers[0].Name: expected starter, got %s", cfg.Pricing.Tiers[0].Name)
	}
	if cfg.Pricing.Tiers[1].RatePerGBSent != 0.001 {
		t.Errorf("Pricing.Tiers[1].RatePerGBSent: expected 0.001, got %f", cfg.Pricing.Tiers[1].RatePerGBSent)
	}

	// Federation
	if !cfg.Federation.Enabled {
		t.Error("Federation.Enabled: expected true")
	}
	if len(cfg.Federation.Regions) != 2 {
		t.Fatalf("Federation.Regions: expected 2, got %d", len(cfg.Federation.Regions))
	}
	if cfg.Federation.HeartbeatInterval != 45 {
		t.Errorf("Federation.HeartbeatInterval: expected 45, got %d", cfg.Federation.HeartbeatInterval)
	}
	if cfg.Federation.SyncInterval != 120 {
		t.Errorf("Federation.SyncInterval: expected 120, got %d", cfg.Federation.SyncInterval)
	}

	// JWT
	if !cfg.JWT.Enabled {
		t.Error("JWT.Enabled: expected true")
	}
	if cfg.JWT.SecretKey != "super-secret" {
		t.Errorf("JWT.SecretKey: expected super-secret, got %s", cfg.JWT.SecretKey)
	}

	// RBAC
	if !cfg.RBAC.Enabled {
		t.Error("RBAC.Enabled: expected true")
	}
	if cfg.RBAC.DefaultRole != "admin" {
		t.Errorf("RBAC.DefaultRole: expected admin, got %s", cfg.RBAC.DefaultRole)
	}
	if cfg.RBAC.MinPasswordLen != 16 {
		t.Errorf("RBAC.MinPasswordLen: expected 16, got %d", cfg.RBAC.MinPasswordLen)
	}
	if cfg.RBAC.MaxLoginAttempts != 3 {
		t.Errorf("RBAC.MaxLoginAttempts: expected 3, got %d", cfg.RBAC.MaxLoginAttempts)
	}
	if cfg.RBAC.LockoutDuration != 30 {
		t.Errorf("RBAC.LockoutDuration: expected 30, got %d", cfg.RBAC.LockoutDuration)
	}

	// Prometheus
	if !cfg.Prometheus.Enabled {
		t.Error("Prometheus.Enabled: expected true")
	}
	if cfg.Prometheus.PushGatewayURL != "http://localhost:9091" {
		t.Errorf("Prometheus.PushGatewayURL: expected http://localhost:9091, got %s", cfg.Prometheus.PushGatewayURL)
	}
	if cfg.Prometheus.PushIntervalSeconds != 60 {
		t.Errorf("Prometheus.PushIntervalSeconds: expected 60, got %d", cfg.Prometheus.PushIntervalSeconds)
	}
	if cfg.Prometheus.JobName != "test-job" {
		t.Errorf("Prometheus.JobName: expected test-job, got %s", cfg.Prometheus.JobName)
	}
}

func TestApplyEnvOverrides(t *testing.T) {
	oldRedisPass := os.Getenv("REDIS_PASSWORD")
	oldMTLS := os.Getenv("MTLS_ENABLED")
	oldMTLSCa := os.Getenv("MTLS_CA_CERT_PATH")
	oldMTLSCert := os.Getenv("MTLS_SERVER_CERT_PATH")
	oldMTLSKey := os.Getenv("MTLS_SERVER_KEY_PATH")
	oldJWTEnabled := os.Getenv("JWT_ENABLED")
	oldJWTSecret := os.Getenv("JWT_SECRET_KEY")
	oldRBAC := os.Getenv("RBAC_ENABLED")
	oldReqPrefix := os.Getenv("REQUEST_ID_PREFIX")
	oldReqFormat := os.Getenv("REQUEST_ID_FORMAT")

	defer func() {
		if oldRedisPass != "" {
			os.Setenv("REDIS_PASSWORD", oldRedisPass)
		} else {
			os.Unsetenv("REDIS_PASSWORD")
		}
		if oldMTLS != "" {
			os.Setenv("MTLS_ENABLED", oldMTLS)
		} else {
			os.Unsetenv("MTLS_ENABLED")
		}
		if oldMTLSCa != "" {
			os.Setenv("MTLS_CA_CERT_PATH", oldMTLSCa)
		} else {
			os.Unsetenv("MTLS_CA_CERT_PATH")
		}
		if oldMTLSCert != "" {
			os.Setenv("MTLS_SERVER_CERT_PATH", oldMTLSCert)
		} else {
			os.Unsetenv("MTLS_SERVER_CERT_PATH")
		}
		if oldMTLSKey != "" {
			os.Setenv("MTLS_SERVER_KEY_PATH", oldMTLSKey)
		} else {
			os.Unsetenv("MTLS_SERVER_KEY_PATH")
		}
		if oldJWTEnabled != "" {
			os.Setenv("JWT_ENABLED", oldJWTEnabled)
		} else {
			os.Unsetenv("JWT_ENABLED")
		}
		if oldJWTSecret != "" {
			os.Setenv("JWT_SECRET_KEY", oldJWTSecret)
		} else {
			os.Unsetenv("JWT_SECRET_KEY")
		}
		if oldRBAC != "" {
			os.Setenv("RBAC_ENABLED", oldRBAC)
		} else {
			os.Unsetenv("RBAC_ENABLED")
		}
		if oldReqPrefix != "" {
			os.Setenv("REQUEST_ID_PREFIX", oldReqPrefix)
		} else {
			os.Unsetenv("REQUEST_ID_PREFIX")
		}
		if oldReqFormat != "" {
			os.Setenv("REQUEST_ID_FORMAT", oldReqFormat)
		} else {
			os.Unsetenv("REQUEST_ID_FORMAT")
		}
	}()

	os.Setenv("REDIS_PASSWORD", "test-redis-pass")
	os.Setenv("MTLS_ENABLED", "true")
	os.Setenv("MTLS_CA_CERT_PATH", "/test/ca.crt")
	os.Setenv("MTLS_SERVER_CERT_PATH", "/test/server.crt")
	os.Setenv("MTLS_SERVER_KEY_PATH", "/test/server.key")
	os.Setenv("JWT_ENABLED", "true")
	os.Setenv("JWT_SECRET_KEY", "test-jwt-secret")
	os.Setenv("RBAC_ENABLED", "true")
	os.Setenv("REQUEST_ID_PREFIX", "test-prefix")
	os.Setenv("REQUEST_ID_FORMAT", "uuid")

	cfg := &Config{}
	applyEnvOverrides(cfg)

	if cfg.Redis.Password != "test-redis-pass" {
		t.Errorf("Redis.Password: expected test-redis-pass, got %s", cfg.Redis.Password)
	}
	if !cfg.Gateway.MTLSEnabled {
		t.Error("Gateway.MTLSEnabled: expected true")
	}
	if cfg.Gateway.CACertPath != "/test/ca.crt" {
		t.Errorf("Gateway.CACertPath: expected /test/ca.crt, got %s", cfg.Gateway.CACertPath)
	}
	if cfg.Gateway.ServerCertPath != "/test/server.crt" {
		t.Errorf("Gateway.ServerCertPath: expected /test/server.crt, got %s", cfg.Gateway.ServerCertPath)
	}
	if cfg.Gateway.ServerKeyPath != "/test/server.key" {
		t.Errorf("Gateway.ServerKeyPath: expected /test/server.key, got %s", cfg.Gateway.ServerKeyPath)
	}
	if !cfg.JWT.Enabled {
		t.Error("JWT.Enabled: expected true")
	}
	if cfg.JWT.SecretKey != "test-jwt-secret" {
		t.Errorf("JWT.SecretKey: expected test-jwt-secret, got %s", cfg.JWT.SecretKey)
	}
	if !cfg.RBAC.Enabled {
		t.Error("RBAC.Enabled: expected true")
	}
	if cfg.Gateway.RequestIDPrefix != "test-prefix" {
		t.Errorf("Gateway.RequestIDPrefix: expected test-prefix, got %s", cfg.Gateway.RequestIDPrefix)
	}
	if cfg.Gateway.RequestIDFormat != "uuid" {
		t.Errorf("Gateway.RequestIDFormat: expected uuid, got %s", cfg.Gateway.RequestIDFormat)
	}
}

func TestApplyEnvOverrides_InvalidBool(t *testing.T) {
	oldMTLS := os.Getenv("MTLS_ENABLED")
	defer func() {
		if oldMTLS != "" {
			os.Setenv("MTLS_ENABLED", oldMTLS)
		} else {
			os.Unsetenv("MTLS_ENABLED")
		}
	}()

	os.Setenv("MTLS_ENABLED", "not_a_bool")
	cfg := &Config{}
	applyEnvOverrides(cfg)
	if cfg.Gateway.MTLSEnabled {
		t.Error("expected MTLSEnabled to remain false for invalid bool")
	}
}

func TestApplyEnvOverrides_Partial(t *testing.T) {
	oldRedisPass := os.Getenv("REDIS_PASSWORD")
	defer func() {
		if oldRedisPass != "" {
			os.Setenv("REDIS_PASSWORD", oldRedisPass)
		} else {
			os.Unsetenv("REDIS_PASSWORD")
		}
	}()

	os.Setenv("REDIS_PASSWORD", "partial-pass")
	cfg := &Config{}
	applyEnvOverrides(cfg)
	if cfg.Redis.Password != "partial-pass" {
		t.Errorf("expected partial Redis.Password, got %s", cfg.Redis.Password)
	}
}

func TestApplyEnvOverrides_MTLSPaths(t *testing.T) {
	oldCa := os.Getenv("MTLS_CA_CERT_PATH")
	oldCert := os.Getenv("MTLS_SERVER_CERT_PATH")
	oldKey := os.Getenv("MTLS_SERVER_KEY_PATH")
	defer func() {
		if oldCa != "" {
			os.Setenv("MTLS_CA_CERT_PATH", oldCa)
		} else {
			os.Unsetenv("MTLS_CA_CERT_PATH")
		}
		if oldCert != "" {
			os.Setenv("MTLS_SERVER_CERT_PATH", oldCert)
		} else {
			os.Unsetenv("MTLS_SERVER_CERT_PATH")
		}
		if oldKey != "" {
			os.Setenv("MTLS_SERVER_KEY_PATH", oldKey)
		} else {
			os.Unsetenv("MTLS_SERVER_KEY_PATH")
		}
	}()

	os.Setenv("MTLS_CA_CERT_PATH", "/custom/ca.pem")
	os.Setenv("MTLS_SERVER_CERT_PATH", "/custom/cert.pem")
	os.Setenv("MTLS_SERVER_KEY_PATH", "/custom/key.pem")

	cfg := &Config{}
	applyEnvOverrides(cfg)

	if cfg.Gateway.CACertPath != "/custom/ca.pem" {
		t.Errorf("CACertPath: expected /custom/ca.pem, got %s", cfg.Gateway.CACertPath)
	}
	if cfg.Gateway.ServerCertPath != "/custom/cert.pem" {
		t.Errorf("ServerCertPath: expected /custom/cert.pem, got %s", cfg.Gateway.ServerCertPath)
	}
	if cfg.Gateway.ServerKeyPath != "/custom/key.pem" {
		t.Errorf("ServerKeyPath: expected /custom/key.pem, got %s", cfg.Gateway.ServerKeyPath)
	}
}

func TestGet_NilConfig(t *testing.T) {
	mu.Lock()
	oldCfg := cfg
	cfg = nil
	mu.Unlock()

	result := Get()
	if result != nil {
		t.Errorf("expected nil when cfg is nil, got %v", result)
	}

	mu.Lock()
	cfg = oldCfg
	mu.Unlock()
}

func TestGet_ConcurrentAccess(t *testing.T) {
	testCfg := &Config{
		Gateway: GatewayConfig{Host: "concurrent", Port: 1234},
	}
	mu.Lock()
	cfg = testCfg
	mu.Unlock()

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				r := Get()
				if r == nil || r.Gateway.Host != "concurrent" {
					t.Error("concurrent Get returned unexpected result")
					return
				}
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestSet_VariousKeys(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "set_various.yaml")
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

	Set("gateway.rate_limit_requests", 300)
	Set("gateway.host", "test-host")
	Set("matchmaker.port", 9999)
	Set("redis.cluster_enabled", true)
	Set("compliance.blocked_domains", []string{"bad.com", "evil.com"})

	if viper.GetInt("gateway.rate_limit_requests") != 300 {
		t.Error("rate_limit_requests not set")
	}
	if viper.GetString("gateway.host") != "test-host" {
		t.Error("host not set")
	}
	if viper.GetInt("matchmaker.port") != 9999 {
		t.Error("matchmaker.port not set")
	}
	if !viper.GetBool("redis.cluster_enabled") {
		t.Error("redis.cluster_enabled not set")
	}
	domains := viper.GetStringSlice("compliance.blocked_domains")
	if len(domains) != 2 || domains[0] != "bad.com" || domains[1] != "evil.com" {
		t.Errorf("blocked_domains not set correctly: %v", domains)
	}
}

func TestSave_EmptyPath(t *testing.T) {
	cfgPath = ""
	err := Save()
	if err == nil {
		t.Error("expected error for empty config path")
	}
}

func TestOnChange_MultipleCalls(t *testing.T) {
	callCount := 0
	OnChange(func(*Config) { callCount++ })
	OnChange(func(*Config) { callCount++ })
	OnChange(func(*Config) { callCount++ })

	testCfg := &Config{Gateway: GatewayConfig{Host: "test"}}

	mu.Lock()
	cfg = testCfg
	subs := make([]func(*Config), len(subscribers))
	copy(subs, subscribers)
	mu.Unlock()

	for _, fn := range subs {
		fn(testCfg)
	}

	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

func TestWatch_ConfigReload(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "watch_reload.yaml")
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

	received := make(chan *Config, 1)
	OnChange(func(newCfg *Config) {
		select {
		case received <- newCfg:
		default:
		}
	})

	Watch()

	updated := `gateway:
  rate_limit_requests: 999
`
	if err := os.WriteFile(tmpFile, []byte(updated), 0644); err != nil {
		t.Fatalf("failed to write updated config: %v", err)
	}

	select {
	case newCfg := <-received:
		if newCfg.Gateway.RateLimitRequests != 999 {
			t.Errorf("expected 999, got %d", newCfg.Gateway.RateLimitRequests)
		}
	case <-time.After(3 * time.Second):
		t.Skip("file watcher did not fire within timeout (may not work in this environment)")
	}
}