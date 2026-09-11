package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const maxStoreBytes = 64 * 1024

type Store struct {
	path  string
	mu    sync.RWMutex
	value Settings
}

func NewStore(path string) (*Store, error) {
	if path == "" {
		return nil, errors.New("settings store path must not be empty")
	}
	s := &Store{path: path, value: Defaults()}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Get() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.value
}

func (s *Store) Set(value Settings) (Settings, error) {
	value.Version = version
	s.mu.Lock()
	defer s.mu.Unlock()
	old := s.value
	s.value = value
	if err := s.saveLocked(); err != nil {
		s.value = old
		return Settings{}, err
	}
	return value, nil
}

func (s *Store) Reset() (Settings, error) { return s.Set(Defaults()) }

func (s *Store) load() error {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return s.save()
	}
	if err != nil {
		return fmt.Errorf("read settings store: %w", err)
	}
	if len(data) > maxStoreBytes {
		return fmt.Errorf("settings store exceeds %d bytes", maxStoreBytes)
	}
	var value Settings
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("parse settings store: %w", err)
	}
	if value.Version != version {
		return fmt.Errorf("unsupported settings version: %d", value.Version)
	}
	s.value = value
	return nil
}

func (s *Store) save() error { s.mu.Lock(); defer s.mu.Unlock(); return s.saveLocked() }

func (s *Store) saveLocked() error {
	data, err := json.MarshalIndent(s.value, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create settings directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".localbridge-settings-*")
	if err != nil {
		return fmt.Errorf("create settings temporary file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("protect settings store: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write settings store: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close settings store: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		if removeErr := os.Remove(s.path); removeErr != nil {
			return fmt.Errorf("replace settings store: %w (remove existing: %v)", err, removeErr)
		}
		if err := os.Rename(tmpName, s.path); err != nil {
			return fmt.Errorf("replace settings store after removing existing: %w", err)
		}
	}
	return nil
}
