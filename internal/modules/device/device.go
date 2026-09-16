package device

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
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
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/XyUFlaW1eSs/LocalBridge/internal/config"
	"github.com/XyUFlaW1eSs/LocalBridge/internal/eventbus"
	"github.com/XyUFlaW1eSs/LocalBridge/internal/syncstore"
	"github.com/XyUFlaW1eSs/LocalBridge/internal/transport"
)

const registryVersion = 3
const maxRegistryBytes = 1024 * 1024
const maxPeers = 1024

const (
	defaultPeerTokenTTL    = 30 * 24 * time.Hour
	defaultTokenOverlapTTL = 10 * time.Minute
)

type Module struct {
	cfg              config.DeviceConfig
	security         config.SecurityConfig
	discovery        config.DiscoveryConfig
	apiPort          int
	capabilities     []string
	logger           *slog.Logger
	bus              *eventbus.Bus
	store            *syncstore.Store
	transport        *transport.Client
	localSecure      bool
	localScheme      string
	localFingerprint string

	mu              sync.RWMutex
	peers           map[string]Peer
	discovered      map[string]DiscoveredPeer
	discoveryMu     sync.Mutex
	discoveryConn   *net.UDPConn
	discoveryCancel context.CancelFunc
	healthCancel    context.CancelFunc
	lifecycleCancel context.CancelFunc
	now             func() time.Time
}

func New(cfg config.DeviceConfig, security config.SecurityConfig, discovery config.DiscoveryConfig, apiPort int, capabilities []string, bus *eventbus.Bus, store *syncstore.Store, logger *slog.Logger) (*Module, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if security.PeerTokenTTL == 0 {
		security.PeerTokenTTL = defaultPeerTokenTTL
	}
	if security.TokenOverlapTTL == 0 {
		security.TokenOverlapTTL = defaultTokenOverlapTTL
	}
	if security.PeerTokenTTL < 0 || security.TokenOverlapTTL < 0 || security.TokenOverlapTTL >= security.PeerTokenTTL {
		return nil, errors.New("invalid peer token lifetime configuration")
	}
	m := &Module{cfg: cfg, security: security, discovery: discovery, apiPort: apiPort, capabilities: cleanCapabilities(capabilities), logger: logger, bus: bus, store: store, transport: transport.NewClient(5 * time.Second), peers: make(map[string]Peer), discovered: make(map[string]DiscoveredPeer), now: func() time.Time { return time.Now().UTC() }}
	m.localScheme = "http"
	if err := m.load(); err != nil {
		return nil, err
	}
	return m, nil
}

// SetLocalTransport supplies the listener's advertised transport identity.
// Discovery is only a hint; this information never grants peer trust.
func (m *Module) SetLocalTransport(secure bool, fingerprint string) {
	m.localSecure = secure
	m.localFingerprint = strings.TrimSpace(fingerprint)
	if secure {
		m.localScheme = "https"
	} else {
		m.localScheme = "http"
		m.localFingerprint = ""
	}
}

func (m *Module) Name() string { return "device" }

func (m *Module) Start(ctx context.Context) error {
	runCtx, lifecycleCancel := context.WithCancel(ctx)
	m.lifecycleCancel = lifecycleCancel
	m.mu.RLock()
	count := len(m.peers)
	m.mu.RUnlock()
	m.logger.Info("device module started", "device_id", m.cfg.ID, "peer_count", count, "registry_path", m.cfg.RegistryPath)
	if m.discovery.Enabled {
		if err := m.startDiscovery(runCtx); err != nil {
			lifecycleCancel()
			m.lifecycleCancel = nil
			return err
		}
	}
	if m.cfg.HealthInterval > 0 {
		healthCtx, cancel := context.WithCancel(runCtx)
		m.healthCancel = cancel
		go m.healthLoop(healthCtx)
	}
	if m.bus != nil {
		events := m.bus.Subscribe(runCtx, eventbus.ClipboardChanged, 64)
		go m.forwardClipboard(runCtx, events)
	}
	return nil
}

