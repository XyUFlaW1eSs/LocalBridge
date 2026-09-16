package native

import (
	"errors"
	"fmt"
	"strings"

	"github.com/XyUFlaW1eSs/LocalBridge/internal/modules/settings"
)

const (
	autoStartKey    = `Software\Microsoft\Windows\CurrentVersion\Run`
	explorerKey     = `Software\Classes\*\shell\LocalBridgeShare\command`
	explorerMenuKey = `Software\Classes\*\shell\LocalBridgeShare`
	registryValue   = "LocalBridge"
)

type Registry interface {
	SetString(key, name, value string) error
	DeleteValue(key, name string) error
	DeleteKey(key string) error
}

func SyncRegistry(reg Registry, executable, configPath string, value settings.Settings) error {
	if reg == nil {
		return errors.New("registry backend is nil")
	}
	if strings.TrimSpace(executable) == "" {
		return errors.New("native executable path is empty")
	}
	autoCommand, err := windowsCommand(executable, configPath, "")
	if err != nil {
		return err
	}
	if value.AutoStart {
		if err := reg.SetString(autoStartKey, registryValue, autoCommand); err != nil {
			return fmt.Errorf("enable auto-start: %w", err)
		}
	} else if err := reg.DeleteValue(autoStartKey, registryValue); err != nil && !errors.Is(err, ErrRegistryNotFound) {
		return fmt.Errorf("disable auto-start: %w", err)
	}
	explorerCommand, err := windowsCommand(executable, configPath, "-share %*")
	if err != nil {
		return err
	}
	if value.ExplorerContextMenu {
		if err := reg.SetString(explorerKey, "", explorerCommand); err != nil {
			return fmt.Errorf("enable Explorer context menu: %w", err)
		}
	} else if err := reg.DeleteKey(explorerMenuKey); err != nil && !errors.Is(err, ErrRegistryNotFound) {
		return fmt.Errorf("disable Explorer context menu: %w", err)
	}
	return nil
}

var ErrRegistryNotFound = errors.New("registry value not found")

func windowsCommand(executable, configPath, suffix string) (string, error) {
	command, err := quoteWindowsArg(executable)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(configPath) != "" {
		quoted, err := quoteWindowsArg(configPath)
		if err != nil {
			return "", err
		}
		command += " -config " + quoted
	}
	if suffix != "" {
		command += " " + suffix
	}
	return command, nil
}

func quoteWindowsArg(value string) (string, error) {
	if value == "" {
		return `""`, nil
	}
	if strings.IndexAny(value, "\r\n") >= 0 {
		return "", errors.New("command argument contains a newline")
	}
	quoted := `"`
	backslashes := 0
	for _, char := range value {
		switch char {
		case '\\':
			backslashes++
		case '"':
			quoted += strings.Repeat("\\", backslashes*2+1) + `"`
			backslashes = 0
		default:
			if backslashes > 0 {
				quoted += strings.Repeat("\\", backslashes)
				backslashes = 0
			}
			quoted += string(char)
		}
	}
	if backslashes > 0 {
		quoted += strings.Repeat("\\", backslashes*2)
	}
	return quoted + `"`, nil
}
