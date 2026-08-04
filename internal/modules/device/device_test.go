package device

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/XyUFlaW1eSs/LocalBridge/internal/config"
)

func TestPairListAndRevoke(t *testing.T) {
	path := filepath.Join(t.TempDir(), "devices.json")
	m, err := New(config.DeviceConfig{ID: "windows-pc", Name: "Windows", RegistryPath: path}, config.SecurityConfig{PairingCode: "pair-me"}, slog.New(slog.NewTextHandler(io.Discard, nil)))
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
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	reloaded, err := New(config.DeviceConfig{ID: "windows-pc", Name: "Windows", RegistryPath: path}, config.SecurityConfig{PairingCode: "pair-me"}, slog.New(slog.NewTextHandler(io.Discard, nil)))
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
	m, err := New(config.DeviceConfig{ID: "windows-pc", Name: "Windows", RegistryPath: filepath.Join(t.TempDir(), "devices.json")}, config.SecurityConfig{PairingCode: "pair-me"}, slog.New(slog.NewTextHandler(io.Discard, nil)))
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
