// Package diagnostics creates content-safe support bundles for troubleshooting.
package diagnostics

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/XyUFlaW1eSs/LocalBridge/internal/config"
)

const reportVersion = 1

// Report is intentionally metadata-only. It excludes device identity, local
// paths, credentials, payload content, and persisted state-file bodies.
type Report struct {
	Version       int                 `json:"version"`
	GeneratedAt   time.Time           `json:"generated_at"`
	Application   ApplicationReport   `json:"application"`
	Runtime       RuntimeReport       `json:"runtime"`
	Configuration ConfigurationReport `json:"configuration"`
	StateFiles    []StateFileReport   `json:"state_files"`
}

type ApplicationReport struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type RuntimeReport struct {
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

type ConfigurationReport struct {
	Server    ServerReport    `json:"server"`
	Device    DeviceReport    `json:"device"`
	Security  SecurityReport  `json:"security"`
	Discovery DiscoveryReport `json:"discovery"`
	Sync      SyncReport      `json:"sync"`
	Clipboard ClipboardReport `json:"clipboard"`
	Files     FilesReport     `json:"files"`
	Logging   LoggingReport   `json:"logging"`
}

type ServerReport struct {
	Host                     string `json:"host"`
	Port                     int    `json:"port"`
	TLSEnabled               bool   `json:"tls_enabled"`
	TLSCertificateConfigured bool   `json:"tls_certificate_configured"`
	TLSKeyConfigured         bool   `json:"tls_key_configured"`
	ReadTimeout              string `json:"read_timeout"`
	WriteTimeout             string `json:"write_timeout"`
	IdleTimeout              string `json:"idle_timeout"`
}

type DeviceReport struct {
	HealthInterval string `json:"health_interval"`
}

type SecurityReport struct {
	AuthenticationEnabled     bool   `json:"authentication_enabled"`
	ManagementTokenConfigured bool   `json:"management_token_configured"`
	PairingCodeConfigured     bool   `json:"pairing_code_configured"`
	CredentialStoreConfigured bool   `json:"credential_store_configured"`
	ProtectionMode            string `json:"credential_protection_mode"`
	ProtectionEffective       string `json:"credential_protection_effective"`
	ProtectionSupported       bool   `json:"credential_protection_supported"`
}

type DiscoveryReport struct {
	Enabled          bool   `json:"enabled"`
	Port             int    `json:"port"`
	AnnounceInterval string `json:"announce_interval"`
}

type SyncReport struct {
	Enabled      bool   `json:"enabled"`
	MaxJobs      int    `json:"max_jobs"`
	JobRetention string `json:"job_retention"`
}

type ClipboardReport struct {
	Enabled       bool   `json:"enabled"`
	MaxTextBytes  int    `json:"max_text_bytes"`
	WatchInterval string `json:"watch_interval"`
}

type FilesReport struct {
	Enabled          bool   `json:"enabled"`
	MaxFileBytes     int64  `json:"max_file_bytes"`
	MaxTotalBytes    int64  `json:"max_total_bytes"`
	MaxFilesPerShare int    `json:"max_files_per_share"`
	ShareTTL         string `json:"share_ttl"`
	UploadTTL        string `json:"upload_ttl"`
}

type LoggingReport struct {
	Level  string `json:"level"`
	Format string `json:"format"`
}

type StateFileReport struct {
	Kind       string     `json:"kind"`
	Status     string     `json:"status"`
	SizeBytes  int64      `json:"size_bytes,omitempty"`
	ModifiedAt *time.Time `json:"modified_at,omitempty"`
}

// Create writes a new support bundle without overwriting an existing file.
// Only owner-readable permissions are requested where supported.
func Create(destination string, cfg config.Config, applicationVersion string) (err error) {
	destination = strings.TrimSpace(destination)
	if destination == "" {
		return errors.New("support bundle destination must not be empty")
	}

	file, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create support bundle: %w", err)
	}
	complete := false
	defer func() {
		if closeErr := file.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("close support bundle: %w", closeErr)
		}
		if !complete || err != nil {
			_ = os.Remove(destination)
		}
	}()

	generatedAt := time.Now().UTC()
	data, err := json.MarshalIndent(buildReport(cfg, applicationVersion, generatedAt), "", "  ")
	if err != nil {
		return fmt.Errorf("encode diagnostics report: %w", err)
	}
	data = append(data, '\n')

	archive := zip.NewWriter(file)
	if err := writeEntry(archive, "diagnostics.json", data, generatedAt); err != nil {
		_ = archive.Close()
		return err
	}
	readme := []byte("LocalBridge support bundle\n\nThis archive contains redacted configuration and state-file metadata only.\nIt does not contain tokens, pairing codes, device identity, local paths, TLS certificate or private-key paths/content, clipboard content, shared files, received files, or persisted state bodies.\n")
	if err := writeEntry(archive, "README.txt", readme, generatedAt); err != nil {
		_ = archive.Close()
		return err
	}
	if err := archive.Close(); err != nil {
		return fmt.Errorf("close support bundle archive: %w", err)
	}
	complete = true
	return nil
}

