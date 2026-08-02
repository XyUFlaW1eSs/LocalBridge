package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/XyUFlaW1eSs/LocalBridge/internal/app"
	"github.com/XyUFlaW1eSs/LocalBridge/internal/config"
	"github.com/XyUFlaW1eSs/LocalBridge/internal/version"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "path to YAML configuration")
	showVersion := flag.Bool("version", false, "print version")
	flag.Parse()
	if *showVersion {
		println("LocalBridge", version.Value)
		return
	}

	cfg, err := config.LoadOrDefault(*configPath)
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}
	runtime, err := app.New(cfg)
	if err != nil {
		slog.Error("failed to initialize application", "error", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := runtime.Start(ctx); err != nil {
		runtime.Logger().Error("failed to start application", "error", err)
		os.Exit(1)
	}
	runtime.Logger().Info("LocalBridge started", "version", version.Value)
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := runtime.Shutdown(shutdownCtx); err != nil {
		runtime.Logger().Error("shutdown failed", "error", err)
		os.Exit(1)
	}
	runtime.Logger().Info("LocalBridge stopped")
}
