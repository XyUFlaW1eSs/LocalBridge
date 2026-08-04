package sync

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/XyUFlaW1eSs/LocalBridge/internal/syncstore"
)

func TestJobEndpoints(t *testing.T) {
	store, err := syncstore.New(filepath.Join(t.TempDir(), "jobs.json"), 10, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Create(syncstore.Envelope{ID: "job-1", Kind: "clipboard.push", Type: "text", MIMEType: "text/plain", Payload: []byte(`{"content":"hello"}`)}); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	New(store, slog.New(slog.NewTextHandler(io.Discard, nil))).Routes(mux)
	list := httptest.NewRecorder()
	mux.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/v1/sync/jobs", nil))
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), "job-1") {
		t.Fatalf("unexpected job list: %d %s", list.Code, list.Body.String())
	}
}
