package device

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/XyUFlaW1eSs/LocalBridge/internal/config"
	"github.com/XyUFlaW1eSs/LocalBridge/internal/eventbus"
	"github.com/XyUFlaW1eSs/LocalBridge/internal/syncstore"
)

type registryTestProtector struct {
	failProtect bool
}

func (registryTestProtector) Name() string    { return "test-protector" }
func (registryTestProtector) Available() bool { return true }
func (p registryTestProtector) Protect(purpose string, plaintext []byte) ([]byte, error) {
	if p.failProtect {
		return nil, errors.New("injected protection failure")
	}
	return append(append([]byte(purpose), 0), plaintext...), nil
}
func (registryTestProtector) Unprotect(purpose string, ciphertext []byte) ([]byte, error) {
	prefix := append([]byte(purpose), 0)
	if !bytes.HasPrefix(ciphertext, prefix) {
		return nil, errors.New("purpose mismatch or damaged ciphertext")
	}
	return append([]byte(nil), ciphertext[len(prefix):]...), nil
}

func TestPairListAndRevoke(t *testing.T) {
	path := filepath.Join(t.TempDir(), "devices.json")
	m, err := New(config.DeviceConfig{ID: "windows-pc", Name: "Windows", RegistryPath: path}, config.SecurityConfig{PairingCode: "pair-me"}, config.DiscoveryConfig{}, 8899, nil, nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	m.Routes(mux)

	pair := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/devices/pair", strings.NewReader(`{"code":"pair-me","id":"iphone","name":"iPhone","address":"192.168.1.20","port":8899,"capabilities":["clipboard.text.push","clipboard.text.push"]}`))
	request.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(pair, request)
	if pair.Code != http.StatusOK {
		t.Fatalf("pair failed: %d %s", pair.Code, pair.Body.String())
	}
	var paired struct {
		Token  string     `json:"token"`
		Device publicPeer `json:"device"`
	}
	if err := json.Unmarshal(pair.Body.Bytes(), &paired); err != nil {
		t.Fatal(err)
	}
	if len(paired.Token) < 32 || paired.Device.ID != "iphone" || len(paired.Device.Capabilities) != 1 {
		t.Fatalf("unexpected pair response: %s", pair.Body.String())
	}
	if !m.ValidatePeerToken(paired.Token) || m.ValidatePeerToken("invalid-peer-token") {
		t.Fatal("peer token validation failed")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	reloaded, err := New(config.DeviceConfig{ID: "windows-pc", Name: "Windows", RegistryPath: path}, config.SecurityConfig{PairingCode: "pair-me"}, config.DiscoveryConfig{}, 8899, nil, nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	reloadedMux := http.NewServeMux()
	reloaded.Routes(reloadedMux)
	if !reloaded.ValidatePeerToken(paired.Token) {
		t.Fatal("persisted peer token was not valid after restart")
	}
	reloadedList := httptest.NewRecorder()
	reloadedMux.ServeHTTP(reloadedList, httptest.NewRequest(http.MethodGet, "/api/v1/devices/iphone", nil))
	if reloadedList.Code != http.StatusOK {
		t.Fatalf("reloaded registry did not contain peer: %d %s", reloadedList.Code, reloadedList.Body.String())
	}

	list := httptest.NewRecorder()
	mux.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil))
	if list.Code != http.StatusOK || strings.Contains(list.Body.String(), paired.Token) {
		t.Fatalf("list should not expose peer token: %d %s", list.Code, list.Body.String())
	}

	revoke := httptest.NewRecorder()
	mux.ServeHTTP(revoke, httptest.NewRequest(http.MethodDelete, "/api/v1/devices/iphone", nil))
	if revoke.Code != http.StatusNoContent {
		t.Fatalf("revoke failed: %d %s", revoke.Code, revoke.Body.String())
	}
	if m.ValidatePeerToken(paired.Token) {
		t.Fatal("revoked peer token remained valid")
	}
}

func TestPeerTokenRotationOverlapAndExpiry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "devices.json")
	security := config.SecurityConfig{PairingCode: "pair-me", PeerTokenTTL: time.Hour, TokenOverlapTTL: 5 * time.Minute}
	m, err := New(config.DeviceConfig{ID: "windows-pc", Name: "Windows", RegistryPath: path}, security, config.DiscoveryConfig{}, 8899, nil, nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 16, 4, 0, 0, 0, time.UTC)
	m.now = func() time.Time { return now }
	mux := http.NewServeMux()
	m.Routes(mux)

	pair := httptest.NewRecorder()
	mux.ServeHTTP(pair, httptest.NewRequest(http.MethodPost, "/api/v1/devices/pair", strings.NewReader(`{"code":"pair-me","id":"iphone","name":"iPhone"}`)))
	if pair.Code != http.StatusOK {
		t.Fatalf("pair failed: %d %s", pair.Code, pair.Body.String())
	}
	var initial struct {
		Token  string     `json:"token"`
		Device publicPeer `json:"device"`
	}
	if err := json.Unmarshal(pair.Body.Bytes(), &initial); err != nil {
		t.Fatal(err)
	}
	if initial.Device.TokenExpiresAt == nil || !initial.Device.TokenExpiresAt.Equal(now.Add(time.Hour)) || !m.ValidatePeerToken(initial.Token) {
		t.Fatalf("unexpected initial token response: %s", pair.Body.String())
	}

	now = now.Add(10 * time.Minute)
	rotate := httptest.NewRecorder()
	rotateRequest := httptest.NewRequest(http.MethodPost, "/api/v1/devices/iphone/token/rotate", nil)
	rotateRequest.Header.Set("Authorization", "Bearer "+initial.Token)
	mux.ServeHTTP(rotate, rotateRequest)
	if rotate.Code != http.StatusOK {
		t.Fatalf("rotate failed: %d %s", rotate.Code, rotate.Body.String())
	}
	var rotated struct {
		Token  string     `json:"token"`
		Device publicPeer `json:"device"`
	}
	if err := json.Unmarshal(rotate.Body.Bytes(), &rotated); err != nil {
		t.Fatal(err)
	}
	if rotated.Token == initial.Token || !m.ValidatePeerToken(initial.Token) || !m.ValidatePeerToken(rotated.Token) {
		t.Fatalf("rotation did not preserve bounded overlap: %s", rotate.Body.String())
	}
	forbidden := httptest.NewRecorder()
	forbiddenRequest := httptest.NewRequest(http.MethodPost, "/api/v1/devices/iphone/token/rotate", nil)
	forbiddenRequest.Header.Set("Authorization", "Bearer unrelated-peer-token")
	mux.ServeHTTP(forbidden, forbiddenRequest)
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("unrelated peer token rotated device: %d %s", forbidden.Code, forbidden.Body.String())
	}

	now = now.Add(6 * time.Minute)
	if m.ValidatePeerToken(initial.Token) || !m.ValidatePeerToken(rotated.Token) {
		t.Fatal("old token survived overlap or current token stopped early")
	}
	reloaded, err := New(config.DeviceConfig{ID: "windows-pc", Name: "Windows", RegistryPath: path}, security, config.DiscoveryConfig{}, 8899, nil, nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	reloaded.now = func() time.Time { return now }
	if !reloaded.ValidatePeerToken(rotated.Token) || reloaded.ValidatePeerToken(initial.Token) {
		t.Fatal("rotated token state did not survive restart")
	}

	now = time.Date(2026, 9, 16, 5, 11, 0, 0, time.UTC)
	if reloaded.ValidatePeerToken(rotated.Token) {
		t.Fatal("current token survived its configured TTL")
	}
	get := httptest.NewRecorder()
	reloadedMux := http.NewServeMux()
	reloaded.Routes(reloadedMux)
	reloadedMux.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/api/v1/devices/iphone", nil))
	if get.Code != http.StatusOK || !strings.Contains(get.Body.String(), `"status":"token_expired"`) || strings.Contains(get.Body.String(), rotated.Token) {
		t.Fatalf("expired public peer is incorrect or leaks token: %d %s", get.Code, get.Body.String())
	}
}

