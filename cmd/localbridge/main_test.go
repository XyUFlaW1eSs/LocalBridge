package main

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/XyUFlaW1eSs/LocalBridge/internal/config"
	"github.com/XyUFlaW1eSs/LocalBridge/internal/credentials"
)

type cliTestProtector struct{}

func (cliTestProtector) Name() string    { return "test-protector" }
func (cliTestProtector) Available() bool { return true }
func (cliTestProtector) Protect(purpose string, plaintext []byte) ([]byte, error) {
	return append(append([]byte(purpose), 0), plaintext...), nil
}
func (cliTestProtector) Unprotect(purpose string, ciphertext []byte) ([]byte, error) {
	prefix := append([]byte(purpose), 0)
	if !bytes.HasPrefix(ciphertext, prefix) {
		return nil, errors.New("purpose mismatch")
	}
	return append([]byte(nil), ciphertext[len(prefix):]...), nil
}

func TestCredentialCLISetStatusDeleteWithoutSecretOutput(t *testing.T) {
	cfg := config.Default()
	cfg.Security.CredentialProtection = "required"
	cfg.Security.CredentialStorePath = filepath.Join(t.TempDir(), "credentials.json")
	protector := cliTestProtector{}
	secret := "cli-management-secret"
	var output bytes.Buffer
	code, err := runCredentialOperation(cfg, "set", "management", protector, func() ([]byte, error) { return []byte(secret), nil }, &output)
	if err != nil || code != 0 {
		t.Fatalf("set code=%d err=%v", code, err)
	}
	if strings.Contains(output.String(), secret) {
		t.Fatal("set output exposed the secret")
	}
	data := output.String()
	output.Reset()
	code, err = runCredentialOperation(cfg, "status", "management", protector, nil, &output)
	if err != nil || code != 0 || !strings.Contains(output.String(), "present") || strings.Contains(output.String(), secret) {
		t.Fatalf("status code=%d err=%v output=%q", code, err, output.String())
	}
	output.Reset()
	code, err = runCredentialOperation(cfg, "delete", "management", protector, nil, &output)
	if err != nil || code != 0 || !strings.Contains(output.String(), "deleted") {
		t.Fatalf("delete code=%d err=%v output=%q", code, err, output.String())
	}
	if strings.Contains(data+output.String(), secret) {
		t.Fatal("credential CLI output exposed the secret")
	}
}

func TestCredentialCLIExitCodesAndUnrelatedCredential(t *testing.T) {
	cfg := config.Default()
	cfg.Security.CredentialProtection = "required"
	cfg.Security.CredentialStorePath = filepath.Join(t.TempDir(), "credentials.json")
	protector := cliTestProtector{}
	store, _ := credentials.NewStore(cfg.Security.CredentialStorePath, protector)
	if err := store.Set("pairing", []byte("keep-this")); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if code, _ := runCredentialOperation(cfg, "unknown", "management", protector, nil, &output); code != 2 {
		t.Fatalf("invalid action exit code=%d", code)
	}
	if code, _ := runCredentialOperation(cfg, "status", "../bad", protector, nil, &output); code != 2 {
		t.Fatalf("invalid name exit code=%d", code)
	}
	if code, err := runCredentialOperation(cfg, "delete", "management", protector, nil, &output); code != 0 || err != nil {
		t.Fatalf("delete missing code=%d err=%v", code, err)
	}
	got, err := store.Get("pairing")
	if err != nil || string(got) != "keep-this" {
		t.Fatalf("unrelated credential changed: %q err=%v", got, err)
	}
}
