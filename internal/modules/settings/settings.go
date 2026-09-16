package settings

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"github.com/XyUFlaW1eSs/LocalBridge/internal/config"
	"github.com/XyUFlaW1eSs/LocalBridge/internal/server"
)

type Module struct {
	store  *Store
	logger *slog.Logger
}

func New(cfg config.SettingsConfig, logger *slog.Logger) (*Module, error) {
	store, err := NewStore(cfg.StorePath)
	if err != nil {
		return nil, err
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Module{store: store, logger: logger}, nil
}

func (m *Module) Name() string                { return "settings" }
func (m *Module) Start(context.Context) error { m.logger.Info("settings module started"); return nil }
func (m *Module) Stop(context.Context) error  { m.logger.Info("settings module stopped"); return nil }
func (m *Module) Store() *Store               { return m.store }

func (m *Module) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/settings", m.handleGet)
	mux.HandleFunc("PUT /api/v1/settings", m.handleSet)
	mux.HandleFunc("POST /api/v1/settings/reset", m.handleReset)
}

func (m *Module) handleGet(w http.ResponseWriter, r *http.Request) {
	if !m.allow(w, r) {
		return
	}
	writeJSON(w, http.StatusOK, m.store.Get())
}

func (m *Module) handleSet(w http.ResponseWriter, r *http.Request) {
	if !m.allow(w, r) {
		return
	}
	body := http.MaxBytesReader(w, r.Body, 64*1024)
	var value Settings
	if err := json.NewDecoder(body).Decode(&value); err != nil {
		writeError(w, http.StatusBadRequest, "invalid settings", r)
		return
	}
	value.Version = version
	updated, err := m.store.Set(value)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to persist settings", r)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (m *Module) handleReset(w http.ResponseWriter, r *http.Request) {
	if !m.allow(w, r) {
		return
	}
	value, err := m.store.Reset()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to reset settings", r)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (m *Module) allow(w http.ResponseWriter, r *http.Request) bool {
	if server.Authenticated(r) || loopback(r) {
		return true
	}
	writeError(w, http.StatusForbidden, "settings management requires loopback or authentication", r)
	return false
}

func loopback(r *http.Request) bool {
	host := strings.TrimSpace(r.RemoteAddr)
	if parsed, _, err := net.SplitHostPort(host); err == nil {
		host = parsed
	}
	host = strings.Trim(host, "[]")
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, status int, message string, r *http.Request) {
	writeJSON(w, status, map[string]string{"error": message, "request_id": strings.TrimSpace(r.Header.Get("X-Request-ID"))})
}
