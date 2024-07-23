// Package svclog provides logging facilities for standard services.
package svclog

import (
	"fmt"
	"io"
	"log"
	"os"

	"log/slog"
)

// Config for logger.
type Config struct {
	AppName  string `env:"APP_NAME,required"`
	Deploy   string `env:"DEPLOY,required"`
	SpaceID  string `env:"SPACE_ID"`
	Dyno     string `env:"DYNO"`
	LogLevel string `env:"LOG_LEVEL,default=INFO"`

	WriteTo io.Writer
}

// NewLogger returns a new logger that includes app and deploy key/value pairs
// in each log line.
func NewLogger(cfg Config) *slog.Logger {
	level, err := ParseLevel(cfg.LogLevel)
	if err != nil {
		log.Fatal(err)
	}

	hopts := &slog.HandlerOptions{
		Level: level,
	}
	var w io.Writer
	w = cfg.WriteTo
	if w == nil {
		w = os.Stdout
	}
	logger := slog.New(slog.NewTextHandler(w, hopts)).With(
		"app", cfg.AppName,
		"deploy", cfg.Deploy,
	)

	if cfg.SpaceID != "" {
		logger = logger.With("space", cfg.SpaceID)
	}
	if cfg.Dyno != "" {
		logger = logger.With("dyno", cfg.Dyno)
	}

	return logger
}

// ReportPanic attempts to report the panic to rollbar via the slog.
func ReportPanic(logger *slog.Logger) {
	if p := recover(); p != nil {
		s := fmt.Sprint(p)
		logger.With("at", "panic").Error(s)
		panic(p)
	}
}

func ParseLevel(s string) (slog.Level, error) {
	var level slog.Level
	var err = level.UnmarshalText([]byte(s))
	return level, err
}
