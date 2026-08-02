package clipboard

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/XyUFlaW1eSs/LocalBridge/internal/config"
	"github.com/XyUFlaW1eSs/LocalBridge/internal/eventbus"
)

type fakePlatform struct {
	writes []string
}

func (f *fakePlatform) ReadText() (string, error) { return "", nil }
func (f *fakePlatform) WriteText(value string) error {
	f.writes = append(f.writes, value)
	return nil
}
func (f *fakePlatform) Watch(context.Context, time.Duration, func(string), func(error)) error {
	return nil
}

func testModule() (*Module, *fakePlatform) {
	fake := &fakePlatform{}
	m := New(config.ClipboardConfig{Enabled: true, MaxTextBytes: 1024, WatchInterval: time.Millisecond}, config.DeviceConfig{ID: "pc"}, eventbus.New(), slog.Default())
	m.platform = fake
	return m, fake
}

func TestPushLatestAndDeduplication(t *testing.T) {
	m, fake := testModule()
	mux := http.NewServeMux()
	m.Routes(mux)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/clipboard", strings.NewReader(`{"content":"hello","device_id":"iphone"}`))
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("push status = %d, body = %s", response.Code, response.Body.String())
	}
	var pushed pushResponse
	if err := json.Unmarshal(response.Body.Bytes(), &pushed); err != nil || !pushed.Accepted {
		t.Fatalf("unexpected push response: %#v, err=%v", pushed, err)
	}
	if len(fake.writes) != 1 || fake.writes[0] != "hello" {
		t.Fatalf("expected one platform write, got %#v", fake.writes)
	}

	request = httptest.NewRequest(http.MethodPost, "/api/v1/clipboard", strings.NewReader(`{"content":"hello"}`))
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	var duplicate pushResponse
	if err := json.Unmarshal(response.Body.Bytes(), &duplicate); err != nil || duplicate.Accepted {
		t.Fatalf("expected duplicate response, got %#v, err=%v", duplicate, err)
	}
	if len(fake.writes) != 1 {
		t.Fatalf("duplicate should not write platform clipboard: %#v", fake.writes)
	}

	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/clipboard/latest", nil))
	if response.Code != http.StatusOK || !contains(response.Body.String(), `"content":"hello"`) {
		t.Fatalf("unexpected latest response: %d %s", response.Code, response.Body.String())
	}
}

func TestPushRejectsOversize(t *testing.T) {
	m, _ := testModule()
	m.cfg.MaxTextBytes = 3
	mux := http.NewServeMux()
	m.Routes(mux)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/clipboard", strings.NewReader(`{"content":"long"}`)))
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestPushAcceptsPlainTextBody(t *testing.T) {
	m, fake := testModule()
	mux := http.NewServeMux()
	m.Routes(mux)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/clipboard", strings.NewReader("plain shortcut content"))
	request.Header.Set("Content-Type", "text/plain; charset=utf-8")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("plain text push status = %d, body = %s", response.Code, response.Body.String())
	}
	if len(fake.writes) != 1 || fake.writes[0] != "plain shortcut content" {
		t.Fatalf("unexpected plain text write: %#v", fake.writes)
	}
}

func contains(value, fragment string) bool {
	for i := 0; i+len(fragment) <= len(value); i++ {
		if value[i:i+len(fragment)] == fragment {
			return true
		}
	}
	return false
}