func (m *Module) Stop(context.Context) error {
	if m.lifecycleCancel != nil {
		m.lifecycleCancel()
		m.lifecycleCancel = nil
	}
	if m.healthCancel != nil {
		m.healthCancel()
		m.healthCancel = nil
	}
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

func (m *Module) ValidatePeerToken(token string) bool {
	token = strings.TrimSpace(token)
	if token == "" {
		return false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	now := m.now()
	for _, peer := range m.peers {
		currentMatch := constantTokenMatch(token, peer.Token)
		previousMatch := constantTokenMatch(token, peer.PreviousToken)
		if (currentMatch && peer.Token != "" && peer.TokenExpiresAt.After(now)) ||
			(previousMatch && peer.PreviousToken != "" && peer.PreviousTokenExpiresAt.After(now)) {
			return true
		}
	}
	return false
}

func (m *Module) healthLoop(ctx context.Context) {
	m.probePeers(ctx)
	ticker := time.NewTicker(m.cfg.HealthInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.probePeers(ctx)
		}
	}
}

func (m *Module) probePeers(ctx context.Context) {
	peers := m.peerSnapshot()
	for _, peer := range peers {
		if ctx.Err() != nil {
			return
		}
		if peer.Address == "" || peer.Port == 0 {
			continue
		}
		if !peerTokenActive(peer, m.now()) {
			m.setPeerHealth(peer.ID, "token_expired", nil)
			continue
		}
		var response struct {
			Capabilities []string `json:"capabilities"`
		}
		err := m.transport.GetJSON(ctx, peerEndpoint(peer), "/api/v1/system/capabilities", &response)
		if err != nil {
			m.setPeerHealth(peer.ID, "offline", nil)
			m.logger.Debug("peer health check failed", "device_id", peer.ID, "address", peer.Address, "error", err)
			continue
		}
		m.setPeerHealth(peer.ID, "online", response.Capabilities)
		m.logger.Debug("peer health check succeeded", "device_id", peer.ID, "address", peer.Address)
	}
}

func (m *Module) setPeerHealth(id, status string, capabilities []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	peer, ok := m.peers[id]
	if !ok {
		return
	}
	peer.Status = status
	if status == "online" {
		peer.LastSeen = time.Now().UTC()
		if len(capabilities) > 0 {
			peer.Capabilities = cleanCapabilities(capabilities)
		}
	}
	m.peers[id] = peer
}

func (m *Module) peerSnapshot() []Peer {
	m.mu.RLock()
	defer m.mu.RUnlock()
	peers := make([]Peer, 0, len(m.peers))
	for _, peer := range m.peers {
		peer.Capabilities = append([]string(nil), peer.Capabilities...)
		peers = append(peers, peer)
	}
	return peers
}

type clipboardEvent struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	MimeType string `json:"mime_type"`
	Content  string `json:"content"`
	Hash     string `json:"hash"`
	DeviceID string `json:"device_id"`
	Source   string `json:"source"`
}

func (m *Module) forwardClipboard(ctx context.Context, events <-chan eventbus.Event) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			if event.Type != eventbus.ClipboardChanged {
				continue
			}
			m.forwardClipboardEvent(ctx, event.Data)
		}
	}
}

