// Package svclog provides logging facilities for standard services.
package svclog

import (
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
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
func NewLogger(cfg Config) logrus.FieldLogger {
	logger := logrus.WithFields(logrus.Fields{
		"app":    cfg.AppName,
		"deploy": cfg.Deploy,
	})
	if cfg.SpaceID != "" {
		logger = logger.WithField("space", cfg.SpaceID)
	}
	if cfg.Dyno != "" {
		logger = logger.WithField("dyno", cfg.Dyno)
	}

	if l, err := logrus.ParseLevel(cfg.LogLevel); err == nil {
		logrus.SetLevel(l)
	}
	return logger
}

// NewSampleLogger creates a rate limited logger that samples logs. The parameter
// logsBurstLimit defines how many logs are allowed per logBurstWindow duration.
// The returned logger derives from the parentLogger, but without inheriting any Hooks.
// All log entries derived from SampleLogger will contain 'sampled=true' field.
func NewSampleLogger(parentLogger logrus.FieldLogger, logsBurstLimit int, logBurstWindow time.Duration) logrus.FieldLogger {
	entry := parentLogger.WithField("sampled", true)
	ll := logrus.New()
	ll.Out = entry.Logger.Out
	ll.Level = entry.Logger.Level
	ll.ReportCaller = entry.Logger.ReportCaller
	ll.Formatter = &sampleFormatter{
		origFormatter: entry.Logger.Formatter,
		limiter:       rate.NewLimiter(rate.Every(logBurstWindow), logsBurstLimit),
	}

	return ll.WithFields(entry.Data)
}

type sampleFormatter struct {
	limiter       *rate.Limiter
	origFormatter logrus.Formatter
}

func (sf *sampleFormatter) Format(e *logrus.Entry) ([]byte, error) {
	if sf.limiter.Allow() {
		return sf.origFormatter.Format(e)
	}

	return nil, nil
}

// SaramaLogger takes FieldLogger and returns a saramaLogger.
func SaramaLogger(logger logrus.FieldLogger) logrus.FieldLogger {
	logger = logger.WithField("component", "sarama")
	return saramaLogger{logger}
}

type saramaLogger struct {
	logrus.FieldLogger
}

func (sl saramaLogger) Printf(format string, args ...interface{}) {
	format = strings.TrimSpace(format)
	sl.FieldLogger.Printf(format, args...)
}
