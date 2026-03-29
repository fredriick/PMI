package config

import (
	"fmt"
	"log"
	"sync"

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
}

type GatewayConfig struct {
	Host                    string `mapstructure:"host"`
	Port                    int    `mapstructure:"port"`
	MTLSEnabled             bool   `mapstructure:"mtls_enabled"`
	CACertPath              string `mapstructure:"ca_cert_path"`
	ServerCertPath          string `mapstructure:"server_cert_path"`
	ServerKeyPath           string `mapstructure:"server_key_path"`
	CircuitBreakerThreshold int    `mapstructure:"circuit_breaker_threshold"`
	RateLimitRequests       int    `mapstructure:"rate_limit_requests"`
	RateLimitWindowSeconds  int    `mapstructure:"rate_limit_window_seconds"`
	RateLimitDistributed    bool   `mapstructure:"rate_limit_distributed"`
	TracingEnabled          bool   `mapstructure:"tracing_enabled"`
}

type MatchmakerConfig struct {
	Host               string `mapstructure:"host"`
	Port               int    `mapstructure:"port"`
	PoolSize           int    `mapstructure:"pool_size"`
	CooldownTTLMinutes int    `mapstructure:"cooldown_ttl_minutes"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
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
		}
	})

	return cfg, err
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
