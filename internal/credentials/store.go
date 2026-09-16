package credentials

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
)

const (
	storeVersion  = 1
	maxStoreBytes = 1024 * 1024
	maxSecretSize = 64 * 1024
)

var validName = regexp.MustCompile(`^[a-z][a-z0-9_.-]{0,63}$`)

type Store struct {
	path      string
	protector Protector
}

type storeFile struct {
	Version int          `json:"version"`
	Entries []storeEntry `json:"entries"`
}

type storeEntry struct {
	Name       string `json:"name"`
	Ciphertext string `json:"ciphertext"`
}

func NewStore(path string, protector Protector) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("credential store path must not be empty")
	}
	if protector == nil || !protector.Available() {
		return nil, ErrUnavailable
	}
	return &Store{path: path, protector: protector}, nil
}

func ValidateName(name string) error {
	if !validName.MatchString(name) {
		return fmt.Errorf("credential name must match %s", validName.String())
	}
	return nil
}

func (s *Store) Set(name string, secret []byte) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	if len(secret) == 0 || len(secret) > maxSecretSize {
		return fmt.Errorf("credential value must contain between 1 and %d bytes", maxSecretSize)
	}
	entries, err := s.load()
	if err != nil {
		return err
	}
	ciphertext, err := s.protector.Protect(storePurpose(name), secret)
	if err != nil {
		return fmt.Errorf("protect credential %q: %w", name, err)
	}
	entries[name] = base64.StdEncoding.EncodeToString(ciphertext)
	return s.save(entries)
}

func (s *Store) Get(name string) ([]byte, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}
	entries, err := s.load()
	if err != nil {
		return nil, err
	}
	encoded, ok := entries[name]
	if !ok {
		return nil, fmt.Errorf("credential %q was not found", name)
	}
	ciphertext, err := base64.StdEncoding.Strict().DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode credential %q: %w", name, err)
	}
	plaintext, err := s.protector.Unprotect(storePurpose(name), ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decrypt credential %q: %w", name, err)
	}
	if len(plaintext) == 0 || len(plaintext) > maxSecretSize {
		return nil, fmt.Errorf("credential %q has an invalid plaintext size", name)
	}
	return plaintext, nil
}

func (s *Store) Delete(name string) (bool, error) {
	if err := ValidateName(name); err != nil {
		return false, err
	}
	entries, err := s.load()
	if err != nil {
		return false, err
	}
	if _, ok := entries[name]; !ok {
		return false, nil
	}
	delete(entries, name)
	return true, s.save(entries)
}

func (s *Store) Has(name string) (bool, error) {
	if err := ValidateName(name); err != nil {
		return false, err
	}
	entries, err := s.load()
	if err != nil {
		return false, err
	}
	_, ok := entries[name]
	return ok, nil
}

func (s *Store) load() (map[string]string, error) {
	file, err := os.Open(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return make(map[string]string), nil
	}
	if err != nil {
		return nil, fmt.Errorf("open credential store: %w", err)
	}
	data, readErr := io.ReadAll(io.LimitReader(file, maxStoreBytes+1))
	closeErr := file.Close()
	if readErr != nil {
		return nil, fmt.Errorf("read credential store: %w", readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close credential store: %w", closeErr)
	}
	if len(data) > maxStoreBytes {
		return nil, fmt.Errorf("credential store exceeds %d bytes", maxStoreBytes)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var state storeFile
	if err := decoder.Decode(&state); err != nil {
		return nil, fmt.Errorf("parse credential store: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("credential store contains multiple JSON values")
		}
		return nil, fmt.Errorf("parse credential store: %w", err)
	}
	if state.Version != storeVersion {
		return nil, fmt.Errorf("unsupported credential store version: %d", state.Version)
	}
	entries := make(map[string]string, len(state.Entries))
	for _, entry := range state.Entries {
		if err := ValidateName(entry.Name); err != nil {
			return nil, fmt.Errorf("invalid credential store entry: %w", err)
		}
		if _, exists := entries[entry.Name]; exists {
			return nil, fmt.Errorf("duplicate credential store entry %q", entry.Name)
		}
		decoded, err := base64.StdEncoding.Strict().DecodeString(entry.Ciphertext)
		if err != nil || len(decoded) == 0 {
			return nil, fmt.Errorf("credential store entry %q has invalid ciphertext", entry.Name)
		}
		entries[entry.Name] = entry.Ciphertext
	}
	return entries, nil
}

func (s *Store) save(entries map[string]string) error {
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)
	state := storeFile{Version: storeVersion, Entries: make([]storeEntry, 0, len(names))}
	for _, name := range names {
		state.Entries = append(state.Entries, storeEntry{Name: name, Ciphertext: entries[name]})
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode credential store: %w", err)
	}
	data = append(data, '\n')
	if len(data) > maxStoreBytes {
		return fmt.Errorf("credential store exceeds %d bytes", maxStoreBytes)
	}
	return WriteFileAtomic(s.path, data)
}

func storePurpose(name string) string { return "localbridge/credential-store/v1/" + name }
