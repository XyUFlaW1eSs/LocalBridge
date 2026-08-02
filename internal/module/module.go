package module

import (
	"context"
	"errors"
	"net/http"
	"sync"
)

type Module interface {
	Name() string
	Start(context.Context) error
	Stop(context.Context) error
	Routes(*http.ServeMux)
}

type Manager struct {
	mu      sync.RWMutex
	modules []Module
}

func NewManager() *Manager { return &Manager{} }

func (m *Manager) Register(mod Module) error {
	if mod == nil || mod.Name() == "" {
		return errors.New("module must have a name")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, existing := range m.modules {
		if existing.Name() == mod.Name() {
			return errors.New("module already registered: " + mod.Name())
		}
	}
	m.modules = append(m.modules, mod)
	return nil
}

func (m *Manager) Start(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, mod := range m.modules {
		if err := mod.Start(ctx); err != nil {
			return errors.New(mod.Name() + ": " + err.Error())
		}
	}
	return nil
}

func (m *Manager) Stop(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var first error
	for i := len(m.modules) - 1; i >= 0; i-- {
		if err := m.modules[i].Stop(ctx); err != nil && first == nil {
			first = errors.New(m.modules[i].Name() + ": " + err.Error())
		}
	}
	return first
}

func (m *Manager) Routes(mux *http.ServeMux) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, mod := range m.modules {
		mod.Routes(mux)
	}
}
