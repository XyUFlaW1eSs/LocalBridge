//go:build windows

package credentials

import (
	"fmt"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

type platformProtector struct{}

func NewPlatformProtector() Protector { return platformProtector{} }

func (platformProtector) Name() string    { return "dpapi-current-user" }
func (platformProtector) Available() bool { return true }

func (platformProtector) Protect(purpose string, plaintext []byte) ([]byte, error) {
	if purpose == "" {
		return nil, fmt.Errorf("credential purpose must not be empty")
	}
	in := dataBlob(plaintext)
	entropyBytes := []byte(purpose)
	entropy := dataBlob(entropyBytes)
	var out windows.DataBlob
	if err := windows.CryptProtectData(&in, nil, &entropy, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		return nil, fmt.Errorf("protect credential with Windows DPAPI: %w", err)
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data))) //nolint:errcheck
	result := append([]byte(nil), unsafe.Slice(out.Data, int(out.Size))...)
	runtime.KeepAlive(plaintext)
	runtime.KeepAlive(entropyBytes)
	return result, nil
}

func (platformProtector) Unprotect(purpose string, ciphertext []byte) ([]byte, error) {
	if purpose == "" {
		return nil, fmt.Errorf("credential purpose must not be empty")
	}
	in := dataBlob(ciphertext)
	entropyBytes := []byte(purpose)
	entropy := dataBlob(entropyBytes)
	var out windows.DataBlob
	if err := windows.CryptUnprotectData(&in, nil, &entropy, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		return nil, fmt.Errorf("unprotect credential with Windows DPAPI: %w", err)
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data))) //nolint:errcheck
	result := append([]byte(nil), unsafe.Slice(out.Data, int(out.Size))...)
	runtime.KeepAlive(ciphertext)
	runtime.KeepAlive(entropyBytes)
	return result, nil
}

func dataBlob(data []byte) windows.DataBlob {
	if len(data) == 0 {
		return windows.DataBlob{}
	}
	return windows.DataBlob{Size: uint32(len(data)), Data: &data[0]}
}
