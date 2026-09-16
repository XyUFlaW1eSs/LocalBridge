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
	Discovery DiscoveryConfig `yaml:"discovery" json:"discovery"`
	Sync      SyncConfig      `yaml:"sync" json:"sync"`
	Clipboard ClipboardConfig `yaml:"clipboard" json:"clipboard"`
	Files     FilesConfig     `yaml:"files" json:"files"`
	Settings  SettingsConfig  `yaml:"settings" json:"settings"`
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
	ID             string        `yaml:"id" json:"id"`
	Name           string        `yaml:"name" json:"name"`
	RegistryPath   string        `yaml:"registry_path" json:"registry_path"`
	HealthInterval time.Duration `yaml:"health_interval" json:"health_interval"`
}

type SecurityConfig struct {
	AuthEnabled     bool          `yaml:"auth_enabled" json:"auth_enabled"`
	BearerToken     string        `yaml:"bearer_token" json:"bearer_token"`
	PairingCode     string        `yaml:"pairing_code" json:"pairing_code"`
	PeerTokenTTL    time.Duration `yaml:"peer_token_ttl" json:"peer_token_ttl"`
	TokenOverlapTTL time.Duration `yaml:"token_overlap_ttl" json:"token_overlap_ttl"`
}

type DiscoveryConfig struct {
	Enabled          bool          `yaml:"enabled" json:"enabled"`
	Port             int           `yaml:"port" json:"port"`
	AnnounceInterval time.Duration `yaml:"announce_interval" json:"announce_interval"`
}

type SyncConfig struct {
	Enabled      bool          `yaml:"enabled" json:"enabled"`
	StorePath    string        `yaml:"store_path" json:"store_path"`
	MaxJobs      int           `yaml:"max_jobs" json:"max_jobs"`
	JobRetention time.Duration `yaml:"job_retention" json:"job_retention"`
}

type ClipboardConfig struct {
	Enabled       bool          `yaml:"enabled" json:"enabled"`
	MaxTextBytes  int           `yaml:"max_text_bytes" json:"max_text_bytes"`
	WatchInterval time.Duration `yaml:"watch_interval" json:"watch_interval"`
}

// FilesConfig controls the local file sharing and receiving boundary. Source
// files are never copied into StorePath; received files are written below
// ReceiveDir and are addressed by generated IDs, not client-supplied paths.
type FilesConfig struct {
	Enabled          bool          `yaml:"enabled" json:"enabled"`
	StorePath        string        `yaml:"store_path" json:"store_path"`
	ShareDir         string        `yaml:"share_dir" json:"share_dir"`
	ReceiveDir       string        `yaml:"receive_dir" json:"receive_dir"`
	MaxFileBytes     int64         `yaml:"max_file_bytes" json:"max_file_bytes"`
	MaxTotalBytes    int64         `yaml:"max_total_bytes" json:"max_total_bytes"`
	MaxFilesPerShare int           `yaml:"max_files_per_share" json:"max_files_per_share"`
	ShareTTL         time.Duration `yaml:"share_ttl" json:"share_ttl"`
	UploadTTL        time.Duration `yaml:"upload_ttl" json:"upload_ttl"`
}

type SettingsConfig struct {
	StorePath string `yaml:"store_path" json:"store_path"`
}

type LoggingConfig struct {
	Level  string `yaml:"level" json:"level"`
	Format string `yaml:"format" json:"format"`
}

