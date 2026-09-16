package transport

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
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
	if _, err := endpointURL(Endpoint{Address: "127.0.0.1", Port: 80, Secure: true, Scheme: "https"}, "/health"); err == nil {
		t.Fatal("expected secure endpoint without a fingerprint to be rejected")
	}
	if _, err := endpointURL(Endpoint{Address: "127.0.0.1", Port: 80, Secure: true, Scheme: "http", CertificateSHA256: strings.Repeat("0", 64)}, "/health"); err == nil {
		t.Fatal("expected secure endpoint with an HTTP scheme to be rejected")
	}
}

func TestPinnedHTTPSAndFingerprintFailure(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secure-token" {
			t.Fatal("pinned request did not carry its bearer token")
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "secure"})
	}))
	defer server.Close()
	host, portText, _ := strings.Cut(strings.TrimPrefix(server.URL, "https://"), ":")
	port, _ := strconv.Atoi(portText)
	digest := sha256.Sum256(server.Certificate().Raw)
	endpoint := Endpoint{Address: host, Port: port, Token: "secure-token", Secure: true, Scheme: "https", CertificateSHA256: fmt.Sprintf("%x", digest)}
	client := NewClient(0)
	var response map[string]string
	if err := client.GetJSON(context.Background(), endpoint, "/api/v1/system/capabilities", &response); err != nil {
		t.Fatal(err)
	}
	if response["status"] != "secure" {
		t.Fatalf("unexpected HTTPS response: %#v", response)
	}
	endpoint.CertificateSHA256 = strings.Repeat("0", 64)
	if err := client.GetJSON(context.Background(), endpoint, "/api/v1/system/capabilities", nil); err == nil || !strings.Contains(err.Error(), "fingerprint") {
		t.Fatalf("wrong certificate fingerprint was accepted: %v", err)
	}
}

func TestRedirectIsNeverFollowed(t *testing.T) {
	visited := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "http://example.invalid/leak", http.StatusFound)
			return
		}
		visited = true
	}))
	defer server.Close()
	host, portText, _ := strings.Cut(strings.TrimPrefix(server.URL, "http://"), ":")
	port, _ := strconv.Atoi(portText)
	err := NewClient(0).GetJSON(context.Background(), Endpoint{Address: host, Port: port}, "/redirect", nil)
	if err == nil || !strings.Contains(err.Error(), ErrRedirect.Error()) {
		t.Fatalf("redirect did not fail closed: %v", err)
	}
	if visited {
		t.Fatal("redirect target was contacted")
	}
}
