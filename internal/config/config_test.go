package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDecodeJSON(t *testing.T) {
	cfg := Default()
	if err := decode([]byte(`{"version":1,"server":{"host":"127.0.0.1","port":9000,"read_timeout":1000000000,"write_timeout":1000000000,"idle_timeout":1000000000},"device":{"id":"json-pc","name":"JSON"},"security":{"auth_enabled":true,"bearer_token":"0123456789abcdef"},"clipboard":{"enabled":true,"max_text_bytes":99,"watch_interval":100000000},"logging":{"level":"debug","format":"json"}}`), &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Version != CurrentVersion || cfg.Device.ID != "json-pc" || cfg.Server.Port != 9000 || cfg.Clipboard.MaxTextBytes != 99 || !cfg.Security.AuthEnabled {
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
  peer_token_ttl: 48h
  token_overlap_ttl: 2m
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
	if cfg.Server.Port != 9999 || cfg.Device.ID != "test-pc" || cfg.Clipboard.MaxTextBytes != 42 || cfg.Logging.Format != "json" || !cfg.Security.AuthEnabled || cfg.Security.PeerTokenTTL != 48*time.Hour || cfg.Security.TokenOverlapTTL != 2*time.Minute || !cfg.Discovery.Enabled || cfg.Discovery.Port != 8898 {
		t.Fatalf("unexpected config: %#v", cfg)
	}
	diagnostics := cfg.Diagnostics()
	if cfg.Version != CurrentVersion || !diagnostics.Migrated || diagnostics.SourceSchemaVersion != 0 || diagnostics.Source != "file" {
		t.Fatalf("legacy config migration metadata is incorrect: %#v", diagnostics)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(content) {
		t.Fatal("legacy migration rewrote the source configuration")
	}
}

func TestLoadVersionedConfiguration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("version: 1\n\ndevice:\n  id: versioned-pc\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	diagnostics := cfg.Diagnostics()
	if cfg.Version != CurrentVersion || diagnostics.Migrated || diagnostics.SourceSchemaVersion != CurrentVersion || diagnostics.Source != "file" {
		t.Fatalf("versioned config metadata is incorrect: %#v", diagnostics)
	}
}

func TestLoadVersionedAndLegacyJSON(t *testing.T) {
	tests := []struct {
		name          string
		body          string
		sourceVersion int
		migrated      bool
	}{
		{name: "versioned", body: `{"version":1,"device":{"id":"json-v1"}}`, sourceVersion: 1},
		{name: "legacy", body: `{"version":0,"device":{"id":"json-v0"}}`, sourceVersion: 0, migrated: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(path, []byte(test.body), 0600); err != nil {
				t.Fatal(err)
			}
			cfg, err := Load(path)
			if err != nil {
				t.Fatal(err)
			}
			diagnostics := cfg.Diagnostics()
			if cfg.Version != CurrentVersion || diagnostics.SourceSchemaVersion != test.sourceVersion || diagnostics.Migrated != test.migrated {
				t.Fatalf("unexpected JSON migration metadata: %#v", diagnostics)
			}
		})
	}
}

func TestLoadOrDefaultReportsDefaultsSource(t *testing.T) {
	cfg, err := LoadOrDefault(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	diagnostics := cfg.Diagnostics()
	if diagnostics.Source != "defaults" || diagnostics.Migrated || diagnostics.SourceSchemaVersion != CurrentVersion {
		t.Fatalf("unexpected default source metadata: %#v", diagnostics)
	}
}

func TestRejectsUnsupportedConfigurationVersions(t *testing.T) {
	for _, body := range []string{"version: -1\n", "version: 2\n", `{"version":2}`} {
		path := filepath.Join(t.TempDir(), "config")
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil {
			t.Fatalf("unsupported version was accepted: %q", body)
		}
	}
}

func TestRejectsUnknownConfigurationKeys(t *testing.T) {
	for _, body := range []string{"version: 1\nunknown:\n", "version:\n", `{"version":1,"unknown":true}`} {
		path := filepath.Join(t.TempDir(), "config")
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil {
			t.Fatalf("unknown key was accepted: %q", body)
		}
	}
}

func TestDiagnosticsRedactsCredentials(t *testing.T) {
	cfg := Default()
	cfg.Security.AuthEnabled = true
	cfg.Security.BearerToken = "management-secret-value"
	cfg.Security.PairingCode = "pairing-secret-value"
	data, err := json.Marshal(cfg.Diagnostics())
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if strings.Contains(text, cfg.Security.BearerToken) || strings.Contains(text, cfg.Security.PairingCode) {
		t.Fatalf("diagnostics leaked a credential: %s", text)
	}
	if !strings.Contains(text, `"bearer_token_configured":true`) || !strings.Contains(text, `"pairing_code_configured":true`) {
		t.Fatalf("diagnostics omitted credential state: %s", text)
	}
}

func TestRejectsPeerTokenOverlapAtOrBeyondTTL(t *testing.T) {
	cfg := Default()
	cfg.Security.PeerTokenTTL = time.Hour
	cfg.Security.TokenOverlapTTL = time.Hour
	if err := cfg.Validate(); err == nil {
		t.Fatal("token overlap equal to token TTL was accepted")
	}
}

func TestTLSConfigurationRequiresCompletePair(t *testing.T) {
	cfg := Default()
	cfg.Server.TLSEnabled = true
	if err := cfg.Validate(); err == nil {
		t.Fatal("TLS without certificate and key was accepted")
	}
	cfg.Server.TLSCertFile = "cert.pem"
	if err := cfg.Validate(); err == nil {
		t.Fatal("TLS without private key was accepted")
	}
	cfg.Server.TLSKeyFile = "key.pem"
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	cfg.Server.TLSEnabled = false
	if err := cfg.Validate(); err == nil {
		t.Fatal("TLS file paths were accepted while TLS was disabled")
	}
}

func TestTLSConfigurationYAMLAndDiagnostics(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte("version: 1\nserver:\n  tls_enabled: true\n  tls_cert_file: cert.pem\n  tls_key_file: key.pem\n")
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(cfg.Diagnostics())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"tls_enabled":true`) || !strings.Contains(string(data), `"tls_cert_configured":true`) || strings.Contains(string(data), "cert.pem") {
		t.Fatalf("TLS diagnostics are incorrect or leaked a path: %s", data)
	}
}
