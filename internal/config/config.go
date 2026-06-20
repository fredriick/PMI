package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

type Config struct {
	Gateway    GatewayConfig    `mapstructure:"gateway"`
	Matchmaker MatchmakerConfig `mapstructure:"matchmaker"`
	Redis      RedisConfig      `mapstructure:"redis"`
	Compliance ComplianceConfig `mapstructure:"compliance"`
	Peer       PeerConfig       `mapstructure:"peer"`
	Subnet     SubnetConfig     `mapstructure:"subnet"`
	Pricing    PricingConfig    `mapstructure:"pricing"`
	Federation FederationConfig `mapstructure:"federation"`
	JWT        JWTConfig        `mapstructure:"jwt"`
	RBAC       RBACConfig       `mapstructure:"rbac"`
	Prometheus PrometheusConfig `mapstructure:"prometheus"`
}

type GatewayConfig struct {
	Host                     string `mapstructure:"host"`
	Port                     int    `mapstructure:"port"`
	MTLSEnabled              bool   `mapstructure:"mtls_enabled"`
	CACertPath               string `mapstructure:"ca_cert_path"`
	ServerCertPath           string `mapstructure:"server_cert_path"`
	ServerKeyPath            string `mapstructure:"server_key_path"`
	CircuitBreakerThreshold  int    `mapstructure:"circuit_breaker_threshold"`
	RateLimitRequests        int    `mapstructure:"rate_limit_requests"`
	RateLimitWindowSeconds   int    `mapstructure:"rate_limit_window_seconds"`
	RateLimitDistributed     bool   `mapstructure:"rate_limit_distributed"`
	TracingEnabled           bool   `mapstructure:"tracing_enabled"`
	CooldownSeconds          int    `mapstructure:"cooldown_seconds"`
	RequestTimeoutSeconds    int    `mapstructure:"request_timeout_seconds"`
	IdleTimeoutSeconds       int    `mapstructure:"idle_timeout_seconds"`
	ReadHeaderTimeoutSeconds int    `mapstructure:"read_header_timeout_seconds"`
}

type MatchmakerConfig struct {
	Host               string `mapstructure:"host"`
	Port               int    `mapstructure:"port"`
	PoolSize           int    `mapstructure:"pool_size"`
	CooldownTTLMinutes int    `mapstructure:"cooldown_ttl_minutes"`
}

type RedisConfig struct {
	Host           string   `mapstructure:"host"`
	Port           int      `mapstructure:"port"`
	Password       string   `mapstructure:"password"`
	DB             int      `mapstructure:"db"`
	ClusterEnabled bool     `mapstructure:"cluster_enabled"`
	ClusterAddrs   []string `mapstructure:"cluster_addrs"`
	PoolSize       int      `mapstructure:"pool_size"`
	MaxRetries     int      `mapstructure:"max_retries"`
}

type ComplianceConfig struct {
	BlockedDomains []string `mapstructure:"blocked_domains"`
	KYCRequired    bool     `mapstructure:"kyc_required"`
}

type PeerConfig struct {
	MinBatteryPercent    int  `mapstructure:"min_battery_percent"`
	MaxCPUPercent        int  `mapstructure:"max_cpu_percent"`
	RequireUnmeteredWiFi bool `mapstructure:"require_unmetered_wifi"`
}

type SubnetConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	Prefix    string `mapstructure:"prefix"`
	PrefixLen int    `mapstructure:"prefix_len"`
}

type PricingConfig struct {
	Tiers []PricingTier `mapstructure:"tiers"`
}

type FederationConfig struct {
	Enabled           bool     `mapstructure:"enabled"`
	Regions           []string `mapstructure:"regions"`
	HeartbeatInterval int      `mapstructure:"heartbeat_interval_seconds"`
	SyncInterval      int      `mapstructure:"sync_interval_seconds"`
}