func (m *Module) forwardClipboardEvent(ctx context.Context, data any) {
	encoded, err := json.Marshal(data)
	if err != nil {
		m.logger.Warn("clipboard event could not be encoded for peers", "error", err)
		return
	}
	var item clipboardEvent
	if err := json.Unmarshal(encoded, &item); err != nil || item.Source != "local" || item.Content == "" {
		return
	}
	request := map[string]any{"id": item.ID, "type": item.Type, "mime_type": item.MimeType, "content": item.Content, "hash": item.Hash, "device_id": item.DeviceID}
	payload, err := json.Marshal(request)
	if err != nil {
		m.logger.Warn("clipboard event could not be encoded as sync payload", "error", err)
		return
	}
	for _, peer := range m.peerSnapshot() {
		if peer.Address == "" || peer.Port == 0 || !peerTokenActive(peer, m.now()) || !supports(peer.Capabilities, "clipboard.text.push") {
			continue
		}
		var job syncstore.Job
		tracked := m.store != nil
		if tracked {
			jobID := item.ID
			if jobID == "" {
				jobID = item.Hash
			}
			jobID = fmt.Sprintf("clipboard:%s:%s", jobID, peer.ID)
			job, err = m.store.Create(syncstore.Envelope{ID: jobID, Kind: "clipboard.push", Type: item.Type, MIMEType: item.MimeType, Hash: item.Hash, SourceDeviceID: item.DeviceID, TargetDeviceID: peer.ID, CorrelationID: item.ID, Size: len(payload), Payload: payload})
			if err != nil {
				m.logger.Warn("clipboard sync job could not be created", "device_id", peer.ID, "hash", item.Hash, "error", err)
				continue
			}
			if job.State == syncstore.StateDelivered {
				continue
			}
			if _, err := m.store.Update(job.ID, syncstore.StateDelivering, ""); err != nil {
				m.logger.Warn("clipboard sync job could not start", "device_id", peer.ID, "hash", item.Hash, "error", err)
				continue
			}
		}
		var response struct {
			Accepted bool `json:"accepted"`
		}
		err = m.transport.PostJSON(ctx, peerEndpoint(peer), "/api/v1/clipboard", request, &response)
		if err != nil {
			if tracked {
				_, _ = m.store.Update(job.ID, syncstore.StateFailed, err.Error())
			}
			m.logger.Warn("clipboard forwarding failed", "device_id", peer.ID, "bytes", len([]byte(item.Content)), "hash", item.Hash, "error", err)
			continue
		}
		if tracked {
			if _, err := m.store.Update(job.ID, syncstore.StateDelivered, ""); err != nil {
				m.logger.Warn("clipboard sync job could not be completed", "device_id", peer.ID, "hash", item.Hash, "error", err)
			}
		}
		m.logger.Info("clipboard forwarded to peer", "device_id", peer.ID, "bytes", len([]byte(item.Content)), "hash", item.Hash, "accepted", response.Accepted)
	}
}

func supports(capabilities []string, wanted string) bool {
	if len(capabilities) == 0 {
		return true
	}
	for _, capability := range capabilities {
		if capability == wanted {
			return true
		}
	}
	return false
}

func peerEndpoint(peer Peer) transport.Endpoint {
	return transport.Endpoint{Address: peer.Address, Port: peer.Port, Token: peer.Token, Secure: peer.Secure, Scheme: peer.Scheme, CertificateSHA256: peer.CertificateSHA256}
}

func (m *Module) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/devices", m.handleList)
	mux.HandleFunc("GET /api/v1/devices/discovered", m.handleDiscovered)
	mux.HandleFunc("GET /api/v1/devices/{id}", m.handleGet)
	mux.HandleFunc("POST /api/v1/devices/pair", m.handlePair)
	mux.HandleFunc("POST /api/v1/devices/{id}/token/rotate", m.handleRotateToken)
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
	now := m.now()
	for _, peer := range m.peers {
		peers = append(peers, toPublic(peer, now))
	}
	m.mu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"local": map[string]any{"id": m.cfg.ID, "name": m.cfg.Name, "secure": m.localSecure, "scheme": m.localScheme, "certificate_sha256": m.localFingerprint},
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
	writeJSON(w, http.StatusOK, toPublic(peer, m.now()))
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
	if !constantTokenMatch(strings.TrimSpace(req.Code), strings.TrimSpace(m.security.PairingCode)) {
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
	now := m.now()
	peer := Peer{ID: strings.TrimSpace(req.ID), Name: strings.TrimSpace(req.Name), Address: strings.TrimSpace(req.Address), Port: req.Port, Capabilities: cleanCapabilities(req.Capabilities), Status: "paired", Secure: req.Secure, Scheme: normalizedScheme(req.Scheme, req.Secure), CertificateSHA256: strings.TrimSpace(req.CertificateSHA256), PairedAt: now, LastSeen: now}
	m.mu.Lock()
	old, existed := m.peers[peer.ID]
	if !existed && len(m.peers) >= maxPeers {
		m.mu.Unlock()
		writeError(w, http.StatusInsufficientStorage, "device registry is full", requestID(r))
		return
	}
	if existed {
		peer.PairedAt = old.PairedAt
		peer.Token = old.Token
		peer.TokenIssuedAt = old.TokenIssuedAt
		peer.TokenExpiresAt = old.TokenExpiresAt
		peer.PreviousToken = old.PreviousToken
		peer.PreviousTokenExpiresAt = old.PreviousTokenExpiresAt
	}
	peer = m.rotatePeer(peer, token, now)
	m.peers[peer.ID] = peer
	if err := m.saveLocked(); err != nil {
		if existed {
			m.peers[peer.ID] = old
		} else {
			delete(m.peers, peer.ID)
		}
		m.mu.Unlock()
		writeError(w, http.StatusInternalServerError, "failed to persist device registry", requestID(r))
		return
	}
	m.mu.Unlock()
	m.logger.Info("device paired", "device_id", peer.ID, "address", peer.Address, "capability_count", len(peer.Capabilities), "rotated", existed)
	writeJSON(w, http.StatusOK, map[string]any{"paired": true, "rotated": existed, "device": toPublic(peer, now), "token": token})
}

