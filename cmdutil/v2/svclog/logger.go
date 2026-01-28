// Package svclog provides logging facilities for standard services.
package svclog

import (
	"io"
	"log/slog"
	"os"
)

// Config for logger.
type Config struct {
	AppName  string `env:"APP_NAME,required"`
	Deploy   string `env:"DEPLOY,required"`
	SpaceID  string `env:"SPACE_ID"`
	Dyno     string `env:"DYNO"`
	LogLevel string `env:"LOG_LEVEL,default=info"`
}

// NewLogger returns a new logger that includes app and deploy key/value pairs
// in each log line.
func NewLogger(cfg Config) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: parseLevel(cfg.LogLevel),
	}

	handler := slog.NewTextHandler(os.Stderr, opts)

	logger := slog.New(handler).With(
		slog.String("app", cfg.AppName),
		slog.String("deploy", cfg.Deploy),
	)

	if cfg.SpaceID != "" {
		logger = logger.With(slog.String("space", cfg.SpaceID))
	}
	if cfg.Dyno != "" {
		logger = logger.With(slog.String("dyno", cfg.Dyno))
	}

	return logger
}

func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// NewNullLogger returns a logger that discards the output useful for testing
func NewNullLogger() *slog.Logger {
	handler := slog.NewTextHandler(io.Discard, nil)
	return slog.New(handler)
}
