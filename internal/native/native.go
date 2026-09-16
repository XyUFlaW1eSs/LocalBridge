package native

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/XyUFlaW1eSs/LocalBridge/internal/eventbus"
	"github.com/XyUFlaW1eSs/LocalBridge/internal/modules/settings"
)

// Config contains the platform integration boundary. The browser GUI remains
// the source of truth; native integrations only apply local OS affordances.
type Config struct {
	GUIURL       string
	Executable   string
	ConfigPath   string
	Settings     *settings.Store
	OnSharePaths func([]string) error
	OnOpenGUI    func(string) error
	OnExit       func()
	Bus          *eventbus.Bus
}

type backend interface {
	ApplySettings(settings.Settings) error
	Start(context.Context) error
	Stop(context.Context) error
	Notify(eventbus.Event, settings.Settings)
}

type Module struct {
	backend  backend
	settings *settings.Store
	logger   *slog.Logger
	bus      *eventbus.Bus
	cancel   context.CancelFunc
}

func New(cfg Config, logger *slog.Logger) (*Module, error) {
	if cfg.Settings == nil {
		return nil, errors.New("native integration requires settings store")
	}
	if logger == nil {
		logger = slog.Default()
	}
	impl, err := newBackend(cfg, logger)
	if err != nil {
		return nil, err
	}
	m := &Module{backend: impl, settings: cfg.Settings, logger: logger, bus: cfg.Bus}
	cfg.Settings.AddListener(func(value settings.Settings) {
		if err := m.backend.ApplySettings(value); err != nil {
			logger.Error("native settings synchronization failed", "error", err)
		}
	})
	return m, nil
}

func (m *Module) Name() string { return "native" }

func (m *Module) Start(ctx context.Context) error {
	if err := m.backend.ApplySettings(m.settings.Get()); err != nil {
		return err
	}
	runCtx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	if err := m.backend.Start(runCtx); err != nil {
		cancel()
		m.cancel = nil
		return err
	}
	if m.bus != nil {
		received := m.bus.Subscribe(runCtx, eventbus.FileReceived, 16)
		sent := m.bus.Subscribe(runCtx, eventbus.FileSent, 16)
		go m.notifyLoop(runCtx, received, sent)
	}
	return nil
}

func (m *Module) Stop(ctx context.Context) error {
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	return m.backend.Stop(ctx)
}

func (m *Module) notifyLoop(ctx context.Context, received, sent <-chan eventbus.Event) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-received:
			if ok {
				m.backend.Notify(event, m.settings.Get())
			}
		case event, ok := <-sent:
			if ok {
				m.backend.Notify(event, m.settings.Get())
			}
		}
	}
}

// ValidateSharePaths is the common validation used by the Explorer command
// entry point. It deliberately returns only normalized paths and never logs
// the original arguments.
func ValidateSharePaths(args []string) ([]string, error) {
	if len(args) == 0 {
		return nil, errors.New("at least one file is required")
	}
	paths := make([]string, 0, len(args))
	seen := make(map[string]struct{}, len(args))
	for _, raw := range args {
		path := strings.TrimSpace(raw)
		if path == "" {
			return nil, errors.New("file path must not be empty")
		}
		absolute, err := filepath.Abs(filepath.Clean(path))
		if err != nil {
			return nil, errors.New("file path is invalid")
		}
		info, err := os.Lstat(absolute)
		if err != nil {
			return nil, errors.New("file does not exist")
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return nil, errors.New("share arguments must be regular files")
		}
		if _, ok := seen[absolute]; ok {
			return nil, errors.New("share arguments contain a duplicate file")
		}
		seen[absolute] = struct{}{}
		paths = append(paths, absolute)
	}
	return paths, nil
}
