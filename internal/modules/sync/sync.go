package sync

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/XyUFlaW1eSs/LocalBridge/internal/syncstore"
)

type Module struct {
	store  *syncstore.Store
	logger *slog.Logger
}

func New(store *syncstore.Store, logger *slog.Logger) *Module {
	if logger == nil {
		logger = slog.Default()
	}
	return &Module{store: store, logger: logger}
}

func (m *Module) Name() string { return "sync" }

func (m *Module) Start(context.Context) error {
	m.logger.Info("sync module started")
	return nil
}

func (m *Module) Stop(context.Context) error {
	m.logger.Info("sync module stopped")
	return nil
}

func (m *Module) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/sync/jobs", m.handleList)
	mux.HandleFunc("GET /api/v1/sync/jobs/{id}", m.handleGet)
}

func (m *Module) handleList(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	if limit < 1 {
		limit = 1
	}
	if limit > 1000 {
		limit = 1000
	}
	writeJSON(w, http.StatusOK, map[string]any{"jobs": m.store.List(limit)})
}

func (m *Module) handleGet(w http.ResponseWriter, r *http.Request) {
	job, ok := m.store.Get(strings.TrimSpace(r.PathValue("id")))
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "sync job not found"})
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
