package transport

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const maxResponseBytes = 4 * 1024 * 1024

type Endpoint struct {
	Address           string
	Port              int
	Token             string
	Secure            bool
	Scheme            string
	CertificateSHA256 string
	Fingerprint       string
}

type Client struct {
	http *http.Client
}

type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("remote HTTP status %d", e.StatusCode)
	}
	return fmt.Sprintf("remote HTTP status %d: %s", e.StatusCode, e.Message)
}

func NewClient(timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &Client{http: &http.Client{Timeout: timeout, CheckRedirect: rejectRedirect}}
}

var ErrRedirect = errors.New("transport redirects are not allowed")

func rejectRedirect(*http.Request, []*http.Request) error { return ErrRedirect }

// HTTPClient returns an HTTP client scoped to one endpoint. Secure endpoints
// use exact leaf-certificate pinning and do not rely on a broad trust bypass.
func (c *Client) HTTPClient(endpoint Endpoint) (*http.Client, error) {
	if c == nil || c.http == nil {
		return nil, errors.New("transport client is not initialized")
	}
	if _, err := endpointURL(endpoint, "/"); err != nil {
		return nil, err
	}
	if !endpoint.Secure {
		return c.http, nil
	}
	base, ok := c.http.Transport.(*http.Transport)
	if !ok || base == nil {
		base = http.DefaultTransport.(*http.Transport)
	}
	fingerprint, err := endpointFingerprint(endpoint)
	if err != nil {
		return nil, err
	}
	transport := base.Clone()
	transport.TLSClientConfig = &tls.Config{
		MinVersion:         tls.VersionTLS12,
		MaxVersion:         tls.VersionTLS13,
		InsecureSkipVerify: true, // VerifyConnection below is the complete trust decision.
		VerifyConnection: func(state tls.ConnectionState) error {
			if len(state.PeerCertificates) == 0 {
				return errors.New("HTTPS peer did not present a certificate")
			}
			digest := sha256.Sum256(state.PeerCertificates[0].Raw)
			if hex.EncodeToString(digest[:]) != fingerprint {
				return fmt.Errorf("HTTPS peer certificate fingerprint mismatch")
			}
			return nil
		},
	}
	return &http.Client{Timeout: c.http.Timeout, Transport: transport, CheckRedirect: rejectRedirect}, nil
}

func (c *Client) DoJSON(ctx context.Context, method string, endpoint Endpoint, path string, requestBody any, responseBody any) error {
	if c == nil || c.http == nil {
		return errors.New("transport client is not initialized")
	}
	target, err := endpointURL(endpoint, path)
	if err != nil {
		return err
	}
	var body io.Reader
	if requestBody != nil {
		data, err := json.Marshal(requestBody)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		body = strings.NewReader(string(data))
	}
	request, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	if requestBody != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if endpoint.Token != "" {
		request.Header.Set("Authorization", "Bearer "+endpoint.Token)
	}
	request.Header.Set("Accept", "application/json")
	client, err := c.HTTPClient(endpoint)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if len(data) > maxResponseBytes {
		return errors.New("remote response exceeds configured limit")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var errorBody struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(data, &errorBody)
		return &HTTPError{StatusCode: response.StatusCode, Message: errorBody.Error}
	}
	if responseBody == nil || len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, responseBody); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func (c *Client) GetJSON(ctx context.Context, endpoint Endpoint, path string, responseBody any) error {
	return c.DoJSON(ctx, http.MethodGet, endpoint, path, nil, responseBody)
}

func (c *Client) PostJSON(ctx context.Context, endpoint Endpoint, path string, requestBody, responseBody any) error {
	return c.DoJSON(ctx, http.MethodPost, endpoint, path, requestBody, responseBody)
}

func endpointURL(endpoint Endpoint, path string) (string, error) {
	address := strings.TrimSpace(endpoint.Address)
	if address == "" || strings.ContainsAny(address, "/?#") {
		return "", errors.New("transport endpoint address is invalid")
	}
	if endpoint.Port < 1 || endpoint.Port > 65535 {
		return "", fmt.Errorf("transport endpoint port is invalid: %d", endpoint.Port)
	}
	if path == "" || !strings.HasPrefix(path, "/") || strings.ContainsAny(path, "\r\n") {
		return "", errors.New("transport path is invalid")
	}
	if strings.Contains(address, ":") && net.ParseIP(address) == nil {
		if _, _, err := net.SplitHostPort(address); err == nil {
			return "", errors.New("transport endpoint must not include a port in address")
		}
	}
	scheme := strings.ToLower(strings.TrimSpace(endpoint.Scheme))
	if scheme == "" {
		if endpoint.Secure {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	if scheme != "http" && scheme != "https" {
		return "", errors.New("transport endpoint scheme is invalid")
	}
	if endpoint.Secure != (scheme == "https") {
		return "", errors.New("transport endpoint secure flag and scheme disagree")
	}
	if endpoint.Secure {
		if _, err := endpointFingerprint(endpoint); err != nil {
			return "", err
		}
	}
	if !endpoint.Secure && (strings.TrimSpace(endpoint.CertificateSHA256) != "" || strings.TrimSpace(endpoint.Fingerprint) != "") {
		return "", errors.New("HTTP transport endpoint must not include a certificate fingerprint")
	}
	if endpoint.Secure && !validFingerprint(endpointFingerprintValue(endpoint)) {
		return "", errors.New("secure transport endpoint requires a lowercase 64-character SHA-256 certificate fingerprint")
	}
	base := scheme + "://" + net.JoinHostPort(address, strconv.Itoa(endpoint.Port))
	parsed, err := url.Parse(base + path)
	if err != nil || parsed.Scheme != scheme || parsed.Host == "" {
		return "", errors.New("transport endpoint URL is invalid")
	}
	return parsed.String(), nil
}

func endpointFingerprintValue(endpoint Endpoint) string {
	if strings.TrimSpace(endpoint.CertificateSHA256) != "" {
		return strings.TrimSpace(endpoint.CertificateSHA256)
	}
	return strings.TrimSpace(endpoint.Fingerprint)
}

func endpointFingerprint(endpoint Endpoint) (string, error) {
	certificateFingerprint := strings.TrimSpace(endpoint.CertificateSHA256)
	legacyFingerprint := strings.TrimSpace(endpoint.Fingerprint)
	if certificateFingerprint != "" && legacyFingerprint != "" && certificateFingerprint != legacyFingerprint {
		return "", errors.New("transport endpoint certificate fingerprints disagree")
	}
	fingerprint := endpointFingerprintValue(endpoint)
	if !validFingerprint(fingerprint) {
		return "", errors.New("secure transport endpoint requires a lowercase 64-character SHA-256 certificate fingerprint")
	}
	return fingerprint, nil
}

func validFingerprint(value string) bool {
	if len(value) != sha256.Size*2 || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
