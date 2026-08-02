package server

import (
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
