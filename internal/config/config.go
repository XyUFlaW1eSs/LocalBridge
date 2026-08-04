package config

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Server    ServerConfig    `yaml:"server" json:"server"`
	Device    DeviceConfig    `yaml:"device" json:"device"`
	Security  SecurityConfig  `yaml:"security" json:"security"`
	Clipboard ClipboardConfig `yaml:"clipboard" json:"clipboard"`
	Logging   LoggingConfig   `yaml:"logging" json:"logging"`
}

type ServerConfig struct {
	Host         string        `yaml:"host" json:"host"`
	Port         int           `yaml:"port" json:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout" json:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout" json:"idle_timeout"`
}

type DeviceConfig struct {
	ID           string `yaml:"id" json:"id"`
	Name         string `yaml:"name" json:"name"`
	RegistryPath string `yaml:"registry_path" json:"registry_path"`
}

type SecurityConfig struct {
	AuthEnabled bool   `yaml:"auth_enabled" json:"auth_enabled"`
	BearerToken string `yaml:"bearer_token" json:"bearer_token"`
	PairingCode string `yaml:"pairing_code" json:"pairing_code"`
}

type ClipboardConfig struct {
	Enabled       bool          `yaml:"enabled" json:"enabled"`
	MaxTextBytes  int           `yaml:"max_text_bytes" json:"max_text_bytes"`
	WatchInterval time.Duration `yaml:"watch_interval" json:"watch_interval"`
}

type LoggingConfig struct {
	Level  string `yaml:"level" json:"level"`
	Format string `yaml:"format" json:"format"`
}

func Default() Config {
	return Config{
		Server:    ServerConfig{Host: "0.0.0.0", Port: 8899, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second},
		Device:    DeviceConfig{ID: "windows-pc", Name: "LocalBridge Windows", RegistryPath: "data/devices.json"},
		Security:  SecurityConfig{},
		Clipboard: ClipboardConfig{Enabled: true, MaxTextBytes: 1024 * 1024, WatchInterval: 300 * time.Millisecond},
		Logging:   LoggingConfig{Level: "info", Format: "text"},
	}
}

func Load(path string) (Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %q: %w", path, err)
	}
	if err := decode(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %q: %w", path, err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func LoadOrDefault(path string) (Config, error) {
	cfg, err := Load(path)
	if err == nil {
		return cfg, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		cfg = Default()
		return cfg, cfg.Validate()
	}
	return Config{}, err
}

// decode accepts JSON and the small, indentation-based YAML subset used by the
// checked-in configuration examples. Keeping this parser intentionally narrow
// avoids making configuration a runtime dependency of the core service.
func decode(data []byte, cfg *Config) error {
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, "{") {
		return json.Unmarshal(data, cfg)
	}
	section := ""
	scanner := bufio.NewScanner(strings.NewReader(trimmed))
	for lineNo := 1; scanner.Scan(); lineNo++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasSuffix(line, ":") {
			section = strings.TrimSpace(strings.TrimSuffix(line, ":"))
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 || section == "" {
			return fmt.Errorf("line %d: expected section or key: value", lineNo)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"")
		if err := setValue(cfg, section, key, value); err != nil {
			return fmt.Errorf("line %d: %w", lineNo, err)
		}
	}
	return scanner.Err()
}

func setValue(cfg *Config, section, key, value string) error {
	switch section + "." + key {
	case "server.host":
		cfg.Server.Host = value
	case "server.port":
		v, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid server.port: %w", err)
		}
		cfg.Server.Port = v
	case "server.read_timeout":
		v, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("invalid server.read_timeout: %w", err)
		}
		cfg.Server.ReadTimeout = v
	case "server.write_timeout":
		v, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("invalid server.write_timeout: %w", err)
		}
		cfg.Server.WriteTimeout = v
	case "server.idle_timeout":
		v, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("invalid server.idle_timeout: %w", err)
		}
		cfg.Server.IdleTimeout = v
	case "device.id":
		cfg.Device.ID = value
	case "device.name":
		cfg.Device.Name = value
	case "device.registry_path":
		cfg.Device.RegistryPath = value
	case "security.auth_enabled":
		v, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid security.auth_enabled: %w", err)
		}
		cfg.Security.AuthEnabled = v
	case "security.bearer_token":
		cfg.Security.BearerToken = value
	case "security.pairing_code":
		cfg.Security.PairingCode = value
	case "clipboard.enabled":
		v, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid clipboard.enabled: %w", err)
		}
		cfg.Clipboard.Enabled = v
	case "clipboard.max_text_bytes":
		v, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid clipboard.max_text_bytes: %w", err)
		}
		cfg.Clipboard.MaxTextBytes = v
	case "clipboard.watch_interval":
		v, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("invalid clipboard.watch_interval: %w", err)
		}
		cfg.Clipboard.WatchInterval = v
	case "logging.level":
		cfg.Logging.Level = value
	case "logging.format":
		cfg.Logging.Format = value
	default:
		return fmt.Errorf("unknown configuration key %s.%s", section, key)
	}
	return nil
}

func (c Config) Validate() error {
	if c.Server.Host == "" {
		return errors.New("server.host must not be empty")
	}
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535: %d", c.Server.Port)
	}
	if c.Device.ID == "" {
		return errors.New("device.id must not be empty")
	}
	if c.Device.RegistryPath == "" {
		return errors.New("device.registry_path must not be empty")
	}
	if c.Security.AuthEnabled && len(c.Security.BearerToken) < 16 {
		return errors.New("security.bearer_token must contain at least 16 characters when authentication is enabled")
	}
	if c.Clipboard.MaxTextBytes < 1 {
		return errors.New("clipboard.max_text_bytes must be positive")
	}
	if c.Clipboard.WatchInterval <= 0 {
		return errors.New("clipboard.watch_interval must be positive")
	}
	return nil
}

func (c ServerConfig) Address() string { return fmt.Sprintf("%s:%d", c.Host, c.Port) }
