package clipboard

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
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
	if logger == nil {
		logger = slog.Default()
	}
	return &Module{cfg: cfg, device: device, bus: bus, logger: logger, platform: newPlatform(), suppress: make(map[string]time.Time)}
}

func (m *Module) Name() string { return "clipboard" }

func (m *Module) Start(ctx context.Context) error {
	watchCtx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	m.logger.Info("clipboard module starting", "device_id", m.device.ID, "watch_interval", m.cfg.WatchInterval.String(), "max_text_bytes", m.cfg.MaxTextBytes)
	if content, err := m.platform.ReadText(); err == nil {
		m.logger.Info("initial system clipboard read", "bytes", len([]byte(content)), "hash", contentHash(content))
		m.onPlatformChange(content)
	} else if !errors.Is(err, ErrUnsupportedPlatform) {
		m.logger.Warn("initial system clipboard read failed", "error", err)
	} else {
		m.logger.Warn("system clipboard integration is unavailable on this platform")
	}
	var lastWatcherError time.Time
	go func() {
		watcher := Watcher{
			platform: m.platform,
			interval: m.cfg.WatchInterval,
			onChange: m.onPlatformChange,
			onError: func(err error) {
				if lastWatcherError.IsZero() || time.Since(lastWatcherError) >= 5*time.Second {
					lastWatcherError = time.Now()
					m.logger.Warn("clipboard watcher read failed", "error", err)
				}
			},
		}
		if err := watcher.Run(watchCtx); err != nil && !errors.Is(err, context.Canceled) {
			m.logger.Warn("clipboard watcher stopped", "error", err)
		}
	}()
	m.logger.Info("clipboard watcher started")
	return nil
}

func (m *Module) Stop(context.Context) error {
	if m.cancel != nil {
		m.cancel()
	}
	m.logger.Info("clipboard module stopped")
	return nil
}

func (m *Module) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/clipboard/latest", m.handleLatest)
	mux.HandleFunc("POST /api/v1/clipboard", m.handlePush)
	mux.HandleFunc("GET /api/v1/clipboard/status", m.handleStatus)
}

func (m *Module) handleLatest(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	item := m.latest
	m.mu.RUnlock()
	if item == nil {
		m.logger.Info("clipboard pull returned empty", "remote", r.RemoteAddr)
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "clipboard is empty"})
		return
	}
	m.logger.Info("clipboard sent to client", "remote", r.RemoteAddr, "bytes", len([]byte(item.Content)), "hash", item.Hash, "source", item.Source)
	writeJSON(w, http.StatusOK, item)
}

func (m *Module) handleStatus(w http.ResponseWriter, _ *http.Request) {
	m.mu.RLock()
	available := m.latest != nil
	m.mu.RUnlock()
	m.logger.Debug("clipboard status requested", "has_latest", available)
	writeJSON(w, http.StatusOK, map[string]any{"module": m.Name(), "enabled": true, "has_latest": available})
}

func (m *Module) handlePush(w http.ResponseWriter, r *http.Request) {
	m.logger.Info("clipboard push received", "remote", r.RemoteAddr, "content_type", r.Header.Get("Content-Type"))
	r.Body = http.MaxBytesReader(w, r.Body, int64(m.cfg.MaxTextBytes)+64*1024)
	req, encoding, err := decodePushRequest(r.Body, r.Header.Get("Content-Type"))
	if err != nil {
		m.logger.Warn("clipboard push rejected: request body could not be decoded", "remote", r.RemoteAddr, "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid clipboard request body"})
		return
	}
	content := req.Content
	if content == "" && req.Text != "" {
		content = req.Text
	}
	if len([]byte(content)) > m.cfg.MaxTextBytes {
		m.logger.Warn("clipboard push rejected: content too large", "remote", r.RemoteAddr, "bytes", len([]byte(content)), "max_text_bytes", m.cfg.MaxTextBytes)
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "clipboard content exceeds configured limit"})
		return
	}
	if content == "" {
		m.logger.Warn("clipboard push rejected: empty content", "remote", r.RemoteAddr, "encoding", encoding)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "content must not be empty"})
		return
	}
	item, accepted, err := m.accept(r.Context(), content, req.Type, req.MimeType, req.Hash, req.DeviceID, req.ID, "remote")
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrClipboardUnavailable) {
			status = http.StatusServiceUnavailable
		}
		m.logger.Error("clipboard push failed", "remote", r.RemoteAddr, "bytes", len([]byte(content)), "error", err, "status", status)
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	m.logger.Info("clipboard push processed", "remote", r.RemoteAddr, "accepted", accepted, "bytes", len([]byte(content)), "hash", item.Hash, "device_id", item.DeviceID, "encoding", encoding)
	writeJSON(w, http.StatusOK, pushResponse{Accepted: accepted, Item: item})
}

func decodePushRequest(body io.Reader, contentType string) (pushRequest, string, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return pushRequest{}, "unknown", err
	}
	trimmed := bytes.TrimSpace(data)
	isJSON := strings.Contains(strings.ToLower(contentType), "application/json") || (len(trimmed) > 0 && trimmed[0] == '{')
	if isJSON {
		var req pushRequest
		if err := json.Unmarshal(data, &req); err != nil {
			return pushRequest{}, "json", err
		}
		return req, "json", nil
	}
	return pushRequest{Type: "text", MimeType: "text/plain", Content: string(data)}, "plain_text", nil
}

func (m *Module) onPlatformChange(content string) {
	if content == "" {
		m.logger.Debug("empty system clipboard change ignored")
		return
	}
	hash := contentHash(content)
	m.mu.Lock()
	if until, ok := m.suppress[hash]; ok && time.Now().Before(until) {
		delete(m.suppress, hash)
		m.mu.Unlock()
		m.logger.Info("system clipboard change ignored: self-write", "bytes", len([]byte(content)), "hash", hash)
		return
	}
	m.mu.Unlock()
	if _, _, err := m.accept(context.Background(), content, "text", "text/plain", hash, m.device.ID, "", "local"); err != nil {
		m.logger.Error("failed to process local clipboard change", "bytes", len([]byte(content)), "hash", hash, "error", err)
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
		m.logger.Info("duplicate clipboard content ignored", "source", source, "bytes", len([]byte(content)), "hash", hash, "device_id", deviceID)
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
		m.logger.Info("writing remote clipboard to system clipboard", "bytes", len([]byte(content)), "hash", hash, "device_id", deviceID)
		if err := m.platform.WriteText(content); err != nil {
			if errors.Is(err, ErrUnsupportedPlatform) {
				return nil, false, fmt.Errorf("%w: system clipboard is unsupported on this platform", ErrClipboardUnavailable)
			}
			return nil, false, fmt.Errorf("%w: write system clipboard: %v", ErrClipboardUnavailable, err)
		}
		m.suppress[hash] = time.Now().Add(2 * time.Second)
		m.logger.Info("system clipboard updated from remote content", "bytes", len([]byte(content)), "hash", hash)
	}
	m.latest = item
	if m.bus != nil {
		m.bus.Publish(eventbus.Event{Type: eventbus.ClipboardChanged, Data: item})
	}
	m.logger.Info("clipboard item accepted", "source", source, "bytes", len([]byte(content)), "hash", hash, "device_id", deviceID)
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