func TestRegistryV1MigratesMissingTokensToRepairRequired(t *testing.T) {
	path := filepath.Join(t.TempDir(), "devices.json")
	legacy := `{"version":1,"peers":[{"id":"legacy-phone","name":"Legacy","status":"paired","paired_at":"2026-01-01T00:00:00Z","last_seen":"2026-01-01T00:00:00Z"}]}`
	if err := os.WriteFile(path, []byte(legacy), 0600); err != nil {
		t.Fatal(err)
	}
	m, err := New(config.DeviceConfig{ID: "windows-pc", Name: "Windows", RegistryPath: path}, config.SecurityConfig{}, config.DiscoveryConfig{}, 8899, nil, nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	if m.ValidatePeerToken("anything") {
		t.Fatal("legacy peer without a token authenticated")
	}
	mux := http.NewServeMux()
	m.Routes(mux)
	get := httptest.NewRecorder()
	mux.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/api/v1/devices/legacy-phone", nil))
	if get.Code != http.StatusOK || !strings.Contains(get.Body.String(), `"status":"repair_required"`) {
		t.Fatalf("legacy migration status=%d body=%s", get.Code, get.Body.String())
	}
	if strings.Contains(get.Body.String(), "token_issued_at") || strings.Contains(get.Body.String(), "token_expires_at") {
		t.Fatalf("legacy peer exposed meaningless zero token timestamps: %s", get.Body.String())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"version": 4`) || !strings.Contains(string(data), `"scheme": "http"`) {
		t.Fatalf("legacy registry was not migrated: %s", data)
	}
}

func TestRegistryV2MigratesPeersToExplicitLegacyHTTP(t *testing.T) {
	path := filepath.Join(t.TempDir(), "devices.json")
	legacy := `{"version":2,"peers":[{"id":"legacy-v2","name":"Legacy","status":"paired","token":"token","token_issued_at":"2026-01-01T00:00:00Z","token_expires_at":"2027-01-01T00:00:00Z"}]}`
	if err := os.WriteFile(path, []byte(legacy), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := New(config.DeviceConfig{ID: "windows-pc", Name: "Windows", RegistryPath: path}, config.SecurityConfig{}, config.DiscoveryConfig{}, 8899, nil, nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, `"version": 4`) || !strings.Contains(text, `"secure": false`) || !strings.Contains(text, `"scheme": "http"`) {
		t.Fatalf("v2 registry did not migrate as legacy HTTP: %s", text)
	}
}

func TestRegistryV3MigratesToProtectedV4AndRestarts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "devices.json")
	currentToken := "current-peer-token-secret"
	previousToken := "previous-peer-token-secret"
	legacy := `{"version":3,"peers":[{"id":"phone","name":"Phone","status":"paired","secure":false,"scheme":"http","token":"` + currentToken + `","token_issued_at":"2026-01-01T00:00:00Z","token_expires_at":"2099-01-01T00:00:00Z","previous_token":"` + previousToken + `","previous_token_expires_at":"2099-01-01T00:00:00Z"}]}`
	if err := os.WriteFile(path, []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	protector := registryTestProtector{}
	newProtected := func() (*Module, error) {
		return NewWithProtector(config.DeviceConfig{ID: "windows-pc", Name: "Windows", RegistryPath: path}, config.SecurityConfig{}, config.DiscoveryConfig{}, 8899, nil, nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), protector, true)
	}
	m, err := newProtected()
	if err != nil {
		t.Fatal(err)
	}
	if !m.ValidatePeerToken(currentToken) || !m.ValidatePeerToken(previousToken) {
		t.Fatal("migrated tokens were not available in memory")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if strings.Contains(text, currentToken) || strings.Contains(text, previousToken) || strings.Contains(text, `"token":`) || strings.Contains(text, `"previous_token":`) {
		t.Fatalf("protected registry contains plaintext token fields: %s", text)
	}
	if !strings.Contains(text, `"version": 4`) || !strings.Contains(text, `"credential_protection": "test-protector"`) || !strings.Contains(text, `"token_ciphertext"`) || !strings.Contains(text, `"previous_token_ciphertext"`) {
		t.Fatalf("registry was not upgraded to protected v4: %s", text)
	}
	reloaded, err := newProtected()
	if err != nil {
		t.Fatal(err)
	}
	if !reloaded.ValidatePeerToken(currentToken) || !reloaded.ValidatePeerToken(previousToken) {
		t.Fatal("protected tokens did not survive restart")
	}
}

func TestProtectedRegistryRotationPersistsTwoCiphertexts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "devices.json")
	security := config.SecurityConfig{PairingCode: "pair-me", PeerTokenTTL: time.Hour, TokenOverlapTTL: 5 * time.Minute}
	m, err := NewWithProtector(config.DeviceConfig{ID: "windows-pc", Name: "Windows", RegistryPath: path}, security, config.DiscoveryConfig{}, 8899, nil, nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), registryTestProtector{}, true)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	m.Routes(mux)
	pair := httptest.NewRecorder()
	mux.ServeHTTP(pair, httptest.NewRequest(http.MethodPost, "/api/v1/devices/pair", strings.NewReader(`{"code":"pair-me","id":"phone","name":"Phone"}`)))
	if pair.Code != http.StatusOK {
		t.Fatalf("pair failed: %s", pair.Body.String())
	}
	var initial struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(pair.Body.Bytes(), &initial); err != nil {
		t.Fatal(err)
	}
	rotate := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/devices/phone/token/rotate", nil)
	request.Header.Set("Authorization", "Bearer "+initial.Token)
	mux.ServeHTTP(rotate, request)
	if rotate.Code != http.StatusOK {
		t.Fatalf("rotate failed: %s", rotate.Body.String())
	}
	var rotated struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(rotate.Body.Bytes(), &rotated); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if strings.Contains(text, initial.Token) || strings.Contains(text, rotated.Token) || strings.Contains(text, `"token":`) || strings.Contains(text, `"previous_token":`) {
		t.Fatal("token rotation wrote plaintext to protected registry")
	}
	if strings.Count(text, `token_ciphertext`) != 2 {
		t.Fatalf("expected current and previous ciphertexts: %s", text)
	}
}

func TestRegistryMigrationProtectionFailurePreservesOriginal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "devices.json")
	legacy := []byte(`{"version":3,"peers":[{"id":"phone","name":"Phone","status":"paired","token":"plaintext-token","token_expires_at":"2099-01-01T00:00:00Z"}]}`)
	if err := os.WriteFile(path, legacy, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := NewWithProtector(config.DeviceConfig{ID: "windows-pc", RegistryPath: path}, config.SecurityConfig{}, config.DiscoveryConfig{}, 8899, nil, nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), registryTestProtector{failProtect: true}, true)
	if err == nil {
		t.Fatal("expected migration protection failure")
	}
	after, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(after, legacy) {
		t.Fatal("failed migration changed the original v3 registry")
	}
}

func TestDamagedProtectedRegistryFailsClosedWithoutRewrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "devices.json")
	bad := base64.StdEncoding.EncodeToString([]byte("damaged"))
	data := []byte(`{"version":4,"credential_protection":"test-protector","peers":[{"id":"phone","name":"Phone","status":"paired","token_ciphertext":"` + bad + `","token_expires_at":"2099-01-01T00:00:00Z"}]}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := NewWithProtector(config.DeviceConfig{ID: "windows-pc", RegistryPath: path}, config.SecurityConfig{}, config.DiscoveryConfig{}, 8899, nil, nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), registryTestProtector{}, true)
	if err == nil {
		t.Fatal("damaged protected token was accepted")
	}
	after, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(after, data) {
		t.Fatal("damaged registry was rewritten")
	}
}

func TestPairRejectsInvalidCode(t *testing.T) {
	m, err := New(config.DeviceConfig{ID: "windows-pc", Name: "Windows", RegistryPath: filepath.Join(t.TempDir(), "devices.json")}, config.SecurityConfig{PairingCode: "pair-me"}, config.DiscoveryConfig{}, 8899, nil, nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	m.Routes(mux)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/devices/pair", strings.NewReader(`{"code":"wrong","id":"iphone"}`)))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected invalid pairing code to return 401, got %d", recorder.Code)
	}
}

