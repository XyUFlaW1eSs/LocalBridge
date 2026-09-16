package settings

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/XyUFlaW1eSs/LocalBridge/internal/config"
)

func TestSettingsPersistenceAndReset(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	store, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	value := Defaults()
	value.AutoStart = true
	listenerCalls := 0
	store.AddListener(func(updated Settings) {
		listenerCalls++
		if !updated.AutoStart {
			t.Errorf("listener received stale settings: %#v", updated)
		}
	})
	if _, err := store.Set(value); err != nil {
		t.Fatal(err)
	}
	if listenerCalls != 1 {
		t.Fatalf("settings listener calls=%d", listenerCalls)
	}
	reloaded, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reloaded.Get().AutoStart {
		t.Fatal("settings did not persist")
	}
	reset, err := reloaded.Reset()
	if err != nil {
		t.Fatal(err)
	}
	if reset.AutoStart || reset.Version != version {
		t.Fatalf("unexpected defaults: %#v", reset)
	}
}

func TestSettingsHTTPAndRemoteBoundary(t *testing.T) {
	module, err := New(config.SettingsConfig{StorePath: filepath.Join(t.TempDir(), "settings.json")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	module.Routes(mux)
	remote := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
	remote.RemoteAddr = "192.168.1.9:1234"
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, remote)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("remote settings status=%d", recorder.Code)
	}
	body, _ := json.Marshal(Settings{Version: version, AutoStart: true})
	request := httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewReader(body))
	request.RemoteAddr = "127.0.0.1:1234"
	request.Header.Set("Content-Type", "application/json")
	recorder = httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("local settings update status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
