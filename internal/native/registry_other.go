//go:build !windows

package native

import (
	"sync"
)

type memoryRegistry struct {
	mu     sync.Mutex
	values map[string]map[string]string
}

func NewMemoryRegistry() Registry {
	return &memoryRegistry{values: make(map[string]map[string]string)}
}

func (r *memoryRegistry) SetString(key, name, value string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.values[key] == nil {
		r.values[key] = make(map[string]string)
	}
	r.values[key][name] = value
	return nil
}

func (r *memoryRegistry) DeleteValue(key, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	values, ok := r.values[key]
	if !ok {
		return ErrRegistryNotFound
	}
	if _, ok := values[name]; !ok {
		return ErrRegistryNotFound
	}
	delete(values, name)
	return nil
}

func (r *memoryRegistry) DeleteKey(key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.values[key]; !ok {
		return ErrRegistryNotFound
	}
	delete(r.values, key)
	return nil
}