func TestPairingIsDisabledWithoutConfiguredCode(t *testing.T) {
	m, err := New(config.DeviceConfig{ID: "windows-pc", Name: "Windows", RegistryPath: filepath.Join(t.TempDir(), "devices.json")}, config.SecurityConfig{}, config.DiscoveryConfig{}, 8899, nil, nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	m.Routes(mux)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/devices/pair", strings.NewReader(`{"id":"iphone"}`)))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("empty pairing code unexpectedly enabled pairing: %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestParseDiscoveryAnnouncement(t *testing.T) {
	valid, ok := parseDiscoveryAnnouncement([]byte(`{"type":"localbridge.discovery.v1","protocol_version":1,"device_id":"phone","device_name":"Phone","api_port":8899,"nonce":"abc"}`))
	if !ok || valid.DeviceID != "phone" || valid.APIPort != 8899 {
		t.Fatalf("valid announcement was rejected: %#v", valid)
	}
	if _, ok := parseDiscoveryAnnouncement([]byte(`{"type":"other","protocol_version":1,"device_id":"phone","nonce":"abc"}`)); ok {
		t.Fatal("unexpectedly accepted invalid announcement type")
	}
	fingerprint := strings.Repeat("a", 64)
	secure, ok := parseDiscoveryAnnouncement([]byte(fmt.Sprintf(`{"type":"localbridge.discovery.v1","protocol_version":1,"device_id":"secure-phone","api_port":8899,"scheme":"https","secure":true,"certificate_sha256":%q,"nonce":"abc"}`, fingerprint)))
	if !ok || !secure.Secure || secure.Scheme != "https" || secure.CertificateSHA256 != fingerprint {
		t.Fatalf("secure discovery fields were not retained: %#v", secure)
	}
	if _, ok := parseDiscoveryAnnouncement([]byte(`{"type":"localbridge.discovery.v1","protocol_version":1,"device_id":"bad","api_port":8899,"scheme":"https","secure":true,"nonce":"abc"}`)); ok {
		t.Fatal("secure discovery without a fingerprint was accepted")
	}
}

func TestPairRequiresPinnedFingerprintForSecurePeer(t *testing.T) {
	if err := validatePairRequest(pairRequest{ID: "phone", Secure: true, Scheme: "https"}, "windows-pc"); err == nil {
		t.Fatal("secure pairing without a fingerprint was accepted")
	}
	fingerprint := strings.Repeat("b", 64)
	if err := validatePairRequest(pairRequest{ID: "phone", Secure: true, Scheme: "https", CertificateSHA256: fingerprint}, "windows-pc"); err != nil {
		t.Fatal(err)
	}
	if err := validatePairRequest(pairRequest{ID: "phone", Secure: false, Scheme: "https", CertificateSHA256: fingerprint}, "windows-pc"); err == nil {
		t.Fatal("insecure HTTPS pairing was accepted")
	}
}

func TestDiscoveryLifecycle(t *testing.T) {
	probe, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		t.Skipf("UDP unavailable: %v", err)
	}
	port := probe.LocalAddr().(*net.UDPAddr).Port
	_ = probe.Close()
	m, err := New(config.DeviceConfig{ID: "windows-pc", Name: "Windows", RegistryPath: filepath.Join(t.TempDir(), "devices.json")}, config.SecurityConfig{}, config.DiscoveryConfig{Enabled: true, Port: port, AnnounceInterval: time.Hour}, 8899, nil, nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := m.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if err := m.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	m.discoveryMu.Lock()
	defer m.discoveryMu.Unlock()
	if m.discoveryConn != nil || m.discoveryCancel != nil {
		t.Fatal("discovery resources were not released")
	}
}

func TestForwardLocalClipboardEvent(t *testing.T) {
	received := make(chan struct{}, 1)
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/clipboard" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err == nil && body["content"] == "from-windows" {
			received <- struct{}{}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"accepted": true})
	}))
	defer remote.Close()
	host, portText, _ := strings.Cut(strings.TrimPrefix(remote.URL, "http://"), ":")
	port, _ := strconv.Atoi(portText)
	bus := eventbus.New()
	store, err := syncstore.New(filepath.Join(t.TempDir(), "jobs.json"), 10, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	m, err := New(config.DeviceConfig{ID: "windows-pc", Name: "Windows", RegistryPath: filepath.Join(t.TempDir(), "devices.json")}, config.SecurityConfig{PairingCode: "pair-me"}, config.DiscoveryConfig{}, 8899, []string{"clipboard.text.push"}, bus, store, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	m.Routes(mux)
	pair := httptest.NewRecorder()
	pairRequest := httptest.NewRequest(http.MethodPost, "/api/v1/devices/pair", strings.NewReader(fmt.Sprintf(`{"code":"pair-me","id":"desktop-peer","name":"Peer","address":%q,"port":%d,"capabilities":["clipboard.text.push"]}`, host, port)))
	mux.ServeHTTP(pair, pairRequest)
	if pair.Code != http.StatusOK {
		t.Fatalf("pair failed: %d %s", pair.Code, pair.Body.String())
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := m.Start(ctx); err != nil {
		t.Fatal(err)
	}
	bus.Publish(eventbus.Event{Type: eventbus.ClipboardChanged, Data: map[string]any{"id": "item-1", "type": "text", "mime_type": "text/plain", "content": "from-windows", "hash": "hash-1", "device_id": "windows-pc", "source": "local"}})
	select {
	case <-received:
	case <-time.After(2 * time.Second):
		t.Fatal("local clipboard event was not forwarded")
	}
	deadline := time.After(2 * time.Second)
	for {
		jobs := store.List(10)
		if len(jobs) == 1 && jobs[0].State == syncstore.StateDelivered {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("unexpected forwarding job state: %#v", jobs)
		case <-time.After(10 * time.Millisecond):
		}
	}
	if err := m.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
}
