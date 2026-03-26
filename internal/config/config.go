package config

import (
	"fmt"
	"sync"

	"github.com/spf13/viper"
)

type Config struct {
	Gateway    GatewayConfig    `mapstructure:"gateway"`
	Matchmaker MatchmakerConfig `mapstructure:"matchmaker"`
	Redis      RedisConfig      `mapstructure:"redis"`
	Compliance ComplianceConfig `mapstructure:"compliance"`
	Peer       PeerConfig       `mapstructure:"peer"`
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
}

type MatchmakerConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	PoolSize int    `mapstructure:"pool_size"`
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

var (
	cfg  *Config
	once sync.Once
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
	return cfg
}