func writeEntry(archive *zip.Writer, name string, data []byte, modified time.Time) error {
	header := &zip.FileHeader{Name: name, Method: zip.Deflate, Modified: modified}
	header.SetMode(0o600)
	entry, err := archive.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("create support bundle entry %q: %w", name, err)
	}
	if _, err := entry.Write(data); err != nil {
		return fmt.Errorf("write support bundle entry %q: %w", name, err)
	}
	return nil
}

func buildReport(cfg config.Config, applicationVersion string, generatedAt time.Time) Report {
	return Report{
		Version:     reportVersion,
		GeneratedAt: generatedAt,
		Application: ApplicationReport{Name: "LocalBridge", Version: applicationVersion},
		Runtime:     RuntimeReport{GoVersion: runtime.Version(), OS: runtime.GOOS, Arch: runtime.GOARCH},
		Configuration: ConfigurationReport{
			Server: ServerReport{
				Host: cfg.Server.Host, Port: cfg.Server.Port, TLSEnabled: cfg.Server.TLSEnabled,
				TLSCertificateConfigured: strings.TrimSpace(cfg.Server.TLSCertFile) != "",
				TLSKeyConfigured:         strings.TrimSpace(cfg.Server.TLSKeyFile) != "",
				ReadTimeout:              cfg.Server.ReadTimeout.String(), WriteTimeout: cfg.Server.WriteTimeout.String(), IdleTimeout: cfg.Server.IdleTimeout.String(),
			},
			Device: DeviceReport{HealthInterval: cfg.Device.HealthInterval.String()},
			Security: SecurityReport{
				AuthenticationEnabled:     cfg.Security.AuthEnabled,
				ManagementTokenConfigured: strings.TrimSpace(cfg.Security.BearerToken) != "" || strings.TrimSpace(cfg.Security.BearerTokenRef) != "",
				PairingCodeConfigured:     strings.TrimSpace(cfg.Security.PairingCode) != "" || strings.TrimSpace(cfg.Security.PairingCodeRef) != "",
				CredentialStoreConfigured: strings.TrimSpace(cfg.Security.CredentialStorePath) != "",
				ProtectionMode:            protectionMode(cfg.Security.CredentialProtection),
				ProtectionEffective:       protectionEffective(cfg.Security.CredentialProtection),
				ProtectionSupported:       runtime.GOOS == "windows",
			},
			Discovery: DiscoveryReport{Enabled: cfg.Discovery.Enabled, Port: cfg.Discovery.Port, AnnounceInterval: cfg.Discovery.AnnounceInterval.String()},
			Sync:      SyncReport{Enabled: cfg.Sync.Enabled, MaxJobs: cfg.Sync.MaxJobs, JobRetention: cfg.Sync.JobRetention.String()},
			Clipboard: ClipboardReport{Enabled: cfg.Clipboard.Enabled, MaxTextBytes: cfg.Clipboard.MaxTextBytes, WatchInterval: cfg.Clipboard.WatchInterval.String()},
			Files:     FilesReport{Enabled: cfg.Files.Enabled, MaxFileBytes: cfg.Files.MaxFileBytes, MaxTotalBytes: cfg.Files.MaxTotalBytes, MaxFilesPerShare: cfg.Files.MaxFilesPerShare, ShareTTL: cfg.Files.ShareTTL.String(), UploadTTL: cfg.Files.UploadTTL.String()},
			Logging:   LoggingReport{Level: cfg.Logging.Level, Format: cfg.Logging.Format},
		},
		StateFiles: []StateFileReport{
			inspectStateFile("credential_store", cfg.Security.CredentialStorePath),
			inspectStateFile("device_registry", cfg.Device.RegistryPath),
			inspectStateFile("sync_jobs", cfg.Sync.StorePath),
			inspectStateFile("file_transfers", cfg.Files.StorePath),
			inspectStateFile("settings", cfg.Settings.StorePath),
		},
	}
}

func protectionMode(mode string) string {
	mode = strings.TrimSpace(mode)
	if mode == "" {
		return "auto"
	}
	return mode
}

func protectionEffective(mode string) string {
	if protectionMode(mode) == "disabled" {
		return "disabled"
	}
	if runtime.GOOS == "windows" {
		return "dpapi-current-user"
	}
	return "unsupported"
}

func inspectStateFile(kind, path string) StateFileReport {
	report := StateFileReport{Kind: kind}
	if strings.TrimSpace(path) == "" {
		report.Status = "not_configured"
		return report
	}
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		report.Status = "missing"
		return report
	}
	if err != nil {
		report.Status = "unavailable"
		return report
	}
	if !info.Mode().IsRegular() {
		report.Status = "not_regular"
		return report
	}
	modified := info.ModTime().UTC()
	report.Status = "present"
	report.SizeBytes = info.Size()
	report.ModifiedAt = &modified
	return report
}
