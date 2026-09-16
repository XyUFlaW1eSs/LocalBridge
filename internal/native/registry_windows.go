//go:build windows

package native

import "golang.org/x/sys/windows/registry"

type windowsRegistry struct{}

func NewWindowsRegistry() Registry { return windowsRegistry{} }

func (windowsRegistry) SetString(key, name, value string) error {
	k, _, err := registry.CurrentUser.CreateKey(key, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetStringValue(name, value)
}

func (windowsRegistry) DeleteValue(key, name string) error {
	k, err := registry.OpenKey(registry.CurrentUser, key, registry.SET_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			return ErrRegistryNotFound
		}
		return err
	}
	defer k.Close()
	if err := k.DeleteValue(name); err == registry.ErrNotExist {
		return ErrRegistryNotFound
	} else {
		return err
	}
}

func (windowsRegistry) DeleteKey(key string) error {
	if err := registry.DeleteKey(registry.CurrentUser, key); err == registry.ErrNotExist {
		return ErrRegistryNotFound
	} else {
		return err
	}
}
