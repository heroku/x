// Package health provides cmdutil-compatible healthcheck utilities.
package health

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/heroku/x/cmdutil"
	"github.com/heroku/x/go-kit/metrics"
	"github.com/heroku/x/healthcheck"
	"github.com/heroku/x/tickgroup"
)

// NewTCPServer returns a cmdutil.Server which emits a health metric whenever a TCP
// connection is opened on the configured port.
func NewTCPServer(logger logrus.FieldLogger, provider metrics.Provider, cfg Config) cmdutil.Server {
	healthLogger := logger.WithField("service", "healthcheck")
	healthLogger.WithFields(logrus.Fields{
		"at":   "binding",
		"port": cfg.Port,
	}).Info()

	return healthcheck.NewTCPServer(healthLogger, provider, fmt.Sprintf(":%d", cfg.Port))
}

// NewTickingServer returns a cmdutil.Server which emits a health metric every
// cfg.MetricInterval seconds.
func NewTickingServer(logger logrus.FieldLogger, provider metrics.Provider, cfg Config) cmdutil.Server {
	logger.WithFields(logrus.Fields{
		"service":  "healthcheck-worker",
		"at":       "starting",
		"interval": cfg.MetricInterval,
	}).Info()

	c := provider.NewCounter("health")

	return cmdutil.NewContextServer(func(ctx context.Context) error {
		g := tickgroup.New(ctx)
		g.Go(time.Duration(cfg.MetricInterval)*time.Second, func() error {
			c.Add(1)
			return nil
		})
		return g.Wait()
	})
}