func Default() Config {
	return Config{
		Server:    ServerConfig{Host: "0.0.0.0", Port: 8899, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second},
		Device:    DeviceConfig{ID: "windows-pc", Name: "LocalBridge Windows", RegistryPath: "data/devices.json", HealthInterval: 30 * time.Second},
		Security:  SecurityConfig{PeerTokenTTL: 30 * 24 * time.Hour, TokenOverlapTTL: 10 * time.Minute},
		Discovery: DiscoveryConfig{Port: 8898, AnnounceInterval: 10 * time.Second},
		Sync:      SyncConfig{Enabled: true, StorePath: "data/sync-jobs.json", MaxJobs: 1000, JobRetention: 7 * 24 * time.Hour},
		Clipboard: ClipboardConfig{Enabled: true, MaxTextBytes: 1024 * 1024, WatchInterval: 300 * time.Millisecond},
		Files:     FilesConfig{Enabled: true, StorePath: "data/files.json", ShareDir: "data/shared", ReceiveDir: "data/received", MaxFileBytes: 2 * 1024 * 1024 * 1024, MaxTotalBytes: 4 * 1024 * 1024 * 1024, MaxFilesPerShare: 100, ShareTTL: 24 * time.Hour, UploadTTL: 24 * time.Hour},
		Settings:  SettingsConfig{StorePath: "data/settings.json"},
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
	case "device.health_interval":
		v, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("invalid device.health_interval: %w", err)
		}
		cfg.Device.HealthInterval = v
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
	case "security.peer_token_ttl":
		v, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("invalid security.peer_token_ttl: %w", err)
		}
		cfg.Security.PeerTokenTTL = v
	case "security.token_overlap_ttl":
		v, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("invalid security.token_overlap_ttl: %w", err)
		}
		cfg.Security.TokenOverlapTTL = v
	case "discovery.enabled":
		v, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid discovery.enabled: %w", err)
		}
		cfg.Discovery.Enabled = v
	case "discovery.port":
		v, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid discovery.port: %w", err)
		}
		cfg.Discovery.Port = v
	case "discovery.announce_interval":
		v, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("invalid discovery.announce_interval: %w", err)
		}
		cfg.Discovery.AnnounceInterval = v
	case "sync.enabled":
		v, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid sync.enabled: %w", err)
		}
		cfg.Sync.Enabled = v
	case "sync.store_path":
		cfg.Sync.StorePath = value
	case "sync.max_jobs":
		v, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid sync.max_jobs: %w", err)
		}
		cfg.Sync.MaxJobs = v
	case "sync.job_retention":
		v, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("invalid sync.job_retention: %w", err)
		}
		cfg.Sync.JobRetention = v
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
	case "files.enabled":
		v, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid files.enabled: %w", err)
		}
		cfg.Files.Enabled = v
	case "files.store_path":
		cfg.Files.StorePath = value
	case "files.share_dir":
		cfg.Files.ShareDir = value
	case "files.receive_dir":
		cfg.Files.ReceiveDir = value
	case "files.max_file_bytes":
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid files.max_file_bytes: %w", err)
		}
		cfg.Files.MaxFileBytes = v
	case "files.max_total_bytes":
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid files.max_total_bytes: %w", err)
		}
		cfg.Files.MaxTotalBytes = v
	case "files.max_files_per_share":
		v, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid files.max_files_per_share: %w", err)
		}
		cfg.Files.MaxFilesPerShare = v
	case "files.share_ttl":
		v, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("invalid files.share_ttl: %w", err)
		}
		cfg.Files.ShareTTL = v
	case "files.upload_ttl":
		v, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("invalid files.upload_ttl: %w", err)
		}
		cfg.Files.UploadTTL = v
	case "settings.store_path":
		cfg.Settings.StorePath = value
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
	if c.Device.HealthInterval < 0 {
		return errors.New("device.health_interval must not be negative")
	}
	if c.Security.AuthEnabled && len(c.Security.BearerToken) < 16 {
		return errors.New("security.bearer_token must contain at least 16 characters when authentication is enabled")
	}
	if c.Security.PeerTokenTTL < 0 || c.Security.TokenOverlapTTL < 0 {
		return errors.New("security peer token durations must not be negative")
	}
	if c.Security.PeerTokenTTL > 0 && c.Security.TokenOverlapTTL >= c.Security.PeerTokenTTL {
		return errors.New("security.token_overlap_ttl must be shorter than peer_token_ttl")
	}
	if c.Discovery.Port < 1 || c.Discovery.Port > 65535 {
		return fmt.Errorf("discovery.port must be between 1 and 65535: %d", c.Discovery.Port)
	}
	if c.Discovery.AnnounceInterval <= 0 {
		return errors.New("discovery.announce_interval must be positive")
	}
	if c.Sync.StorePath == "" {
		return errors.New("sync.store_path must not be empty")
	}
	if c.Sync.MaxJobs < 1 {
		return errors.New("sync.max_jobs must be positive")
	}
	if c.Sync.JobRetention <= 0 {
		return errors.New("sync.job_retention must be positive")
	}
	if c.Clipboard.MaxTextBytes < 1 {
		return errors.New("clipboard.max_text_bytes must be positive")
	}
	if c.Clipboard.WatchInterval <= 0 {
		return errors.New("clipboard.watch_interval must be positive")
	}
	if c.Files.Enabled {
		if c.Files.StorePath == "" {
			return errors.New("files.store_path must not be empty")
		}
		if c.Files.ShareDir == "" {
			return errors.New("files.share_dir must not be empty")
		}
		if c.Files.ReceiveDir == "" {
			return errors.New("files.receive_dir must not be empty")
		}
		if c.Files.MaxFileBytes < 1 {
			return errors.New("files.max_file_bytes must be positive")
		}
		if c.Files.MaxTotalBytes < c.Files.MaxFileBytes {
			return errors.New("files.max_total_bytes must be at least max_file_bytes")
		}
		if c.Files.MaxFilesPerShare < 1 {
			return errors.New("files.max_files_per_share must be positive")
		}
		if c.Files.ShareTTL <= 0 || c.Files.UploadTTL <= 0 {
			return errors.New("files.share_ttl and files.upload_ttl must be positive")
		}
	}
	if c.Settings.StorePath == "" {
		return errors.New("settings.store_path must not be empty")
	}
	return nil
}

func (c ServerConfig) Address() string { return fmt.Sprintf("%s:%d", c.Host, c.Port) }
