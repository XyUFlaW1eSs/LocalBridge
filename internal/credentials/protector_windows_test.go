//go:build windows

package credentials

import (
	"bytes"
	"testing"
)

func TestDPAPIRoundTripAndPurposeBinding(t *testing.T) {
	protector := NewPlatformProtector()
	plaintext := []byte("dpapi-round-trip-secret")
	ciphertext, err := protector.Protect("purpose-a", plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(ciphertext, plaintext) {
		t.Fatal("DPAPI ciphertext contains plaintext")
	}
	got, err := protector.Unprotect("purpose-a", ciphertext)
	if err != nil || !bytes.Equal(got, plaintext) {
		t.Fatalf("round trip got %q, err=%v", got, err)
	}
	if _, err := protector.Unprotect("purpose-b", ciphertext); err == nil {
		t.Fatal("DPAPI accepted ciphertext under a different purpose")
	}
	ciphertext[len(ciphertext)/2] ^= 0xff
	if _, err := protector.Unprotect("purpose-a", ciphertext); err == nil {
		t.Fatal("DPAPI accepted damaged ciphertext")
	}
}
