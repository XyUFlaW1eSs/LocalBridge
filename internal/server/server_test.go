package server

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthEndpoint(t *testing.T) {
	s := New("127.0.0.1:0", 0, 0, 0, slog.New(slog.NewTextHandler(io.Discard, nil)), nil)
	recorder := httptest.NewRecorder()
	s.http.Handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/system/health", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"status":"ok"`) {
		t.Fatalf("unexpected response: %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestCapabilitiesEndpoint(t *testing.T) {
	s := New("127.0.0.1:0", 0, 0, 0, slog.New(slog.NewTextHandler(io.Discard, nil)), nil)
	s.SetRuntimeInfo(RuntimeInfo{Version: "test", DeviceID: "test-pc", DeviceName: "Test PC", Capabilities: []string{"clipboard.text.push"}})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/system/capabilities", nil)
	s.http.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || recorder.Header().Get("X-Request-ID") == "" {
		t.Fatalf("unexpected capabilities response: %d headers=%v", recorder.Code, recorder.Header())
	}
	var body struct {
		Version string `json:"version"`
		Device  struct {
			ID string `json:"id"`
		} `json:"device"`
		Capabilities []string `json:"capabilities"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Version != "test" || body.Device.ID != "test-pc" || len(body.Capabilities) != 1 {
		t.Fatalf("unexpected capabilities body: %s", recorder.Body.String())
	}
}

func TestAuthentication(t *testing.T) {
	s := New("127.0.0.1:0", 0, 0, 0, slog.New(slog.NewTextHandler(io.Discard, nil)), nil)
	s.SetAuthToken("0123456789abcdef")
	s.SetPeerTokenValidator(func(token string) bool { return token == "peer-token" })

	unauthorized := httptest.NewRecorder()
	s.http.Handler.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/api/v1/system/capabilities", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", unauthorized.Code, unauthorized.Body.String())
	}

	health := httptest.NewRecorder()
	s.http.Handler.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/api/v1/system/health", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("health should remain available for diagnostics, got %d", health.Code)
	}

	authorized := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/system/capabilities", nil)
	request.Header.Set("Authorization", "Bearer 0123456789abcdef")
	s.http.Handler.ServeHTTP(authorized, request)
	if authorized.Code != http.StatusOK {
		t.Fatalf("expected authenticated request to pass, got %d: %s", authorized.Code, authorized.Body.String())
	}

	peerAuthorized := httptest.NewRecorder()
	peerRequest := httptest.NewRequest(http.MethodGet, "/api/v1/system/capabilities", nil)
	peerRequest.Header.Set("Authorization", "Bearer peer-token")
	s.http.Handler.ServeHTTP(peerAuthorized, peerRequest)
	if peerAuthorized.Code != http.StatusOK {
		t.Fatalf("expected peer token request to pass, got %d: %s", peerAuthorized.Code, peerAuthorized.Body.String())
	}
}

func TestRequestIDValidation(t *testing.T) {
	s := New("127.0.0.1:0", 0, 0, 0, slog.New(slog.NewTextHandler(io.Discard, nil)), nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/system/health", nil)
	request.Header.Set("X-Request-ID", "bad value\r\n")
	s.http.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || !validRequestID(recorder.Header().Get("X-Request-ID")) || recorder.Header().Get("X-Request-ID") == "bad value\r\n" {
		t.Fatalf("request ID was not sanitized: %q", recorder.Header().Get("X-Request-ID"))
	}
}

func TestPublicPathPrefixStillRequiresCapabilityHandler(t *testing.T) {
	s := New("127.0.0.1:0", 0, 0, 0, slog.New(slog.NewTextHandler(io.Discard, nil)), nil)
	s.SetAuthToken("0123456789abcdef")
	s.SetPublicPathPrefixes("/share/", "/receive/")
	public := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/share/random-capability", nil)
	s.http.Handler.ServeHTTP(public, request)
	if public.Code != http.StatusNotFound {
		t.Fatalf("public capability path should reach route handler, got auth status %d", public.Code)
	}
	protected := httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/api/v1/system/capabilities", nil)
	s.http.Handler.ServeHTTP(protected, request)
	if protected.Code != http.StatusUnauthorized {
		t.Fatalf("management path should remain protected, got %d", protected.Code)
	}
}

func TestAuthenticatedContextMarker(t *testing.T) {
	s := New("127.0.0.1:0", 0, 0, 0, slog.New(slog.NewTextHandler(io.Discard, nil)), func(mux *http.ServeMux) {
		mux.HandleFunc("GET /api/v1/test-auth-context", func(w http.ResponseWriter, r *http.Request) {
			if !Authenticated(r) {
				writeError(w, http.StatusForbidden, "missing auth marker", requestIDFrom(r))
				return
			}
			w.WriteHeader(http.StatusNoContent)
		})
	})
	s.SetAuthToken("0123456789abcdef")
	request := httptest.NewRequest(http.MethodGet, "/api/v1/test-auth-context", nil)
	request.Header.Set("Authorization", "Bearer 0123456789abcdef")
	recorder := httptest.NewRecorder()
	s.http.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("authenticated context marker missing, got %d: %s", recorder.Code, recorder.Body.String())
	}
}
