package app

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"

	"github.com/XyUFlaW1eSs/LocalBridge/internal/config"
	"github.com/XyUFlaW1eSs/LocalBridge/internal/credentials"
)

type appTestProtector struct{}

func (appTestProtector) Name() string    { return "test-protector" }
func (appTestProtector) Available() bool { return true }
func (appTestProtector) Protect(purpose string, plaintext []byte) ([]byte, error) {
	return append(append([]byte(purpose), 0), plaintext...), nil
}
func (appTestProtector) Unprotect(purpose string, ciphertext []byte) ([]byte, error) {
	prefix := append([]byte(purpose), 0)
	if !bytes.HasPrefix(ciphertext, prefix) {
		return nil, errors.New("purpose mismatch")
	}
	return append([]byte(nil), ciphertext[len(prefix):]...), nil
}

func TestResolveConfigCredentials(t *testing.T) {
	cfg := config.Default()
	cfg.Security.CredentialProtection = "required"
	cfg.Security.CredentialStorePath = filepath.Join(t.TempDir(), "credentials.json")
	cfg.Security.AuthEnabled = true
	cfg.Security.BearerTokenRef = "management"
	cfg.Security.PairingCodeRef = "pairing"
	protector := appTestProtector{}
	store, err := credentials.NewStore(cfg.Security.CredentialStorePath, protector)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Set("management", []byte("0123456789abcdef")); err != nil {
		t.Fatal(err)
	}
	if err := store.Set("pairing", []byte("pair-me")); err != nil {
		t.Fatal(err)
	}
	resolved, protection, err := resolveConfigCredentials(cfg, protector)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Security.BearerToken != "0123456789abcdef" || resolved.Security.PairingCode != "pair-me" || !protection.Enabled {
		t.Fatalf("unexpected resolved credentials or protection: %#v %#v", resolved.Security, protection)
	}
}

func TestResolveConfigCredentialsFailsForMissingOrShortManagementToken(t *testing.T) {
	protector := appTestProtector{}
	for name, prepare := range map[string]func(config.Config) error{
		"missing": func(config.Config) error { return nil },
		"short": func(cfg config.Config) error {
			store, err := credentials.NewStore(cfg.Security.CredentialStorePath, protector)
			if err != nil {
				return err
			}
			return store.Set("management", []byte("too-short"))
		},
	} {
		t.Run(name, func(t *testing.T) {
			cfg := config.Default()
			cfg.Security.CredentialProtection = "required"
			cfg.Security.CredentialStorePath = filepath.Join(t.TempDir(), "credentials.json")
			cfg.Security.AuthEnabled = true
			cfg.Security.BearerTokenRef = "management"
			if err := prepare(cfg); err != nil {
				t.Fatal(err)
			}
			if _, _, err := resolveConfigCredentials(cfg, protector); err == nil {
				t.Fatalf("%s management token was accepted", name)
			}
		})
	}
}

func TestResolveProtectionUnavailableFailsUnlessDisabled(t *testing.T) {
	unavailable := unavailableAppProtector{}
	if _, err := credentials.ResolveProtection("auto", unavailable); err == nil {
		t.Fatal("auto protection accepted an unavailable platform protector")
	}
	protection, err := credentials.ResolveProtection("disabled", unavailable)
	if err != nil || protection.Enabled || protection.Effective != "disabled" {
		t.Fatalf("explicit disabled mode failed: %#v %v", protection, err)
	}
}

type unavailableAppProtector struct{}

func (unavailableAppProtector) Name() string    { return "unavailable" }
func (unavailableAppProtector) Available() bool { return false }
func (unavailableAppProtector) Protect(string, []byte) ([]byte, error) {
	return nil, credentials.ErrUnavailable
}
func (unavailableAppProtector) Unprotect(string, []byte) ([]byte, error) {
	return nil, credentials.ErrUnavailable
}