func (m *Module) handleRotateToken(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	now := m.now()
	m.mu.Lock()
	old, ok := m.peers[id]
	if !ok {
		m.mu.Unlock()
		writeError(w, http.StatusNotFound, "device not found", requestID(r))
		return
	}
	if !m.canRotatePeer(r, old, now) {
		m.mu.Unlock()
		writeError(w, http.StatusForbidden, "token rotation requires local management, the management token, or this device's current token", requestID(r))
		return
	}
	token, err := newToken()
	if err != nil {
		m.mu.Unlock()
		writeError(w, http.StatusInternalServerError, "failed to generate device token", requestID(r))
		return
	}
	peer := m.rotatePeer(old, token, now)
	m.peers[id] = peer
	if err := m.saveLocked(); err != nil {
		m.peers[id] = old
		m.mu.Unlock()
		writeError(w, http.StatusInternalServerError, "failed to persist device registry", requestID(r))
		return
	}
	m.mu.Unlock()
	m.logger.Info("device token rotated", "device_id", id, "expires_at", peer.TokenExpiresAt)
	writeJSON(w, http.StatusOK, map[string]any{"rotated": true, "device": toPublic(peer, now), "token": token})
}

func (m *Module) handleRevoke(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	m.mu.Lock()
	old, ok := m.peers[id]
	if !ok {
		m.mu.Unlock()
		writeError(w, http.StatusNotFound, "device not found", requestID(r))
		return
	}
	delete(m.peers, id)
	if err := m.saveLocked(); err != nil {
		m.peers[id] = old
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
		packet := discoveryAnnouncement{Type: "localbridge.discovery.v1", ProtocolVersion: 1, DeviceID: m.cfg.ID, DeviceName: m.cfg.Name, APIPort: m.apiPort, Capabilities: append([]string(nil), m.capabilities...), Nonce: discoveryNonce(), Secure: m.localSecure, Scheme: m.localScheme, CertificateSHA256: m.localFingerprint, Fingerprint: m.localFingerprint}
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
		fingerprint := strings.TrimSpace(announcement.CertificateSHA256)
		if fingerprint == "" {
			fingerprint = strings.TrimSpace(announcement.Fingerprint)
		}
		secure := announcement.Secure || strings.EqualFold(strings.TrimSpace(announcement.Scheme), "https")
		if !validCertificateFingerprint(fingerprint) {
			fingerprint = ""
			secure = false
		}
		peer := DiscoveredPeer{ID: announcement.DeviceID, Name: announcement.DeviceName, Address: peerAddress, Port: announcement.APIPort, Capabilities: cleanCapabilities(announcement.Capabilities), Status: "discovered", Secure: secure, Scheme: normalizedScheme(announcement.Scheme, secure), CertificateSHA256: fingerprint, LastSeen: time.Now().UTC()}
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
	fingerprint := strings.TrimSpace(announcement.CertificateSHA256)
	if fingerprint == "" {
		fingerprint = strings.TrimSpace(announcement.Fingerprint)
	}
	if announcement.Secure || strings.EqualFold(strings.TrimSpace(announcement.Scheme), "https") {
		if !announcement.Secure || strings.TrimSpace(announcement.Scheme) != "https" || !validCertificateFingerprint(fingerprint) {
			return discoveryAnnouncement{}, false
		}
	} else if strings.TrimSpace(announcement.Scheme) != "" && strings.TrimSpace(announcement.Scheme) != "http" {
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
	if err := validatePeerTransport(req.Secure, req.Scheme, req.CertificateSHA256); err != nil {
		return err
	}
	return nil
}

func validatePeerTransport(secure bool, scheme, fingerprint string) error {
	scheme = strings.ToLower(strings.TrimSpace(scheme))
	fingerprint = strings.TrimSpace(fingerprint)
	if secure {
		if scheme != "https" {
			return errors.New("secure peers must use the https scheme")
		}
		if !validCertificateFingerprint(fingerprint) {
			return errors.New("secure peers require a lowercase 64-character SHA-256 certificate fingerprint")
		}
		return nil
	}
	if scheme == "" {
		return nil
	}
	if scheme != "http" {
		return errors.New("legacy peers must use the http scheme")
	}
	if fingerprint != "" {
		return errors.New("HTTP peers must not include a certificate fingerprint")
	}
	return nil
}

func validCertificateFingerprint(value string) bool {
	if len(value) != sha256.Size*2 || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func normalizedScheme(_ string, secure bool) string {
	if secure {
		return "https"
	}
	return "http"
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

func toPublic(peer Peer, now time.Time) publicPeer {
	status := peer.Status
	if peer.Token == "" {
		status = "repair_required"
	} else if !peer.TokenExpiresAt.After(now) {
		status = "token_expired"
	}
	public := publicPeer{ID: peer.ID, Name: peer.Name, Address: peer.Address, Port: peer.Port, Capabilities: append([]string(nil), peer.Capabilities...), Status: status, Secure: peer.Secure, Scheme: normalizedScheme(peer.Scheme, peer.Secure), CertificateSHA256: peer.CertificateSHA256, PairedAt: peer.PairedAt, LastSeen: peer.LastSeen}
	if !peer.TokenIssuedAt.IsZero() {
		issuedAt := peer.TokenIssuedAt
		public.TokenIssuedAt = &issuedAt
	}
	if !peer.TokenExpiresAt.IsZero() {
		expiresAt := peer.TokenExpiresAt
		public.TokenExpiresAt = &expiresAt
	}
	return public
}

func (m *Module) rotatePeer(peer Peer, token string, now time.Time) Peer {
	if peer.Token != "" && peer.TokenExpiresAt.After(now) && m.security.TokenOverlapTTL > 0 {
		peer.PreviousToken = peer.Token
		peer.PreviousTokenExpiresAt = now.Add(m.security.TokenOverlapTTL)
		if peer.TokenExpiresAt.Before(peer.PreviousTokenExpiresAt) {
			peer.PreviousTokenExpiresAt = peer.TokenExpiresAt
		}
	} else {
		peer.PreviousToken = ""
		peer.PreviousTokenExpiresAt = time.Time{}
	}
	peer.Token = token
	peer.TokenIssuedAt = now
	peer.TokenExpiresAt = now.Add(m.security.PeerTokenTTL)
	peer.Status = "paired"
	return peer
}

func peerTokenActive(peer Peer, now time.Time) bool {
	return peer.Token != "" && peer.TokenExpiresAt.After(now)
}

func (m *Module) canRotatePeer(r *http.Request, peer Peer, now time.Time) bool {
	if requestFromLoopback(r) {
		return true
	}
	const prefix = "Bearer "
	provided := strings.TrimSpace(r.Header.Get("Authorization"))
	if len(provided) <= len(prefix) || !strings.EqualFold(provided[:len(prefix)], prefix) {
		return false
	}
	provided = strings.TrimSpace(provided[len(prefix):])
	if constantTokenMatch(provided, strings.TrimSpace(m.security.BearerToken)) {
		return true
	}
	return (peer.TokenExpiresAt.After(now) && constantTokenMatch(provided, peer.Token)) ||
		(peer.PreviousTokenExpiresAt.After(now) && constantTokenMatch(provided, peer.PreviousToken))
}

func requestFromLoopback(r *http.Request) bool {
	host := strings.TrimSpace(r.RemoteAddr)
	if parsed, _, err := net.SplitHostPort(host); err == nil {
		host = parsed
	}
	host = strings.Trim(host, "[]")
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func newToken() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}

func constantTokenMatch(candidate, stored string) bool {
	candidateDigest := sha256.Sum256([]byte(candidate))
	storedDigest := sha256.Sum256([]byte(stored))
	return stored != "" && subtle.ConstantTimeCompare(candidateDigest[:], storedDigest[:]) == 1
}

func (m *Module) load() error {
	file, err := os.Open(m.cfg.RegistryPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read device registry: %w", err)
	}
	data, err := io.ReadAll(io.LimitReader(file, maxRegistryBytes+1))
	closeErr := file.Close()
	if err != nil {
		return fmt.Errorf("read device registry: %w", err)
	}
	if closeErr != nil {
		return fmt.Errorf("close device registry: %w", closeErr)
	}
	if len(data) > maxRegistryBytes {
		return fmt.Errorf("device registry exceeds %d bytes", maxRegistryBytes)
	}
	var state registryFile
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("parse device registry: %w", err)
	}
	if state.Version != 1 && state.Version != 2 && state.Version != registryVersion {
		return fmt.Errorf("unsupported device registry version: %d", state.Version)
	}
	for _, stored := range state.Peers {
		if stored.Secure {
			if err := validatePeerTransport(stored.Secure, stored.Scheme, stored.CertificateSHA256); err != nil {
				return fmt.Errorf("invalid secure peer %q in device registry: %w", stored.ID, err)
			}
		}
		peer := peerFromPersisted(stored)
		if peer.ID == "" || peer.ID == m.cfg.ID {
			continue
		}
		if peer.Token == "" {
			peer.Status = "repair_required"
		}
		m.peers[peer.ID] = peer
	}
	if state.Version < registryVersion {
		for id, peer := range m.peers {
			peer.Secure = false
			peer.Scheme = "http"
			peer.CertificateSHA256 = ""
			m.peers[id] = peer
		}
		if err := m.saveLocked(); err != nil {
			return fmt.Errorf("migrate device registry: %w", err)
		}
	}
	return nil
}

func (m *Module) saveLocked() error {
	peers := make([]persistedPeer, 0, len(m.peers))
	for _, peer := range m.peers {
		peers = append(peers, persistedFromPeer(peer))
	}
	sort.Slice(peers, func(i, j int) bool { return peers[i].ID < peers[j].ID })
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
	tmp, err := os.CreateTemp(dir, ".localbridge-devices-*")
	if err != nil {
		return fmt.Errorf("create device registry temporary file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("protect device registry temporary file: %w", err)
	}
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write device registry: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync device registry: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close device registry: %w", err)
	}
	if err := os.Rename(tmpName, m.cfg.RegistryPath); err != nil {
		if removeErr := os.Remove(m.cfg.RegistryPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return fmt.Errorf("replace device registry: %w (remove existing: %v)", err, removeErr)
		}
		if err := os.Rename(tmpName, m.cfg.RegistryPath); err != nil {
			return fmt.Errorf("replace device registry after removing existing: %w", err)
		}
	}
	return nil
}

func persistedFromPeer(peer Peer) persistedPeer {
	return persistedPeer{ID: peer.ID, Name: peer.Name, Address: peer.Address, Port: peer.Port, Capabilities: append([]string(nil), peer.Capabilities...), Status: peer.Status, Secure: peer.Secure, Scheme: normalizedScheme(peer.Scheme, peer.Secure), CertificateSHA256: peer.CertificateSHA256, PairedAt: peer.PairedAt, LastSeen: peer.LastSeen, Token: peer.Token, TokenIssuedAt: peer.TokenIssuedAt, TokenExpiresAt: peer.TokenExpiresAt, PreviousToken: peer.PreviousToken, PreviousTokenExpiresAt: peer.PreviousTokenExpiresAt}
}

func peerFromPersisted(peer persistedPeer) Peer {
	secure := peer.Secure && peer.Scheme == "https" && validCertificateFingerprint(peer.CertificateSHA256)
	fingerprint := ""
	if secure {
		fingerprint = peer.CertificateSHA256
	}
	return Peer{ID: peer.ID, Name: peer.Name, Address: peer.Address, Port: peer.Port, Capabilities: append([]string(nil), peer.Capabilities...), Status: peer.Status, Secure: secure, Scheme: normalizedScheme(peer.Scheme, secure), CertificateSHA256: fingerprint, PairedAt: peer.PairedAt, LastSeen: peer.LastSeen, Token: peer.Token, TokenIssuedAt: peer.TokenIssuedAt, TokenExpiresAt: peer.TokenExpiresAt, PreviousToken: peer.PreviousToken, PreviousTokenExpiresAt: peer.PreviousTokenExpiresAt}
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
