package native

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/XyUFlaW1eSs/LocalBridge/internal/modules/settings"
)

type fakeRegistry struct {
	values  map[string]map[string]string
	deleted []string
}

func newFakeRegistry() *fakeRegistry {
	return &fakeRegistry{values: make(map[string]map[string]string)}
}

func (r *fakeRegistry) SetString(key, name, value string) error {
	if r.values[key] == nil {
		r.values[key] = make(map[string]string)
	}
	r.values[key][name] = value
	return nil
}

func (r *fakeRegistry) DeleteValue(key, name string) error {
	values, ok := r.values[key]
	if !ok {
		return ErrRegistryNotFound
	}
	if _, ok := values[name]; !ok {
		return ErrRegistryNotFound
	}
	delete(values, name)
	r.deleted = append(r.deleted, key+"|"+name)
	return nil
}

func (r *fakeRegistry) DeleteKey(key string) error {
	if _, ok := r.values[key]; !ok {
		return ErrRegistryNotFound
	}
	delete(r.values, key)
	r.deleted = append(r.deleted, key)
	return nil
}

func TestSyncRegistryEnablesAndDisablesNativeIntegrations(t *testing.T) {
	reg := newFakeRegistry()
	value := settings.Defaults()
	value.AutoStart = true
	value.ExplorerContextMenu = true
	if err := SyncRegistry(reg, `C:\Program Files\LocalBridge\localbridge.exe`, `C:\Users\me\Local Bridge\config.yaml`, value); err != nil {
		t.Fatal(err)
	}
	auto := reg.values[autoStartKey][registryValue]
	if !strings.Contains(auto, `"C:\Program Files\LocalBridge\localbridge.exe"`) || !strings.Contains(auto, `-config "C:\Users\me\Local Bridge\config.yaml"`) {
		t.Fatalf("unsafe or incomplete auto-start command: %q", auto)
	}
	command := reg.values[explorerKey][""]
	if !strings.Contains(command, "-share %*") {
		t.Fatalf("Explorer command does not preserve multi-selection: %q", command)
	}
	if reg.values[explorerMenuKey][""] != explorerLabel || reg.values[explorerMenuKey]["MultiSelectModel"] != "Player" {
		t.Fatalf("Explorer menu metadata is incomplete: %#v", reg.values[explorerMenuKey])
	}

	value.AutoStart = false
	value.ExplorerContextMenu = false
	if err := SyncRegistry(reg, `C:\Program Files\LocalBridge\localbridge.exe`, "", value); err != nil {
		t.Fatal(err)
	}
	if _, ok := reg.values[autoStartKey][registryValue]; ok {
		t.Fatal("auto-start value survived disable")
	}
	if _, ok := reg.values[explorerKey]; ok {
		t.Fatal("Explorer command key survived disable")
	}
	if _, ok := reg.values[explorerMenuKey]; ok {
		t.Fatal("Explorer menu key survived disable")
	}
}

func TestValidateSharePathsRejectsUnsafeSelections(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(file, []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}
	paths, err := ValidateSharePaths([]string{file})
	if err != nil || len(paths) != 1 || !filepath.IsAbs(paths[0]) {
		t.Fatalf("valid file rejected: paths=%#v err=%v", paths, err)
	}
	if _, err := ValidateSharePaths([]string{file, file}); err == nil {
		t.Fatal("duplicate selection was accepted")
	}
	if _, err := ValidateSharePaths([]string{dir}); err == nil {
		t.Fatal("directory selection was accepted")
	}
	if _, err := ValidateSharePaths([]string{filepath.Join(dir, "missing.txt")}); err == nil {
		t.Fatal("missing file was accepted")
	}
	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(file, link); err == nil {
		if _, err := ValidateSharePaths([]string{link}); err == nil {
			t.Fatal("symbolic link was accepted")
		}
	} else if !errors.Is(err, os.ErrPermission) {
		t.Logf("symlink test unavailable: %v", err)
	}
}

func TestQuoteWindowsArgRejectsNewlines(t *testing.T) {
	if _, err := quoteWindowsArg("bad\nargument"); err == nil {
		t.Fatal("newline command argument was accepted")
	}
	quoted, err := quoteWindowsArg(`C:\path with space\`)
	if err != nil || quoted != `"C:\path with space\\"` {
		t.Fatalf("unexpected quoted argument %q err=%v", quoted, err)
	}
}
