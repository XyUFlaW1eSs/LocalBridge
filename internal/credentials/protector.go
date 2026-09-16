package credentials

import "errors"

var ErrUnavailable = errors.New("credential protection is unavailable on this platform")

// Protector encrypts secrets for a specific local security boundary. Purpose
// must be authenticated by implementations so ciphertext cannot be moved
// between fields.
type Protector interface {
	Name() string
	Available() bool
	Protect(purpose string, plaintext []byte) ([]byte, error)
	Unprotect(purpose string, ciphertext []byte) ([]byte, error)
}

const (
	ModeAuto     = "auto"
	ModeRequired = "required"
	ModeDisabled = "disabled"
)

func NormalizeMode(mode string) string {
	if mode == "" {
		return ModeAuto
	}
	return mode
}
