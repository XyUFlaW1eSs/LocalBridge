package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/XyUFlaW1eSs/LocalBridge/internal/config"
	"github.com/XyUFlaW1eSs/LocalBridge/internal/eventbus"
	"github.com/XyUFlaW1eSs/LocalBridge/internal/logger"
	"github.com/XyUFlaW1eSs/LocalBridge/internal/module"
	"github.com/XyUFlaW1eSs/LocalBridge/internal/modules/clipboard"
	deviceModule "github.com/XyUFlaW1eSs/LocalBridge/internal/modules/device"
	fileModule "github.com/XyUFlaW1eSs/LocalBridge/internal/modules/files"
	settingsModule "github.com/XyUFlaW1eSs/LocalBridge/internal/modules/settings"
	syncModule "github.com/XyUFlaW1eSs/LocalBridge/internal/modules/sync"
	"github.com/XyUFlaW1eSs/LocalBridge/internal/native"
	"github.com/XyUFlaW1eSs/LocalBridge/internal/server"
	"github.com/XyUFlaW1eSs/LocalBridge/internal/syncstore"
	"github.com/XyUFlaW1eSs/LocalBridge/internal/version"
	"github.com/XyUFlaW1eSs/LocalBridge/internal/web"
)

type App struct {
	cfg     config.Config
	logger  *slog.Logger
	bus     *eventbus.Bus
	manager *module.Manager
	server  *server.Server
	files   *fileModule.Module
	native  *native.Module
	exit    chan struct{}
}

type Options struct {
	ConfigPath string
	Executable string
}

func New(cfg config.Config) (*App, error) {
	return NewWithOptions(cfg, Options{})
}

func NewWithOptions(cfg config.Config, options Options) (*App, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	var tlsConfig server.TLSConfig
	if cfg.Server.TLSEnabled {
		var err error
		tlsConfig, err = server.LoadTLSCertificate(cfg.Server.TLSCertFile, cfg.Server.TLSKeyFile)
		if err != nil {
			return nil, fmt.Errorf("initialize TLS: %w", err)
		}
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
	capabilities := []string{"system.health", "system.capabilities", "system.config.read", "device.registry", "device.pairing", "device.discovery", "device.token.rotate"}
	if cfg.Sync.Enabled {
		capabilities = append(capabilities, "sync.jobs")
	}
	if cfg.Clipboard.Enabled {
		capabilities = append(capabilities, "clipboard.text.push", "clipboard.text.pull")
	}
	if cfg.Files.Enabled {
		capabilities = append(capabilities, "files.share", "files.download", "files.receive", "files.resume")
	}
	if cfg.Server.TLSEnabled {
		capabilities = append(capabilities, "transport.https")
	} else {
		capabilities = append(capabilities, "transport.http")
	}
	settings, err := settingsModule.New(cfg.Settings, log)
	if err != nil {
		return nil, err
	}
	if err := manager.Register(settings); err != nil {
		return nil, err
	}
	var files *fileModule.Module
	if cfg.Files.Enabled {
		files, err = fileModule.NewWithBus(cfg.Files, bus, log)
		if err != nil {
			return nil, err
		}
		if err := manager.Register(files); err != nil {
			return nil, err
		}
		files.SetAutoAcceptProvider(func() bool { return settings.Store().Get().AutoAccept })
	}
	devices, err := deviceModule.New(cfg.Device, cfg.Security, cfg.Discovery, cfg.Server.Port, capabilities, bus, jobStore, log)
	if err != nil {
		return nil, err
	}
	devices.SetLocalTransport(cfg.Server.TLSEnabled, tlsConfig.Fingerprint)
	if err := manager.Register(devices); err != nil {
		return nil, err
	}
	if cfg.Clipboard.Enabled {
		if err := manager.Register(clipboard.New(cfg.Clipboard, cfg.Device, bus, log)); err != nil {
			return nil, err
		}
	}
	exitRequests := make(chan struct{})
	var exitOnce sync.Once
	nativeModule, err := native.New(native.Config{
		GUIURL:     fmt.Sprintf("%s://127.0.0.1:%d/app/", serverScheme(cfg), cfg.Server.Port),
		Executable: options.Executable,
		ConfigPath: options.ConfigPath,
		Settings:   settings.Store(),
		Bus:        bus,
		OnExit: func() {
			exitOnce.Do(func() { close(exitRequests) })
		},
	}, log)
	if err != nil {
		return nil, err
	}
	if err := manager.Register(nativeModule); err != nil {
		return nil, err
	}
	srv := server.NewWithTLS(cfg.Server.Address(), cfg.Server.ReadTimeout, cfg.Server.WriteTimeout, cfg.Server.IdleTimeout, log, func(mux *http.ServeMux) {
		manager.Routes(mux)
		web.Routes(mux)
	}, tlsConfig)
	srv.SetAuthToken(authToken(cfg))
	srv.SetPeerTokenValidator(devices.ValidatePeerToken)
	srv.SetPublicPathPrefixes("/share/", "/receive/")
	srv.SetRuntimeInfo(server.RuntimeInfo{Version: version.Value, DeviceID: cfg.Device.ID, DeviceName: cfg.Device.Name, Capabilities: capabilities})
	srv.SetRuntimeConfig(cfg.Diagnostics())
	return &App{cfg: cfg, logger: log, bus: bus, manager: manager, server: srv, files: files, native: nativeModule, exit: exitRequests}, nil
}

func serverScheme(cfg config.Config) string {
	if cfg.Server.TLSEnabled {
		return "https"
	}
	return "http"
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

func (a *App) ExitRequests() <-chan struct{} { return a.exit }

func (a *App) SharePaths(paths []string, idempotencyKey string) (fileModule.Share, error) {
	if a.files == nil {
		return fileModule.Share{}, errors.New("file sharing is disabled")
	}
	return a.files.CreateShare(paths, idempotencyKey)
}

func (a *App) OpenGUI(view string) error {
	if a.native == nil {
		return errors.New("native integration is unavailable")
	}
	return a.native.OpenGUI(view)
}
