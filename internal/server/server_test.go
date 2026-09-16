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

func TestAuthenticatedServerAllowsLoopbackGUI(t *testing.T) {
	s := New("127.0.0.1:0", 0, 0, 0, slog.New(slog.NewTextHandler(io.Discard, nil)), func(mux *http.ServeMux) {
		mux.HandleFunc("GET /app/", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	})
	s.SetAuthToken("0123456789abcdef")
	local := httptest.NewRequest(http.MethodGet, "/app/", nil)
	local.RemoteAddr = "127.0.0.1:1234"
	localRecorder := httptest.NewRecorder()
	s.http.Handler.ServeHTTP(localRecorder, local)
	if localRecorder.Code != http.StatusOK {
		t.Fatalf("loopback GUI should bypass management auth, got %d: %s", localRecorder.Code, localRecorder.Body.String())
	}
	remote := httptest.NewRequest(http.MethodGet, "/app/", nil)
	remote.RemoteAddr = "192.168.1.9:1234"
	remoteRecorder := httptest.NewRecorder()
	s.http.Handler.ServeHTTP(remoteRecorder, remote)
	if remoteRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("remote GUI should remain protected, got %d: %s", remoteRecorder.Code, remoteRecorder.Body.String())
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

func TestEffectiveConfigDiagnosticsAccess(t *testing.T) {
	s := New("127.0.0.1:0", 0, 0, 0, slog.New(slog.NewTextHandler(io.Discard, nil)), nil)
	s.SetRuntimeConfig(map[string]any{"schema_version": 1, "effective": map[string]any{"security": map[string]any{"bearer_token_configured": false}}})

	remote := httptest.NewRequest(http.MethodGet, "/api/v1/system/config", nil)
	remote.RemoteAddr = "192.168.1.20:1234"
	remoteRecorder := httptest.NewRecorder()
	s.http.Handler.ServeHTTP(remoteRecorder, remote)
	if remoteRecorder.Code != http.StatusForbidden {
		t.Fatalf("unauthenticated remote diagnostics should be forbidden, got %d: %s", remoteRecorder.Code, remoteRecorder.Body.String())
	}

	local := httptest.NewRequest(http.MethodGet, "/api/v1/system/config", nil)
	local.RemoteAddr = "127.0.0.1:1234"
	localRecorder := httptest.NewRecorder()
	s.http.Handler.ServeHTTP(localRecorder, local)
	if localRecorder.Code != http.StatusOK || !strings.Contains(localRecorder.Body.String(), `"schema_version":1`) {
		t.Fatalf("loopback diagnostics failed: %d %s", localRecorder.Code, localRecorder.Body.String())
	}
}

func TestEffectiveConfigDiagnosticsRequireManagementToken(t *testing.T) {
	s := New("127.0.0.1:0", 0, 0, 0, slog.New(slog.NewTextHandler(io.Discard, nil)), nil)
	s.SetAuthToken("0123456789abcdef")
	s.SetPeerTokenValidator(func(token string) bool { return token == "peer-token" })
	s.SetRuntimeConfig(map[string]any{"schema_version": 1})

	localWithoutToken := httptest.NewRequest(http.MethodGet, "/api/v1/system/config", nil)
	localWithoutToken.RemoteAddr = "127.0.0.1:1234"
	localWithoutTokenRecorder := httptest.NewRecorder()
	s.http.Handler.ServeHTTP(localWithoutTokenRecorder, localWithoutToken)
	if localWithoutTokenRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("authenticated deployment allowed tokenless loopback diagnostics: %d %s", localWithoutTokenRecorder.Code, localWithoutTokenRecorder.Body.String())
	}

	peer := httptest.NewRequest(http.MethodGet, "/api/v1/system/config", nil)
	peer.RemoteAddr = "192.168.1.20:1234"
	peer.Header.Set("Authorization", "Bearer peer-token")
	peerRecorder := httptest.NewRecorder()
	s.http.Handler.ServeHTTP(peerRecorder, peer)
	if peerRecorder.Code != http.StatusForbidden {
		t.Fatalf("peer token read management diagnostics: %d %s", peerRecorder.Code, peerRecorder.Body.String())
	}

	management := httptest.NewRequest(http.MethodGet, "/api/v1/system/config", nil)
	management.RemoteAddr = "192.168.1.20:1234"
	management.Header.Set("Authorization", "Bearer 0123456789abcdef")
	managementRecorder := httptest.NewRecorder()
	s.http.Handler.ServeHTTP(managementRecorder, management)
	if managementRecorder.Code != http.StatusOK {
		t.Fatalf("management token could not read diagnostics: %d %s", managementRecorder.Code, managementRecorder.Body.String())
	}
}
