package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/XyUFlaW1eSs/LocalBridge/internal/app"
	"github.com/XyUFlaW1eSs/LocalBridge/internal/config"
	"github.com/XyUFlaW1eSs/LocalBridge/internal/native"
	"github.com/XyUFlaW1eSs/LocalBridge/internal/server"
	"github.com/XyUFlaW1eSs/LocalBridge/internal/transport"
	"github.com/XyUFlaW1eSs/LocalBridge/internal/version"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "path to YAML configuration")
	showVersion := flag.Bool("version", false, "print version")
	checkConfig := flag.Bool("check-config", false, "validate configuration and print redacted effective values")
	shareFiles := flag.Bool("share", false, "share file arguments through LocalBridge")
	flag.Parse()
	if *showVersion {
		println("LocalBridge", version.Value)
		return
	}

	var cfg config.Config
	var err error
	if *checkConfig {
		cfg, err = config.Load(*configPath)
	} else {
		cfg, err = config.LoadOrDefault(*configPath)
	}
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}
	if *checkConfig {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(cfg.Diagnostics()); err != nil {
			slog.Error("failed to write configuration diagnostics", "error", err)
			os.Exit(1)
		}
		return
	}
	var paths []string
	if *shareFiles {
		paths, err = native.ValidateSharePaths(flag.Args())
		if err != nil {
			slog.Error("invalid share selection", "error", err)
			os.Exit(2)
		}
		if reached, apiErr := shareViaRunningService(cfg, paths); reached {
			if apiErr != nil {
				slog.Error("failed to share files through running LocalBridge", "error", apiErr)
				os.Exit(1)
			}
			_ = native.OpenURL(fmt.Sprintf("%s://127.0.0.1:%d/app/#shares", serverScheme(cfg), cfg.Server.Port))
			return
		}
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	executable, _ := os.Executable()
	runtime, err := app.NewWithOptions(cfg, app.Options{ConfigPath: *configPath, Executable: executable})
	if err != nil {
		slog.Error("failed to initialize application", "error", err)
		os.Exit(1)
	}
	if err := runtime.Start(ctx); err != nil {
		runtime.Logger().Error("failed to start application", "error", err)
		os.Exit(1)
	}
	runtime.Logger().Info("LocalBridge started", "version", version.Value)
	if *shareFiles {
		if _, err := runtime.SharePaths(paths, fmt.Sprintf("explorer-%d-%d", os.Getpid(), time.Now().UnixNano())); err != nil {
			runtime.Logger().Error("failed to create Explorer file share", "error", err)
			stop()
		} else if err := runtime.OpenGUI("shares"); err != nil {
			runtime.Logger().Warn("failed to open file-share GUI", "error", err)
		}
	} else if err := runtime.OpenGUI(""); err != nil {
		runtime.Logger().Warn("failed to open GUI", "error", err)
	}
	select {
	case <-ctx.Done():
	case <-runtime.ExitRequests():
		stop()
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := runtime.Shutdown(shutdownCtx); err != nil {
		runtime.Logger().Error("shutdown failed", "error", err)
		os.Exit(1)
	}
	runtime.Logger().Info("LocalBridge stopped")
}

func shareViaRunningService(cfg config.Config, paths []string) (bool, error) {
	files := make([]map[string]string, 0, len(paths))
	for _, path := range paths {
		files = append(files, map[string]string{"path": path})
	}
	body, err := json.Marshal(map[string]any{"files": files})
	if err != nil {
		return false, err
	}
	endpoint := transport.Endpoint{Address: "127.0.0.1", Port: cfg.Server.Port, Secure: cfg.Server.TLSEnabled}
	if cfg.Server.TLSEnabled {
		tlsConfig, err := server.LoadTLSCertificate(cfg.Server.TLSCertFile, cfg.Server.TLSKeyFile)
		if err != nil {
			return false, nil
		}
		endpoint.CertificateSHA256 = tlsConfig.Fingerprint
	}
	client, err := transport.NewClient(2 * time.Second).HTTPClient(endpoint)
	if err != nil {
		return false, nil
	}
	request, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s://127.0.0.1:%d/api/v1/files/shares", serverScheme(cfg), cfg.Server.Port), bytes.NewReader(body))
	if err != nil {
		return false, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", fmt.Sprintf("explorer-%d-%d", os.Getpid(), time.Now().UnixNano()))
	response, err := client.Do(request)
	if err != nil {
		return false, nil
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		message, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return true, fmt.Errorf("service returned %s: %s", response.Status, bytes.TrimSpace(message))
	}
	return true, nil
}

func serverScheme(cfg config.Config) string {
	if cfg.Server.TLSEnabled {
		return "https"
	}
	return "http"
}
