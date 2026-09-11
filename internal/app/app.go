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
	deviceModule "github.com/XyUFlaW1eSs/LocalBridge/internal/modules/device"
	fileModule "github.com/XyUFlaW1eSs/LocalBridge/internal/modules/files"
	syncModule "github.com/XyUFlaW1eSs/LocalBridge/internal/modules/sync"
	"github.com/XyUFlaW1eSs/LocalBridge/internal/server"
	"github.com/XyUFlaW1eSs/LocalBridge/internal/syncstore"
	"github.com/XyUFlaW1eSs/LocalBridge/internal/version"
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
	var jobStore *syncstore.Store
	if cfg.Sync.Enabled {
		var err error
		jobStore, err = syncstore.New(cfg.Sync.StorePath, cfg.Sync.MaxJobs, cfg.Sync.JobRetention)
		if err != nil {
			return nil, err
		}
		if err := manager.Register(syncModule.New(jobStore, log)); err != nil {
			return nil, err
		}
	}
	capabilities := []string{"system.health", "system.capabilities", "device.registry", "device.pairing", "device.discovery"}
	if cfg.Sync.Enabled {
		capabilities = append(capabilities, "sync.jobs")
	}
	if cfg.Clipboard.Enabled {
		capabilities = append(capabilities, "clipboard.text.push", "clipboard.text.pull")
	}
	if cfg.Files.Enabled {
		capabilities = append(capabilities, "files.share", "files.download", "files.receive", "files.resume")
	}
	if cfg.Files.Enabled {
		fileStore, err := fileModule.New(cfg.Files, log)
		if err != nil {
			return nil, err
		}
		if err := manager.Register(fileStore); err != nil {
			return nil, err
		}
	}
	devices, err := deviceModule.New(cfg.Device, cfg.Security, cfg.Discovery, cfg.Server.Port, capabilities, bus, jobStore, log)
	if err != nil {
		return nil, err
	}
	if err := manager.Register(devices); err != nil {
		return nil, err
	}
	if cfg.Clipboard.Enabled {
		if err := manager.Register(clipboard.New(cfg.Clipboard, cfg.Device, bus, log)); err != nil {
			return nil, err
		}
	}
	srv := server.New(cfg.Server.Address(), cfg.Server.ReadTimeout, cfg.Server.WriteTimeout, cfg.Server.IdleTimeout, log, manager.Routes)
	srv.SetAuthToken(authToken(cfg))
	srv.SetPeerTokenValidator(devices.ValidatePeerToken)
	srv.SetPublicPathPrefixes("/share/", "/receive/")
	srv.SetRuntimeInfo(server.RuntimeInfo{Version: version.Value, DeviceID: cfg.Device.ID, DeviceName: cfg.Device.Name, Capabilities: capabilities})
	return &App{cfg: cfg, logger: log, bus: bus, manager: manager, server: srv}, nil
}

func authToken(cfg config.Config) string {
	if !cfg.Security.AuthEnabled {
		return ""
	}
	return cfg.Security.BearerToken
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
