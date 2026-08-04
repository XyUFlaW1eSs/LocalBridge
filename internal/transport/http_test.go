package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestClientJSONAndToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer peer-token" {
			http.Error(w, `{"error":"missing token"}`, http.StatusUnauthorized)
			return
		}
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/test" {
			http.NotFound(w, r)
			return
		}
		var request map[string]string
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request["value"] != "hello" {
			http.Error(w, `{"error":"bad body"}`, http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()
	host, portText, _ := strings.Cut(strings.TrimPrefix(server.URL, "http://"), ":")
	port, _ := strconv.Atoi(portText)
	client := NewClient(0)
	var response map[string]string
	if err := client.PostJSON(context.Background(), Endpoint{Address: host, Port: port, Token: "peer-token"}, "/api/v1/test", map[string]string{"value": "hello"}, &response); err != nil {
		t.Fatal(err)
	}
	if response["status"] != "ok" {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestEndpointValidation(t *testing.T) {
	if _, err := endpointURL(Endpoint{Address: "http://localhost", Port: 80}, "/health"); err == nil {
		t.Fatal("expected endpoint URL to reject a scheme")
	}
	if _, err := endpointURL(Endpoint{Address: "127.0.0.1", Port: 0}, "/health"); err == nil {
		t.Fatal("expected endpoint URL to reject invalid port")
	}
}
