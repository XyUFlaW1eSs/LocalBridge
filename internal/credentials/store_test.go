package credentials

import (
	"bytes"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type testProtector struct {
	failProtect bool
}

func (testProtector) Name() string    { return "test" }
func (testProtector) Available() bool { return true }
func (p testProtector) Protect(purpose string, plaintext []byte) ([]byte, error) {
	if p.failProtect {
		return nil, errors.New("injected protection failure")
	}
	return append(append([]byte(purpose), 0), plaintext...), nil
}
func (testProtector) Unprotect(purpose string, ciphertext []byte) ([]byte, error) {
	prefix := append([]byte(purpose), 0)
	if !bytes.HasPrefix(ciphertext, prefix) {
		return nil, errors.New("purpose mismatch or damaged ciphertext")
	}
	return append([]byte(nil), ciphertext[len(prefix):]...), nil
}

func TestStoreRoundTripDeleteAndPreserveUnrelated(t *testing.T) {
	path := filepath.Join(t.TempDir(), "credentials.json")
	store, err := NewStore(path, testProtector{})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Set("management", []byte("management-secret")); err != nil {
		t.Fatal(err)
	}
	if err := store.Set("pairing", []byte("pairing-secret")); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get("management")
	if err != nil || string(got) != "management-secret" {
		t.Fatalf("round trip got %q, err=%v", got, err)
	}
	deleted, err := store.Delete("management")
	if err != nil || !deleted {
		t.Fatalf("delete result=%v err=%v", deleted, err)
	}
	if _, err := store.Get("management"); err == nil {
		t.Fatal("deleted credential remained readable")
	}
	got, err = store.Get("pairing")
	if err != nil || string(got) != "pairing-secret" {
		t.Fatalf("unrelated credential changed: %q err=%v", got, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "pairing-secret") {
		t.Fatal("credential store persisted plaintext")
	}
}

func TestStoreRejectsPurposeMismatchAndDamagedCiphertext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "credentials.json")
	store, _ := NewStore(path, testProtector{})
	wrongPurpose := base64.StdEncoding.EncodeToString([]byte("localbridge/credential-store/v1/other\x00secret"))
	data := `{"version":1,"entries":[{"name":"management","ciphertext":"` + wrongPurpose + `"}]}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get("management"); err == nil {
		t.Fatal("purpose mismatch was accepted")
	}
	if err := os.WriteFile(path, []byte(`{"version":1,"entries":[{"name":"management","ciphertext":"%%%"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get("management"); err == nil {
		t.Fatal("damaged ciphertext was accepted")
	}
}

func TestStoreRejectsUnknownDuplicateVersionAndSize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "credentials.json")
	store, _ := NewStore(path, testProtector{})
	validCiphertext := base64.StdEncoding.EncodeToString([]byte("ciphertext"))
	cases := []string{
		`{"version":2,"entries":[]}`,
		`{"version":1,"unknown":true,"entries":[]}`,
		`{"version":1,"entries":[{"name":"one","ciphertext":"` + validCiphertext + `"},{"name":"one","ciphertext":"` + validCiphertext + `"}]}`,
	}
	for _, data := range cases {
		if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Has("one"); err == nil {
			t.Fatalf("invalid store was accepted: %s", data)
		}
	}
	if err := os.WriteFile(path, bytes.Repeat([]byte("x"), maxStoreBytes+1), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Has("one"); err == nil {
		t.Fatal("oversized store was accepted")
	}
}

func TestStoreProtectionFailureDoesNotOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "credentials.json")
	good, _ := NewStore(path, testProtector{})
	if err := good.Set("management", []byte("original-secret")); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	failing, _ := NewStore(path, testProtector{failProtect: true})
	if err := failing.Set("management", []byte("replacement-secret")); err == nil {
		t.Fatal("expected protection failure")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("store changed after protection failure")
	}
}

func TestStoreValidatesNamesAndSecretSize(t *testing.T) {
	store, _ := NewStore(filepath.Join(t.TempDir(), "credentials.json"), testProtector{})
	for _, name := range []string{"", "UPPER", "../escape", strings.Repeat("a", 65)} {
		if err := store.Set(name, []byte("value")); err == nil {
			t.Fatalf("invalid name %q was accepted", name)
		}
	}
	if err := store.Set("empty", nil); err == nil {
		t.Fatal("empty secret was accepted")
	}
	if err := store.Set("large", bytes.Repeat([]byte("x"), maxSecretSize+1)); err == nil {
		t.Fatal("oversized secret was accepted")
	}
}
