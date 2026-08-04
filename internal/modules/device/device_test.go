package device

import (
	"context"
	"encoding/json"
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
)

func TestPairListAndRevoke(t *testing.T) {
	path := filepath.Join(t.TempDir(), "devices.json")
	m, err := New(config.DeviceConfig{ID: "windows-pc", Name: "Windows", RegistryPath: path}, config.SecurityConfig{PairingCode: "pair-me"}, config.DiscoveryConfig{}, 8899, nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
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
	reloaded, err := New(config.DeviceConfig{ID: "windows-pc", Name: "Windows", RegistryPath: path}, config.SecurityConfig{PairingCode: "pair-me"}, config.DiscoveryConfig{}, 8899, nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	reloadedMux := http.NewServeMux()
	reloaded.Routes(reloadedMux)
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
}

func TestPairRejectsInvalidCode(t *testing.T) {
	m, err := New(config.DeviceConfig{ID: "windows-pc", Name: "Windows", RegistryPath: filepath.Join(t.TempDir(), "devices.json")}, config.SecurityConfig{PairingCode: "pair-me"}, config.DiscoveryConfig{}, 8899, nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
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

func TestParseDiscoveryAnnouncement(t *testing.T) {
	valid, ok := parseDiscoveryAnnouncement([]byte(`{"type":"localbridge.discovery.v1","protocol_version":1,"device_id":"phone","device_name":"Phone","api_port":8899,"nonce":"abc"}`))
	if !ok || valid.DeviceID != "phone" || valid.APIPort != 8899 {
		t.Fatalf("valid announcement was rejected: %#v", valid)
	}
	if _, ok := parseDiscoveryAnnouncement([]byte(`{"type":"other","protocol_version":1,"device_id":"phone","nonce":"abc"}`)); ok {
		t.Fatal("unexpectedly accepted invalid announcement type")
	}
}

func TestDiscoveryLifecycle(t *testing.T) {
	probe, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		t.Skipf("UDP unavailable: %v", err)
	}
	port := probe.LocalAddr().(*net.UDPAddr).Port
	_ = probe.Close()
	m, err := New(config.DeviceConfig{ID: "windows-pc", Name: "Windows", RegistryPath: filepath.Join(t.TempDir(), "devices.json")}, config.SecurityConfig{}, config.DiscoveryConfig{Enabled: true, Port: port, AnnounceInterval: time.Hour}, 8899, nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
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
	m, err := New(config.DeviceConfig{ID: "windows-pc", Name: "Windows", RegistryPath: filepath.Join(t.TempDir(), "devices.json")}, config.SecurityConfig{PairingCode: "pair-me"}, config.DiscoveryConfig{}, 8899, []string{"clipboard.text.push"}, bus, slog.New(slog.NewTextHandler(io.Discard, nil)))
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
	if err := m.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
}
