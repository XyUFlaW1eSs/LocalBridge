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
	"net"
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
	cfg          config.DeviceConfig
	security     config.SecurityConfig
	discovery    config.DiscoveryConfig
	apiPort      int
	capabilities []string
	logger       *slog.Logger

	mu              sync.RWMutex
	peers           map[string]Peer
	discovered      map[string]DiscoveredPeer
	discoveryMu     sync.Mutex
	discoveryConn   *net.UDPConn
	discoveryCancel context.CancelFunc
}

func New(cfg config.DeviceConfig, security config.SecurityConfig, discovery config.DiscoveryConfig, apiPort int, capabilities []string, logger *slog.Logger) (*Module, error) {
	if logger == nil {
		logger = slog.Default()
	}
	m := &Module{cfg: cfg, security: security, discovery: discovery, apiPort: apiPort, capabilities: cleanCapabilities(capabilities), logger: logger, peers: make(map[string]Peer), discovered: make(map[string]DiscoveredPeer)}
	if err := m.load(); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Module) Name() string { return "device" }

func (m *Module) Start(ctx context.Context) error {
	m.mu.RLock()
	count := len(m.peers)
	m.mu.RUnlock()
	m.logger.Info("device module started", "device_id", m.cfg.ID, "peer_count", count, "registry_path", m.cfg.RegistryPath)
	if m.discovery.Enabled {
		if err := m.startDiscovery(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (m *Module) Stop(context.Context) error {
	m.discoveryMu.Lock()
	if m.discoveryCancel != nil {
		m.discoveryCancel()
		m.discoveryCancel = nil
	}
	if m.discoveryConn != nil {
		_ = m.discoveryConn.Close()
		m.discoveryConn = nil
	}
	m.discoveryMu.Unlock()
	m.logger.Info("device module stopped")
	return nil
}

func (m *Module) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/devices", m.handleList)
	mux.HandleFunc("GET /api/v1/devices/discovered", m.handleDiscovered)
	mux.HandleFunc("GET /api/v1/devices/{id}", m.handleGet)
	mux.HandleFunc("POST /api/v1/devices/pair", m.handlePair)
	mux.HandleFunc("DELETE /api/v1/devices/{id}", m.handleRevoke)
}

func (m *Module) handleDiscovered(w http.ResponseWriter, _ *http.Request) {
	m.mu.RLock()
	peers := make([]DiscoveredPeer, 0, len(m.discovered))
	for _, peer := range m.discovered {
		peers = append(peers, peer)
	}
	m.mu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]any{"peers": peers})
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

func (m *Module) startDiscovery(ctx context.Context) error {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: m.discovery.Port})
	if err != nil {
		return fmt.Errorf("start device discovery on UDP %d: %w", m.discovery.Port, err)
	}
	discoveryCtx, cancel := context.WithCancel(ctx)
	m.discoveryMu.Lock()
	m.discoveryConn = conn
	m.discoveryCancel = cancel
	m.discoveryMu.Unlock()
	m.logger.Info("device discovery started", "udp_port", m.discovery.Port, "announce_interval", m.discovery.AnnounceInterval.String())
	go m.receiveAnnouncements(discoveryCtx, conn)
	go m.announceLoop(discoveryCtx, conn)
	return nil
}

func (m *Module) announceLoop(ctx context.Context, conn *net.UDPConn) {
	announce := func() {
		packet := discoveryAnnouncement{Type: "localbridge.discovery.v1", ProtocolVersion: 1, DeviceID: m.cfg.ID, DeviceName: m.cfg.Name, APIPort: m.apiPort, Capabilities: append([]string(nil), m.capabilities...), Nonce: discoveryNonce()}
		data, err := json.Marshal(packet)
		if err != nil {
			m.logger.Warn("device discovery announcement encode failed", "error", err)
			return
		}
		_, err = conn.WriteToUDP(data, &net.UDPAddr{IP: net.IPv4bcast, Port: m.discovery.Port})
		if err != nil {
			m.logger.Warn("device discovery announcement failed", "error", err)
			return
		}
		m.logger.Debug("device discovery announcement sent", "bytes", len(data))
	}
	announce()
	ticker := time.NewTicker(m.discovery.AnnounceInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			announce()
		}
	}
}

func (m *Module) receiveAnnouncements(ctx context.Context, conn *net.UDPConn) {
	buffer := make([]byte, 16*1024)
	for {
		if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
			return
		}
		n, address, err := conn.ReadFromUDP(buffer)
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return
			}
			var networkError net.Error
			if errors.As(err, &networkError) && networkError.Timeout() {
				continue
			}
			m.logger.Warn("device discovery receive failed", "error", err)
			continue
		}
		announcement, ok := parseDiscoveryAnnouncement(buffer[:n])
		if !ok || announcement.DeviceID == m.cfg.ID {
			continue
		}
		peerAddress := address.IP.String()
		if peerAddress == "" || announcement.APIPort < 1 || announcement.APIPort > 65535 {
			continue
		}
		peer := DiscoveredPeer{ID: announcement.DeviceID, Name: announcement.DeviceName, Address: peerAddress, Port: announcement.APIPort, Capabilities: cleanCapabilities(announcement.Capabilities), Status: "discovered", LastSeen: time.Now().UTC()}
		m.mu.Lock()
		m.discovered[peer.ID] = peer
		m.mu.Unlock()
		m.logger.Debug("device discovered", "device_id", peer.ID, "address", peer.Address, "port", peer.Port)
	}
}

func parseDiscoveryAnnouncement(data []byte) (discoveryAnnouncement, bool) {
	if len(data) == 0 || len(data) > 16*1024 {
		return discoveryAnnouncement{}, false
	}
	var announcement discoveryAnnouncement
	if err := json.Unmarshal(data, &announcement); err != nil {
		return discoveryAnnouncement{}, false
	}
	if announcement.Type != "localbridge.discovery.v1" || announcement.ProtocolVersion != 1 || strings.TrimSpace(announcement.DeviceID) == "" || strings.TrimSpace(announcement.Nonce) == "" {
		return discoveryAnnouncement{}, false
	}
	return announcement, true
}

func discoveryNonce() string {
	token, err := newToken()
	if err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return token[:16]
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
