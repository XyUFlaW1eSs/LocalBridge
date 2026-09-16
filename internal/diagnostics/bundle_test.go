package diagnostics

import (
	"archive/zip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/XyUFlaW1eSs/LocalBridge/internal/config"
)

func TestCreateWritesRedactedSupportBundle(t *testing.T) {
	dir := t.TempDir()
	registryPath := filepath.Join(dir, "private-device-registry.json")
	certPath := filepath.Join(dir, "private-certificate.pem")
	keyPath := filepath.Join(dir, "private-key.pem")
	secretState := "state-body-secret"
	for path, body := range map[string]string{registryPath: `{"peer_token":"peer-secret-value"}`, certPath: "CERTIFICATE BODY", keyPath: "PRIVATE KEY BODY"} {
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	syncPath := filepath.Join(dir, "private-sync-path.json")
	if err := os.WriteFile(syncPath, []byte(secretState), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.Device.ID = "private-device-id"
	cfg.Device.Name = "Private Device Name"
	cfg.Device.RegistryPath = registryPath
	cfg.Security.AuthEnabled = true
	cfg.Security.BearerToken = "management-secret-value"
	cfg.Security.PairingCode = "pairing-secret-value"
	cfg.Server.TLSEnabled = true
	cfg.Server.TLSCertFile = certPath
	cfg.Server.TLSKeyFile = keyPath
	cfg.Sync.StorePath = syncPath
	cfg.Files.StorePath = filepath.Join(dir, "private-files-path.json")
	cfg.Files.ShareDir = filepath.Join(dir, "private-share-directory")
	cfg.Files.ReceiveDir = filepath.Join(dir, "private-receive-directory")
	cfg.Settings.StorePath = filepath.Join(dir, "private-settings-path.json")

	destination := filepath.Join(dir, "support.zip")
	if err := Create(destination, cfg, "test-version"); err != nil {
		t.Fatal(err)
	}

	archive, err := zip.OpenReader(destination)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	if len(archive.File) != 2 {
		t.Fatalf("expected two bundle entries, got %d", len(archive.File))
	}
	diagnosticsJSON := readZipEntry(t, archive.File, "diagnostics.json")
	for _, privateValue := range []string{
		cfg.Device.ID, cfg.Device.Name, cfg.Device.RegistryPath, cfg.Server.TLSCertFile, cfg.Server.TLSKeyFile,
		cfg.Security.BearerToken, cfg.Security.PairingCode, cfg.Sync.StorePath, cfg.Files.StorePath,
		cfg.Files.ShareDir, cfg.Files.ReceiveDir, cfg.Settings.StorePath, "peer-secret-value", secretState,
		"CERTIFICATE BODY", "PRIVATE KEY BODY",
	} {
		if strings.Contains(string(diagnosticsJSON), privateValue) {
			t.Fatalf("bundle exposed private value %q", privateValue)
		}
	}
	var report Report
	if err := json.Unmarshal(diagnosticsJSON, &report); err != nil {
		t.Fatal(err)
	}
	if report.Version != reportVersion || report.Application.Version != "test-version" {
		t.Fatalf("unexpected report identity: %#v", report)
	}
	if !report.Configuration.Server.TLSEnabled || !report.Configuration.Server.TLSCertificateConfigured || !report.Configuration.Server.TLSKeyConfigured {
		t.Fatalf("TLS configuration presence was not preserved: %#v", report.Configuration.Server)
	}
	if !report.Configuration.Security.AuthenticationEnabled || !report.Configuration.Security.ManagementTokenConfigured || !report.Configuration.Security.PairingCodeConfigured {
		t.Fatalf("security configuration presence was not preserved: %#v", report.Configuration.Security)
	}
	if got := stateFile(report, "device_registry"); got.Status != "present" || got.SizeBytes == 0 || got.ModifiedAt == nil {
		t.Fatalf("unexpected registry metadata: %#v", got)
	}
	if got := stateFile(report, "sync_jobs"); got.Status != "present" || got.SizeBytes == 0 {
		t.Fatalf("unexpected sync metadata: %#v", got)
	}
	readme := string(readZipEntry(t, archive.File, "README.txt"))
	if !strings.Contains(readme, "does not contain tokens") || !strings.Contains(readme, "TLS certificate") {
		t.Fatalf("privacy notice missing from README: %q", readme)
	}
}

func TestCreateDoesNotOverwriteExistingFile(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "support.zip")
	if err := os.WriteFile(destination, []byte("keep-me"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Create(destination, config.Default(), "test"); err == nil {
		t.Fatal("expected existing destination to be rejected")
	}
	data, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "keep-me" {
		t.Fatalf("existing destination was changed: %q", data)
	}
}

func TestBuildReportClassifiesStateFilesWithoutPaths(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default()
	cfg.Device.RegistryPath = ""
	cfg.Sync.StorePath = dir
	cfg.Files.StorePath = filepath.Join(dir, "missing.json")
	cfg.Settings.StorePath = filepath.Join(dir, "settings.json")
	if err := os.WriteFile(cfg.Settings.StorePath, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	report := buildReport(cfg, "test", time.Unix(123, 0).UTC())
	if got := stateFile(report, "device_registry").Status; got != "not_configured" {
		t.Fatalf("expected not_configured, got %q", got)
	}
	if got := stateFile(report, "sync_jobs").Status; got != "not_regular" {
		t.Fatalf("expected not_regular, got %q", got)
	}
	if got := stateFile(report, "file_transfers").Status; got != "missing" {
		t.Fatalf("expected missing, got %q", got)
	}
	if got := stateFile(report, "settings").Status; got != "present" {
		t.Fatalf("expected present, got %q", got)
	}
}

func readZipEntry(t *testing.T, files []*zip.File, name string) []byte {
	t.Helper()
	for _, file := range files {
		if file.Name != name {
			continue
		}
		reader, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(reader)
		_ = reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	t.Fatalf("bundle entry %q not found", name)
	return nil
}

func stateFile(report Report, kind string) StateFileReport {
	for _, file := range report.StateFiles {
		if file.Kind == kind {
			return file
		}
	}
	return StateFileReport{}
}
