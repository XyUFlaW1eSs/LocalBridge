package credentials

import (
	"fmt"
	"strings"
)

type Protection struct {
	Mode      string
	Effective string
	Enabled   bool
	Protector Protector
}

func ResolveProtection(mode string, protector Protector) (Protection, error) {
	mode = NormalizeMode(strings.TrimSpace(mode))
	if mode != ModeAuto && mode != ModeRequired && mode != ModeDisabled {
		return Protection{}, fmt.Errorf("unsupported credential protection mode %q", mode)
	}
	if mode == ModeDisabled {
		return Protection{Mode: mode, Effective: ModeDisabled}, nil
	}
	if protector == nil {
		protector = NewPlatformProtector()
	}
	if !protector.Available() {
		return Protection{}, fmt.Errorf("credential protection mode %q: %w; set credential_protection to disabled only for explicit legacy compatibility", mode, ErrUnavailable)
	}
	return Protection{Mode: mode, Effective: protector.Name(), Enabled: true, Protector: protector}, nil
}
