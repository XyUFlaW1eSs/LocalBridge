package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDecodeJSON(t *testing.T) {
	var cfg Config
	if err := decode([]byte(`{"server":{"host":"127.0.0.1","port":9000,"read_timeout":1000000000,"write_timeout":1000000000,"idle_timeout":1000000000},"device":{"id":"json-pc","name":"JSON"},"security":{"auth_enabled":true,"bearer_token":"0123456789abcdef"},"clipboard":{"enabled":true,"max_text_bytes":99,"watch_interval":100000000},"logging":{"level":"debug","format":"json"}}`), &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Device.ID != "json-pc" || cfg.Server.Port != 9000 || cfg.Clipboard.MaxTextBytes != 99 || !cfg.Security.AuthEnabled {
		t.Fatalf("unexpected JSON config: %#v", cfg)
	}
	if _, err := json.Marshal(cfg); err != nil {
		t.Fatal(err)
	}
}

func TestLoadYAMLSubset(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte(`server:
  host: "127.0.0.1"
  port: 9999
  read_timeout: 2s
  write_timeout: 3s
  idle_timeout: 4s
device:
  id: "test-pc"
  name: "Test"
security:
  auth_enabled: true
  bearer_token: "0123456789abcdef"
discovery:
  enabled: true
  port: 8898
  announce_interval: 2s
clipboard:
  enabled: true
  max_text_bytes: 42
  watch_interval: 1s
logging:
  level: "debug"
  format: "json"
`)
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Port != 9999 || cfg.Device.ID != "test-pc" || cfg.Clipboard.MaxTextBytes != 42 || cfg.Logging.Format != "json" || !cfg.Security.AuthEnabled || !cfg.Discovery.Enabled || cfg.Discovery.Port != 8898 {
		t.Fatalf("unexpected config: %#v", cfg)
	}
}
