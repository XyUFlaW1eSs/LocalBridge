package clipboard

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/XyUFlaW1eSs/LocalBridge/internal/config"
	"github.com/XyUFlaW1eSs/LocalBridge/internal/eventbus"
)

type Module struct {
	cfg      config.ClipboardConfig
	device   config.DeviceConfig
	bus      *eventbus.Bus
	logger   *slog.Logger
	platform Platform

	mu       sync.RWMutex
	latest   *Item
	suppress map[string]time.Time
	cancel   context.CancelFunc
}

func New(cfg config.ClipboardConfig, device config.DeviceConfig, bus *eventbus.Bus, logger *slog.Logger) *Module {
	return &Module{cfg: cfg, device: device, bus: bus, logger: logger, platform: newPlatform(), suppress: make(map[string]time.Time)}
}

func (m *Module) Name() string { return "clipboard" }

func (m *Module) Start(ctx context.Context) error {
	watchCtx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	if content, err := m.platform.ReadText(); err == nil {
		m.onPlatformChange(content)
	} else if !errors.Is(err, ErrUnsupportedPlatform) {
		m.logger.Debug("initial clipboard read unavailable", "error", err)
	}
	go func() {
		watcher := Watcher{platform: m.platform, interval: m.cfg.WatchInterval, onChange: m.onPlatformChange}
		if err := watcher.Run(watchCtx); err != nil && !errors.Is(err, context.Canceled) {
			m.logger.Warn("clipboard watcher stopped", "error", err)
		}
	}()
	return nil
}

func (m *Module) Stop(context.Context) error {
	if m.cancel != nil {
		m.cancel()
	}
	return nil
}

func (m *Module) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/clipboard/latest", m.handleLatest)
	mux.HandleFunc("POST /api/v1/clipboard", m.handlePush)
	mux.HandleFunc("GET /api/v1/clipboard/status", m.handleStatus)
}

func (m *Module) handleLatest(w http.ResponseWriter, _ *http.Request) {
	m.mu.RLock()
	item := m.latest
	m.mu.RUnlock()
	if item == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "clipboard is empty"})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (m *Module) handleStatus(w http.ResponseWriter, _ *http.Request) {
	m.mu.RLock()
	available := m.latest != nil
	m.mu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]any{"module": m.Name(), "enabled": true, "has_latest": available})
}

func (m *Module) handlePush(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, int64(m.cfg.MaxTextBytes)+64*1024)
	var req pushRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	content := req.Content
	if content == "" && req.Text != "" {
		content = req.Text
	}
	if len([]byte(content)) > m.cfg.MaxTextBytes {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "clipboard content exceeds configured limit"})
		return
	}
	if content == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "content must not be empty"})
		return
	}
	item, accepted, err := m.accept(context.Background(), content, req.Type, req.MimeType, req.Hash, req.DeviceID, req.ID, "remote")
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrClipboardUnavailable) {
			status = http.StatusServiceUnavailable
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, pushResponse{Accepted: accepted, Item: item})
}

func (m *Module) onPlatformChange(content string) {
	if content == "" {
		return
	}
	hash := contentHash(content)
	m.mu.Lock()
	if until, ok := m.suppress[hash]; ok && time.Now().Before(until) {
		delete(m.suppress, hash)
		m.mu.Unlock()
		return
	}
	m.mu.Unlock()
	if _, _, err := m.accept(context.Background(), content, "text", "text/plain", hash, m.device.ID, "", "local"); err != nil {
		m.logger.Warn("failed to publish local clipboard change", "error", err)
	}
}

var ErrClipboardUnavailable = errors.New("clipboard is unavailable")

func (m *Module) accept(_ context.Context, content, typ, mime, hash, deviceID, id, source string) (*Item, bool, error) {
	if hash == "" {
		hash = contentHash(content)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.latest != nil && m.latest.Hash == hash {
		return m.latest, false, nil
	}
	if typ == "" {
		typ = "text"
	}
	if mime == "" {
		mime = "text/plain"
	}
	if deviceID == "" {
		deviceID = m.device.ID
	}
	if id == "" {
		id = fmt.Sprintf("%s-%d", deviceID, time.Now().UnixNano())
	}
	item := &Item{ID: id, Type: typ, MimeType: mime, Content: content, Hash: hash, DeviceID: deviceID, Source: source, CreatedAt: time.Now().UTC()}
	if source == "remote" {
		m.suppress[hash] = time.Now().Add(2 * time.Second)
		if err := m.platform.WriteText(content); err != nil && !errors.Is(err, ErrUnsupportedPlatform) {
			return nil, false, fmt.Errorf("write system clipboard: %w", err)
		}
	}
	m.latest = item
	if m.bus != nil {
		m.bus.Publish(eventbus.Event{Type: eventbus.ClipboardChanged, Data: item})
	}
	return item, true, nil
}

func contentHash(content string) string {
	digest := sha256.Sum256([]byte(content))
	return hex.EncodeToString(digest[:])
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
