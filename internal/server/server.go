package server

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

type Server struct {
	logger             *slog.Logger
	http               *http.Server
	authToken          string
	peerTokenValidator func(string) bool
	publicPathPrefixes []string
	runtimeInfo        RuntimeInfo
	runtimeConfig      any
	tlsEnabled         bool
	tlsFingerprint     string
}

type TLSConfig struct {
	Enabled     bool
	Certificate tls.Certificate
	Fingerprint string
}

// LoadTLSCertificate loads and validates the configured certificate/key pair.
// The fingerprint is the lowercase SHA-256 of the leaf certificate DER.
func LoadTLSCertificate(certFile, keyFile string) (TLSConfig, error) {
	certFile = strings.TrimSpace(certFile)
	keyFile = strings.TrimSpace(keyFile)
	if certFile == "" || keyFile == "" {
		return TLSConfig{}, errors.New("TLS certificate and private key are both required")
	}
	if _, err := os.Stat(certFile); err != nil {
		return TLSConfig{}, fmt.Errorf("TLS certificate file: %w", err)
	}
	if _, err := os.Stat(keyFile); err != nil {
		return TLSConfig{}, fmt.Errorf("TLS private key file: %w", err)
	}
	certificate, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return TLSConfig{}, fmt.Errorf("load TLS certificate and private key: %w", err)
	}
	if len(certificate.Certificate) == 0 {
		return TLSConfig{}, errors.New("TLS certificate chain is empty")
	}
	digest := sha256.Sum256(certificate.Certificate[0])
	return TLSConfig{Enabled: true, Certificate: certificate, Fingerprint: hex.EncodeToString(digest[:])}, nil
}

type RuntimeInfo struct {
	Version      string
	DeviceID     string
	DeviceName   string
	Capabilities []string
}

func New(address string, readTimeout, writeTimeout, idleTimeout time.Duration, logger *slog.Logger, routes func(*http.ServeMux)) *Server {
	return NewWithTLS(address, readTimeout, writeTimeout, idleTimeout, logger, routes, TLSConfig{})
}

func NewWithTLS(address string, readTimeout, writeTimeout, idleTimeout time.Duration, logger *slog.Logger, routes func(*http.ServeMux), tlsConfig TLSConfig) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	s := &Server{logger: logger, runtimeInfo: RuntimeInfo{Version: "dev"}, tlsEnabled: tlsConfig.Enabled, tlsFingerprint: tlsConfig.Fingerprint}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/system/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "service": "localbridge", "time": time.Now().UTC(), "request_id": requestIDFrom(r)})
	})
	mux.HandleFunc("GET /api/v1/system/capabilities", func(w http.ResponseWriter, r *http.Request) {
		info := s.runtimeInfo
		capabilities := append([]string(nil), info.Capabilities...)
		writeJSON(w, http.StatusOK, map[string]any{
			"service":          "localbridge",
			"version":          info.Version,
			"api_version":      "v1",
			"protocol_version": 1,
			"request_id":       requestIDFrom(r),
			"device": map[string]string{
				"id":   info.DeviceID,
				"name": info.DeviceName,
			},
			"transport": map[string]string{
				"scheme":             s.Scheme(),
				"certificate_sha256": s.CertificateFingerprint(),
				"fingerprint":        s.CertificateFingerprint(),
			},
			"scheme":             s.Scheme(),
			"certificate_sha256": s.CertificateFingerprint(),
			"fingerprint":        s.CertificateFingerprint(),
			"capabilities":       capabilities,
		})
	})
	mux.HandleFunc("GET /api/v1/system/config", func(w http.ResponseWriter, r *http.Request) {
		if !requestIsLoopback(r) && !ManagementAuthenticated(r) {
			writeError(w, http.StatusForbidden, "effective configuration diagnostics require local or management access", requestIDFrom(r))
			return
		}
		if s.runtimeConfig == nil {
			writeError(w, http.StatusServiceUnavailable, "effective configuration diagnostics are unavailable", requestIDFrom(r))
			return
		}
		writeJSON(w, http.StatusOK, s.runtimeConfig)
	})
	if routes != nil {
		routes(mux)
	}
	handler := requestIDMiddleware(requestLogging(logger, authentication(s, mux)))
	serverConfig := &http.Server{Addr: address, Handler: handler, ReadTimeout: readTimeout, WriteTimeout: writeTimeout, IdleTimeout: idleTimeout}
	if tlsConfig.Enabled {
		serverConfig.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12, MaxVersion: tls.VersionTLS13, Certificates: []tls.Certificate{tlsConfig.Certificate}}
	}
	s.http = serverConfig
	return s
}

func (s *Server) Scheme() string {
	if s != nil && s.tlsEnabled {
		return "https"
	}
	return "http"
}

func (s *Server) CertificateFingerprint() string {
	if s == nil || !s.tlsEnabled {
		return ""
	}
	return s.tlsFingerprint
}

func (s *Server) SetAuthToken(token string) { s.authToken = strings.TrimSpace(token) }

func (s *Server) SetPeerTokenValidator(validator func(string) bool) { s.peerTokenValidator = validator }