type PricingTier struct {
	Name          string  `mapstructure:"name"`
	MinGBMonthly  int     `mapstructure:"min_gb_monthly"`
	MaxGBMonthly  int     `mapstructure:"max_gb_monthly"`
	RatePerGBSent float64 `mapstructure:"rate_per_gb_sent"`
	RatePerGBRecv float64 `mapstructure:"rate_per_gb_recv"`
}

type JWTConfig struct {
	Enabled    bool          `mapstructure:"enabled"`
	SecretKey  string        `mapstructure:"secret_key"`
	Expiration time.Duration `mapstructure:"expiration"`
}

type RBACConfig struct {
	Enabled          bool   `mapstructure:"enabled"`
	DefaultRole      string `mapstructure:"default_role"`
	MinPasswordLen   int    `mapstructure:"min_password_len"`
	MaxLoginAttempts int    `mapstructure:"max_login_attempts"`
	LockoutDuration  int    `mapstructure:"lockout_duration_minutes"`
}

type PrometheusConfig struct {
	Enabled             bool   `mapstructure:"enabled"`
	PushGatewayURL      string `mapstructure:"push_gateway_url"`
	PushIntervalSeconds int    `mapstructure:"push_interval_seconds"`
	JobName             string `mapstructure:"job_name"`
}

var (
	cfg         *Config
	once        sync.Once
	mu          sync.RWMutex
	subscribers []func(*Config)
)

func Load(path string) (*Config, error) {
	var err error
	once.Do(func() {
		viper.SetConfigFile(path)
		viper.SetConfigType("yaml")

		err = viper.ReadInConfig()
		if err != nil {
			err = fmt.Errorf("failed to read config: %w", err)
			return
		}

		cfg = &Config{}
		err = viper.Unmarshal(cfg)
		if err != nil {
			err = fmt.Errorf("failed to unmarshal config: %w", err)
			return
		}

		applyEnvOverrides(cfg)
	})

	return cfg, err
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		cfg.Redis.Password = v
	}

	if v := os.Getenv("MTLS_ENABLED"); v != "" {
		if enabled, err := strconv.ParseBool(v); err == nil {
			cfg.Gateway.MTLSEnabled = enabled
		}
	}
	if v := os.Getenv("MTLS_CA_CERT_PATH"); v != "" {
		cfg.Gateway.CACertPath = v
	}
	if v := os.Getenv("MTLS_SERVER_CERT_PATH"); v != "" {
		cfg.Gateway.ServerCertPath = v
	}
	if v := os.Getenv("MTLS_SERVER_KEY_PATH"); v != "" {
		cfg.Gateway.ServerKeyPath = v
	}

	if v := os.Getenv("JWT_ENABLED"); v != "" {
		if enabled, err := strconv.ParseBool(v); err == nil {
			cfg.JWT.Enabled = enabled
		}
	}
	if v := os.Getenv("JWT_SECRET_KEY"); v != "" {
		cfg.JWT.SecretKey = v
	}

	if v := os.Getenv("RBAC_ENABLED"); v != "" {
		if enabled, err := strconv.ParseBool(v); err == nil {
			cfg.RBAC.Enabled = enabled
		}
	}
}

func Get() *Config {
	mu.RLock()
	defer mu.RUnlock()
	return cfg
}

// OnChange registers a callback that fires when config changes.
func OnChange(fn func(*Config)) {
	mu.Lock()
	defer mu.Unlock()
	subscribers = append(subscribers, fn)
}

// Watch starts watching the config file for changes and reloads automatically.
func Watch() {
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Printf("Config file changed: %s", e.Name)

		newCfg := &Config{}
		if err := viper.Unmarshal(newCfg); err != nil {
			log.Printf("Failed to reload config: %v", err)
			return
		}

		mu.Lock()
		cfg = newCfg
		subs := make([]func(*Config), len(subscribers))
		copy(subs, subscribers)
		mu.Unlock()

		for _, fn := range subs {
			fn(newCfg)
		}

		log.Println("Config reloaded successfully")
	})
}
