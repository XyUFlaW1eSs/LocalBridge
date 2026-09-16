//go:build !windows

package credentials

type unavailableProtector struct{}

func NewPlatformProtector() Protector { return unavailableProtector{} }

func (unavailableProtector) Name() string    { return "unavailable" }
func (unavailableProtector) Available() bool { return false }
func (unavailableProtector) Protect(string, []byte) ([]byte, error) {
	return nil, ErrUnavailable
}
func (unavailableProtector) Unprotect(string, []byte) ([]byte, error) {
	return nil, ErrUnavailable
}