// SetPublicPathPrefixes allows capability URLs to work when management
// authentication is enabled. Every handler under these prefixes must perform
// its own bearer-like capability-token validation.
func (s *Server) SetPublicPathPrefixes(prefixes ...string) {
	s.publicPathPrefixes = append([]string(nil), prefixes...)
}

func (s *Server) SetRuntimeInfo(info RuntimeInfo) { s.runtimeInfo = info }

func (s *Server) SetRuntimeConfig(value any) { s.runtimeConfig = value }

func (s *Server) Start() error {
	s.logger.Info("API server started", "address", s.http.Addr, "scheme", s.Scheme(), "certificate_sha256", s.CertificateFingerprint())
	var err error
	if s.tlsEnabled {
		err = s.http.ListenAndServeTLS("", "")
	} else {
		err = s.http.ListenAndServe()
	}
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}

func requestLogging(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		capture := &responseCapture{ResponseWriter: w}
		next.ServeHTTP(capture, r)
		logger.Info("HTTP request", "request_id", requestIDFrom(r), "method", r.Method, "path", r.URL.Path, "remote", r.RemoteAddr, "status", capture.status(), "response_bytes", capture.bytes, "duration", time.Since(started).String())
	})
}

type requestIDKey struct{}
type authenticatedKey struct{}
type authenticationKindKey struct{}

// Authenticated reports whether the request passed the configured global or
// peer bearer-token check. Feature modules can use this to distinguish an
// authenticated LAN request from an unauthenticated loopback request.
func Authenticated(r *http.Request) bool {
	value, _ := r.Context().Value(authenticatedKey{}).(bool)
	return value
}

// ManagementAuthenticated reports whether the configured management bearer
// token, rather than a peer token, authenticated the request.
func ManagementAuthenticated(r *http.Request) bool {
	value, _ := r.Context().Value(authenticationKindKey{}).(string)
	return value == "management"
}

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if !validRequestID(id) {
			id = newRequestID()
		}
		ctx := context.WithValue(r.Context(), requestIDKey{}, id)
		r.Header.Set("X-Request-ID", id)
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func validRequestID(id string) bool {
	if id == "" || len(id) > 128 {
		return false
	}
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' || r == ':' {
			continue
		}
		return false
	}
	return true
}

func requestIDFrom(r *http.Request) string {
	if id, ok := r.Context().Value(requestIDKey{}).(string); ok {
		return id
	}
	return strings.TrimSpace(r.Header.Get("X-Request-ID"))
}

func newRequestID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err == nil {
		return fmt.Sprintf("%x", raw[:])
	}
	return fmt.Sprintf("%x", time.Now().UnixNano())
}

func authentication(s *Server, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.authToken == "" || r.URL.Path == "/api/v1/system/health" || s.isPublicPath(r.URL.Path) || (s.isLocalManagementPath(r.URL.Path) && requestIsLoopback(r)) {
			next.ServeHTTP(w, r)
			return
		}
		const prefix = "Bearer "
		provided := strings.TrimSpace(r.Header.Get("Authorization"))
		if len(provided) <= len(prefix) || !strings.EqualFold(provided[:len(prefix)], prefix) {
			writeError(w, http.StatusUnauthorized, "authentication required", requestIDFrom(r))
			return
		}
		provided = strings.TrimSpace(provided[len(prefix):])
		validGlobal := subtle.ConstantTimeCompare([]byte(provided), []byte(s.authToken)) == 1
		validPeer := s.peerTokenValidator != nil && s.peerTokenValidator(provided)
		if !validGlobal && !validPeer {
			writeError(w, http.StatusUnauthorized, "invalid authentication token", requestIDFrom(r))
			return
		}
		kind := "peer"
		if validGlobal {
			kind = "management"
		}
		ctx := context.WithValue(r.Context(), authenticatedKey{}, true)
		ctx = context.WithValue(ctx, authenticationKindKey{}, kind)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) isLocalManagementPath(path string) bool {
	return path == "/" || path == "/app" || strings.HasPrefix(path, "/app/") || strings.HasPrefix(path, "/api/v1/files/") || path == "/api/v1/settings" || strings.HasPrefix(path, "/api/v1/settings/")
}

func requestIsLoopback(r *http.Request) bool {
	host := strings.TrimSpace(r.RemoteAddr)
	if parsed, _, err := net.SplitHostPort(host); err == nil {
		host = parsed
	}
	host = strings.Trim(host, "[]")
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (s *Server) isPublicPath(path string) bool {
	for _, prefix := range s.publicPathPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

type responseCapture struct {
	http.ResponseWriter
	statusCode int
	bytes      int
}

func (w *responseCapture) WriteHeader(status int) {
	if w.statusCode != 0 {
		return
	}
	w.statusCode = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseCapture) Write(data []byte) (int, error) {
	if w.statusCode == 0 {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(data)
	w.bytes += n
	return n, err
}

func (w *responseCapture) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (w *responseCapture) status() int {
	if w.statusCode == 0 {
		return http.StatusOK
	}
	return w.statusCode
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message, requestID string) {
	writeJSON(w, status, map[string]string{"error": message, "request_id": requestID})
}
