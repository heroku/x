// Package svclog provides logging facilities for standard services.
package svclog

import (
	"fmt"
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
	logger := slog.New(slog.NewTextHandler(os.Stdout, hopts)).With(
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
func ReportPanic(logger slog.Logger) {
	if p := recover(); p != nil {
		s := fmt.Sprint(p)
		logger.With("at", "panic").Error(s)
		panic(p)
	}
}

/*
// NewSampleLogger creates a rate limited logger that samples logs. The parameter
// logsBurstLimit defines how many logs are allowed per logBurstWindow duration.
// The returned logger derives from the parentLogger, but without inheriting any Hooks.
// All log entries derived from SampleLogger will contain 'sampled=true' field.
func NewSampleLogger(parentLogger slog.Logger, logsBurstLimit int, logBurstWindow time.Duration) slog.Logger {
	entry := parentLogger.With("sampled", true)
	ll := slog.New()
	ll.Out = entry.Logger.Out
	ll.Level = entry.Logger.Level
	ll.ReportCaller = entry.Logger.ReportCaller
	ll.Formatter = &sampleFormatter{
		origFormatter: entry.Logger.Formatter,
		limiter:       rate.NewLimiter(rate.Every(logBurstWindow), logsBurstLimit),
	}

	return ll.With(entry.Data)
}

type sampleFormatter struct {
	limiter       *rate.Limiter
	origFormatter slog.Formatter
}

func (sf *sampleFormatter) Format(e *slog.Entry) ([]byte, error) {
	if sf.limiter.Allow() {
		return sf.origFormatter.Format(e)
	}

	return nil, nil
}

// SaramaLogger takes Logger and returns a saramaLogger.
func SaramaLogger(logger slog.Logger) slog.Logger {
	logger = logger.With("component", "sarama")
	return saramaLogger{logger}
}

type saramaLogger struct {
	slog.Logger
}

func (sl saramaLogger) Printf(format string, args ...interface{}) {
	format = strings.TrimSpace(format)
	sl.Logger.Printf(format, args...)
}

// NewNullLogger returns a logger that discards the output useful for testing
func NewNullLogger() slog.Logger {
	logger := slog.New()
	logger.SetOutput(io.Discard)
	return logger
}

// LoggerOrNull ensures non-nil logger is passed in or creates a Null Logger
func LoggerOrNull(l slog.Logger) slog.Logger {
	if l == nil {
		return NewNullLogger()
	}

	return l
}
*/

func ParseLevel(s string) (slog.Level, error) {
	var level slog.Level
	var err = level.UnmarshalText([]byte(s))
	return level, err
}
