package server

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type Server struct {
	logger             *slog.Logger
	http               *http.Server
	authToken          string
	peerTokenValidator func(string) bool
	runtimeInfo        RuntimeInfo
}

type RuntimeInfo struct {
	Version      string
	DeviceID     string
	DeviceName   string
	Capabilities []string
}

func New(address string, readTimeout, writeTimeout, idleTimeout time.Duration, logger *slog.Logger, routes func(*http.ServeMux)) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	s := &Server{logger: logger, runtimeInfo: RuntimeInfo{Version: "dev"}}
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
			"capabilities": capabilities,
		})
	})
	if routes != nil {
		routes(mux)
	}
	handler := requestIDMiddleware(requestLogging(logger, authentication(s, mux)))
	s.http = &http.Server{Addr: address, Handler: handler, ReadTimeout: readTimeout, WriteTimeout: writeTimeout, IdleTimeout: idleTimeout}
	return s
}

func (s *Server) SetAuthToken(token string) { s.authToken = strings.TrimSpace(token) }

func (s *Server) SetPeerTokenValidator(validator func(string) bool) { s.peerTokenValidator = validator }

func (s *Server) SetRuntimeInfo(info RuntimeInfo) { s.runtimeInfo = info }

func (s *Server) Start() error {
	s.logger.Info("HTTP server started", "address", s.http.Addr)
	err := s.http.ListenAndServe()
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
		if s.authToken == "" || r.URL.Path == "/api/v1/system/health" {
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
		next.ServeHTTP(w, r)
	})
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
