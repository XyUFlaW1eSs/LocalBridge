package server

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

type Server struct {
	logger *slog.Logger
	http   *http.Server
}

func New(address string, readTimeout, writeTimeout, idleTimeout time.Duration, logger *slog.Logger, routes func(*http.ServeMux)) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/system/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "service": "localbridge", "time": time.Now().UTC()})
	})
	if routes != nil {
		routes(mux)
	}
	handler := requestLogging(logger, mux)
	return &Server{logger: logger, http: &http.Server{Addr: address, Handler: handler, ReadTimeout: readTimeout, WriteTimeout: writeTimeout, IdleTimeout: idleTimeout}}
}

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
		logger.Info("HTTP request", "method", r.Method, "path", r.URL.Path, "remote", r.RemoteAddr, "status", capture.status(), "response_bytes", capture.bytes, "duration", time.Since(started).String())
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
