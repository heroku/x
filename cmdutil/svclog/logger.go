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

type printfer interface {
	Printf(format string, args ...interface{})
}

// SampleLogger is a logger that allows to use Printf sampling. Burst logs are
// limited to N reports per given window.
type SampleLogger struct {
	logger  printfer
	limiter *rate.Limiter
}

// NewSampleLogger creates a rate limited logger that samples logs. The parameter
// logsBurstLimit defines how many logs are allowed per logBurstWindow duration.
//
// A SampleLogger can be created using https://godoc.org/github.com/sirupsen/logrus#Printf.
// The caller must explicitly call Printf to sample logs. Any other logging call
// will not be sampled.
//
//  sampledLogger := NewSampleLogger(logger, burstLimit, burstWindow)
//  sampledLogger.Printf("LOSSY")
//  sampledLogger.Debug("NOT LOSSY")
//  sampledLogger.Info("NOT LOSSY")
//  sampledLogger.Error("NOT LOSSY")
//
func NewSampleLogger(printfer printfer, logsBurstLimit int, logBurstWindow time.Duration) *SampleLogger {
	limiter := rate.NewLimiter(rate.Every(logBurstWindow), logsBurstLimit)
	return &SampleLogger{
		logger:  printfer,
		limiter: limiter,
	}
}

// Printf may log depending on if the limiter is exceeded or not.
func (l *SampleLogger) Printf(format string, args ...interface{}) {
	if l.limiter.Allow() {
		l.logger.Printf(format, args...)
	}
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
