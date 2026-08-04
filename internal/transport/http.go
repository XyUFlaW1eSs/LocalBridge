package transport

import (
	"context"
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
	Address string
	Port    int
	Token   string
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
	return &Client{http: &http.Client{Timeout: timeout}}
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
	response, err := c.http.Do(request)
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
	base := "http://" + net.JoinHostPort(address, strconv.Itoa(endpoint.Port))
	parsed, err := url.Parse(base + path)
	if err != nil || parsed.Scheme != "http" || parsed.Host == "" {
		return "", errors.New("transport endpoint URL is invalid")
	}
	return parsed.String(), nil
}
