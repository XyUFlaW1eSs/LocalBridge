//go:build !windows

package native

import (
	"context"
	"log/slog"

	"github.com/XyUFlaW1eSs/LocalBridge/internal/eventbus"
	"github.com/XyUFlaW1eSs/LocalBridge/internal/modules/settings"
)

type platformBackend struct{}

func newBackend(Config, *slog.Logger) (backend, error)           { return platformBackend{}, nil }
func (platformBackend) ApplySettings(settings.Settings) error    { return nil }
func (platformBackend) Start(context.Context) error              { return nil }
func (platformBackend) Stop(context.Context) error               { return nil }
func (platformBackend) Notify(eventbus.Event, settings.Settings) {}
func (platformBackend) OpenGUI(string) error                     { return nil }
