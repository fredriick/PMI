package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
