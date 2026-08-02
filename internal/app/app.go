package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/XyUFlaW1eSs/LocalBridge/internal/config"
	"github.com/XyUFlaW1eSs/LocalBridge/internal/eventbus"
	"github.com/XyUFlaW1eSs/LocalBridge/internal/logger"
	"github.com/XyUFlaW1eSs/LocalBridge/internal/module"
	"github.com/XyUFlaW1eSs/LocalBridge/internal/modules/clipboard"
	"github.com/XyUFlaW1eSs/LocalBridge/internal/server"
)

type App struct {
	cfg     config.Config
	logger  *slog.Logger
	bus     *eventbus.Bus
	manager *module.Manager
	server  *server.Server
}

func New(cfg config.Config) (*App, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	log := logger.New(cfg.Logging.Level, cfg.Logging.Format, nil)
	bus := eventbus.New()
	manager := module.NewManager()
	if cfg.Clipboard.Enabled {
		if err := manager.Register(clipboard.New(cfg.Clipboard, cfg.Device, bus, log)); err != nil {
			return nil, err
		}
	}
	srv := server.New(cfg.Server.Address(), cfg.Server.ReadTimeout, cfg.Server.WriteTimeout, cfg.Server.IdleTimeout, log, manager.Routes)
	return &App{cfg: cfg, logger: log, bus: bus, manager: manager, server: srv}, nil
}

func (a *App) Start(ctx context.Context) error {
	if err := a.manager.Start(ctx); err != nil {
		return err
	}
	go func() {
		if err := a.server.Start(); err != nil {
			a.logger.Error("HTTP server stopped unexpectedly", "error", err)
		}
	}()
	return nil
}

func (a *App) Shutdown(ctx context.Context) error {
	var first error
	if err := a.server.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		first = err
	}
	moduleCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := a.manager.Stop(moduleCtx); err != nil && first == nil {
		first = err
	}
	return first
}

func (a *App) Logger() *slog.Logger    { return a.logger }
func (a *App) EventBus() *eventbus.Bus { return a.bus }
