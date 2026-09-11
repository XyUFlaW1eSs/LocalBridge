package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEmbeddedGUIResources(t *testing.T) {
	mux := http.NewServeMux()
	Routes(mux)
	server := httptest.NewServer(mux)
	defer server.Close()
	for _, asset := range []string{"/", "/app/", "/app/app.js", "/app/styles.css"} {
		response, err := http.Get(server.URL + asset)
		if err != nil {
			t.Fatal(err)
		}
		data, _ := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if response.StatusCode != http.StatusOK || len(data) == 0 {
			t.Fatalf("asset %s status=%d bytes=%d", asset, response.StatusCode, len(data))
		}
	}
	response, err := http.Get(server.URL + "/app/../internal/config/config.go")
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("asset traversal status=%d", response.StatusCode)
	}
	_ = response.Body.Close()
	response, err = http.Get(server.URL + "/app/app.js")
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if !strings.Contains(string(data), "文件分享") || strings.Contains(string(data), "cdn") {
		t.Fatal("embedded GUI content is missing or uses an external CDN")
	}
}
