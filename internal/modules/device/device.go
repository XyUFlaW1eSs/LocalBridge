package device

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/XyUFlaW1eSs/LocalBridge/internal/config"
)

const registryVersion = 1
const maxRegistryBytes = 1024 * 1024
const maxPeers = 1024

type Module struct {
	cfg      config.DeviceConfig
	security config.SecurityConfig
	logger   *slog.Logger

	mu    sync.RWMutex
	peers map[string]Peer
}

func New(cfg config.DeviceConfig, security config.SecurityConfig, logger *slog.Logger) (*Module, error) {
	if logger == nil {
		logger = slog.Default()
	}
	m := &Module{cfg: cfg, security: security, logger: logger, peers: make(map[string]Peer)}
	if err := m.load(); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Module) Name() string { return "device" }

func (m *Module) Start(_ context.Context) error {
	m.mu.RLock()
	count := len(m.peers)
	m.mu.RUnlock()
	m.logger.Info("device module started", "device_id", m.cfg.ID, "peer_count", count, "registry_path", m.cfg.RegistryPath)
	return nil
}

func (m *Module) Stop(context.Context) error {
	m.logger.Info("device module stopped")
	return nil
}

func (m *Module) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/devices", m.handleList)
	mux.HandleFunc("GET /api/v1/devices/{id}", m.handleGet)
	mux.HandleFunc("POST /api/v1/devices/pair", m.handlePair)
	mux.HandleFunc("DELETE /api/v1/devices/{id}", m.handleRevoke)
}

func (m *Module) handleList(w http.ResponseWriter, _ *http.Request) {
	m.mu.RLock()
	peers := make([]publicPeer, 0, len(m.peers))
	for _, peer := range m.peers {
		peers = append(peers, toPublic(peer))
	}
	m.mu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"local": map[string]string{"id": m.cfg.ID, "name": m.cfg.Name},
		"peers": peers,
	})
}

func (m *Module) handleGet(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	m.mu.RLock()
	peer, ok := m.peers[id]
	m.mu.RUnlock()
	if !ok {
		writeError(w, http.StatusNotFound, "device not found", requestID(r))
		return
	}
	writeJSON(w, http.StatusOK, toPublic(peer))
}

func (m *Module) handlePair(w http.ResponseWriter, r *http.Request) {
	if m.security.PairingCode == "" {
		writeError(w, http.StatusServiceUnavailable, "pairing is not configured", requestID(r))
		return
	}
	body := http.MaxBytesReader(w, r.Body, 64*1024)
	var req pairRequest
	decoder := json.NewDecoder(body)
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid pairing request", requestID(r))
		return
	}
	if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(req.Code)), []byte(m.security.PairingCode)) != 1 {
		writeError(w, http.StatusUnauthorized, "invalid pairing code", requestID(r))
		return
	}
	if err := validatePairRequest(req, m.cfg.ID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), requestID(r))
		return
	}
	token, err := newToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate device token", requestID(r))
		return
	}
	now := time.Now().UTC()
	peer := Peer{ID: strings.TrimSpace(req.ID), Name: strings.TrimSpace(req.Name), Address: strings.TrimSpace(req.Address), Port: req.Port, Capabilities: cleanCapabilities(req.Capabilities), Status: "paired", PairedAt: now, LastSeen: now, Token: token}
	m.mu.Lock()
	_, existed := m.peers[peer.ID]
	if !existed && len(m.peers) >= maxPeers {
		m.mu.Unlock()
		writeError(w, http.StatusInsufficientStorage, "device registry is full", requestID(r))
		return
	}
	if old, ok := m.peers[peer.ID]; ok {
		peer.PairedAt = old.PairedAt
	}
	m.peers[peer.ID] = peer
	if err := m.saveLocked(); err != nil {
		m.mu.Unlock()
		writeError(w, http.StatusInternalServerError, "failed to persist device registry", requestID(r))
		return
	}
	m.mu.Unlock()
	m.logger.Info("device paired", "device_id", peer.ID, "address", peer.Address, "capability_count", len(peer.Capabilities), "rotated", existed)
	writeJSON(w, http.StatusOK, map[string]any{"paired": true, "rotated": existed, "device": toPublic(peer), "token": token})
}

func (m *Module) handleRevoke(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	m.mu.Lock()
	if _, ok := m.peers[id]; !ok {
		m.mu.Unlock()
		writeError(w, http.StatusNotFound, "device not found", requestID(r))
		return
	}
	delete(m.peers, id)
	if err := m.saveLocked(); err != nil {
		m.mu.Unlock()
		writeError(w, http.StatusInternalServerError, "failed to persist device registry", requestID(r))
		return
	}
	m.mu.Unlock()
	m.logger.Info("device revoked", "device_id", id)
	w.WriteHeader(http.StatusNoContent)
}

func validatePairRequest(req pairRequest, localID string) error {
	if strings.TrimSpace(req.ID) == "" {
		return errors.New("device id must not be empty")
	}
	if strings.TrimSpace(req.ID) == localID {
		return errors.New("device id must differ from local device")
	}
	if len(strings.TrimSpace(req.ID)) > 128 || len(strings.TrimSpace(req.Name)) > 256 || len(strings.TrimSpace(req.Address)) > 256 {
		return errors.New("device identity fields are too long")
	}
	if req.Port < 0 || req.Port > 65535 {
		return errors.New("device port must be between 0 and 65535")
	}
	return nil
}

func cleanCapabilities(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || len(value) > 128 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func toPublic(peer Peer) publicPeer {
	return publicPeer{ID: peer.ID, Name: peer.Name, Address: peer.Address, Port: peer.Port, Capabilities: append([]string(nil), peer.Capabilities...), Status: peer.Status, PairedAt: peer.PairedAt, LastSeen: peer.LastSeen}
}

func newToken() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}

func (m *Module) load() error {
	file, err := os.Open(m.cfg.RegistryPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read device registry: %w", err)
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxRegistryBytes+1))
	if err != nil {
		return fmt.Errorf("read device registry: %w", err)
	}
	if len(data) > maxRegistryBytes {
		return fmt.Errorf("device registry exceeds %d bytes", maxRegistryBytes)
	}
	var state registryFile
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("parse device registry: %w", err)
	}
	if state.Version != registryVersion {
		return fmt.Errorf("unsupported device registry version: %d", state.Version)
	}
	for _, peer := range state.Peers {
		if peer.ID == "" || peer.ID == m.cfg.ID {
			continue
		}
		m.peers[peer.ID] = peer
	}
	return nil
}

func (m *Module) saveLocked() error {
	peers := make([]Peer, 0, len(m.peers))
	for _, peer := range m.peers {
		peers = append(peers, peer)
	}
	data, err := json.MarshalIndent(registryFile{Version: registryVersion, Peers: peers}, "", "  ")
	if err != nil {
		return err
	}
	if len(data) > maxRegistryBytes {
		return fmt.Errorf("device registry exceeds %d bytes", maxRegistryBytes)
	}
	dir := filepath.Dir(m.cfg.RegistryPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create device registry directory: %w", err)
	}
	return os.WriteFile(m.cfg.RegistryPath, append(data, '\n'), 0600)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func requestID(r *http.Request) string { return strings.TrimSpace(r.Header.Get("X-Request-ID")) }

func writeError(w http.ResponseWriter, status int, message, requestID string) {
	writeJSON(w, status, map[string]string{"error": message, "request_id": requestID})
}
