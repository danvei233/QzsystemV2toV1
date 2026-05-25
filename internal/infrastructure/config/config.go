package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Admin    AdminConfig    `yaml:"admin"`
	Database DatabaseConfig `yaml:"database"`
	Logs     LogsConfig     `yaml:"logs"`
	Gateway  GatewayConfig  `yaml:"gateway"`
}

func Save(path string, cfg *Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(path) == "" {
		path = "config.yaml"
	}
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	b, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}

type ServerConfig struct {
	Addr           string        `yaml:"addr"`
	ReadTimeout    time.Duration `yaml:"read_timeout"`
	WriteTimeout   time.Duration `yaml:"write_timeout"`
	TrustedProxies []string      `yaml:"trusted_proxies"`
}

type AdminConfig struct {
	Username   string        `yaml:"username"`
	Password   string        `yaml:"password"`
	SessionTTL time.Duration `yaml:"session_ttl"`
}

type DatabaseConfig struct {
	DSN string `yaml:"dsn"`
}

type LogsConfig struct {
	Request LogLimitConfig `yaml:"request"`
}

type LogLimitConfig struct {
	RetentionDays int   `yaml:"retention_days"`
	MaxSizeMB     int64 `yaml:"max_size_mb"`
}

type GatewayConfig struct {
	UpstreamBaseURL string        `yaml:"upstream_base_url"`
	UpstreamAPIKey  string        `yaml:"upstream_api_key"`
	Timeout         time.Duration `yaml:"timeout"`
	CacheTTL        time.Duration `yaml:"cache_ttl"`
	MaxBodyBytes    int64         `yaml:"max_body_bytes"`
	RequireAPIKey   bool          `yaml:"require_api_key"`
	AcceptedAPIKeys []string      `yaml:"accepted_api_keys"`
	RedactFields    []string      `yaml:"redact_fields"`
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	applyDefaults(&cfg)
	return &cfg, cfg.Validate()
}

func applyDefaults(cfg *Config) {
	if strings.TrimSpace(cfg.Server.Addr) == "" {
		cfg.Server.Addr = ":8090"
	}
	if cfg.Server.ReadTimeout <= 0 {
		cfg.Server.ReadTimeout = 15 * time.Second
	}
	if cfg.Server.WriteTimeout <= 0 {
		cfg.Server.WriteTimeout = 60 * time.Second
	}
	if strings.TrimSpace(cfg.Database.DSN) == "" {
		cfg.Database.DSN = "data/request_logs.db"
	}
	if cfg.Logs.Request.RetentionDays <= 0 {
		cfg.Logs.Request.RetentionDays = 7
	}
	if cfg.Logs.Request.MaxSizeMB <= 0 {
		cfg.Logs.Request.MaxSizeMB = 100
	}
	if cfg.Admin.SessionTTL <= 0 {
		cfg.Admin.SessionTTL = 12 * time.Hour
	}
	if cfg.Gateway.Timeout <= 0 {
		cfg.Gateway.Timeout = 30 * time.Second
	}
	if cfg.Gateway.CacheTTL <= 0 {
		cfg.Gateway.CacheTTL = 30 * time.Second
	}
	if cfg.Gateway.MaxBodyBytes <= 0 {
		cfg.Gateway.MaxBodyBytes = 1 << 20
	}
	if len(cfg.Gateway.RedactFields) == 0 {
		cfg.Gateway.RedactFields = []string{"apikey", "api_key", "signature", "authorization", "password", "sys_pwd", "panel_password", "token"}
	}
}

func (c *Config) Validate() error {
	if strings.TrimSpace(c.Admin.Username) == "" || strings.TrimSpace(c.Admin.Password) == "" {
		return errors.New("admin username/password required")
	}
	if strings.TrimSpace(c.Gateway.UpstreamBaseURL) == "" {
		return errors.New("gateway upstream_base_url required")
	}
	if strings.TrimSpace(c.Gateway.UpstreamAPIKey) == "" {
		return errors.New("gateway upstream_api_key required")
	}
	if c.Gateway.RequireAPIKey && len(c.Gateway.AcceptedAPIKeys) == 0 {
		return errors.New("accepted_api_keys required when require_api_key is enabled")
	}
	if c.Logs.Request.RetentionDays <= 0 {
		return errors.New("request log retention_days must be greater than 0")
	}
	if c.Logs.Request.MaxSizeMB <= 0 {
		return errors.New("request log max_size_mb must be greater than 0")
	}
	return nil
}
