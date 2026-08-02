package logger

import (
	"io"
	"log/slog"
	"os"
)

func New(cfgLevel, cfgFormat string, out io.Writer) *slog.Logger {
	level := slog.LevelInfo
	if cfgLevel == "debug" {
		level = slog.LevelDebug
	} else if cfgLevel == "warn" {
		level = slog.LevelWarn
	} else if cfgLevel == "error" {
		level = slog.LevelError
	}
	if out == nil {
		out = os.Stdout
	}
	opts := &slog.HandlerOptions{Level: level}
	if cfgFormat == "json" {
		return slog.New(slog.NewJSONHandler(out, opts))
	}
	return slog.New(slog.NewTextHandler(out, opts))
}
