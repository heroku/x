// Package health provides cmdutil-compatible healthcheck utilities.
package health

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/heroku/x/cmdutil"
	"github.com/heroku/x/go-kit/metrics"
	"github.com/heroku/x/tickgroup"
	"github.com/heroku/x/v2/healthcheck"
)

// NewTCPServer returns a cmdutil.Server which emits a health metric whenever a TCP
// connection is opened on the configured port.
func NewTCPServer(logger zerolog.Logger, provider metrics.Provider, cfg Config) cmdutil.Server {
	healthLogger := logger.With().Str("service", "healthcheck").Logger()
	healthLogger.Info().
		Str("at", "binding").
		Int("port", cfg.Port).
		Send()

	return healthcheck.NewTCPServer(healthLogger, provider, fmt.Sprintf(":%d", cfg.Port))
}

// NewTickingServer returns a cmdutil.Server which emits a health metric every
// cfg.MetricInterval seconds.
func NewTickingServer(logger zerolog.Logger, provider metrics.Provider, cfg Config) cmdutil.Server {
	logger.Info().
		Str("service", "healthcheck-worker").
		Str("at", "starting").
		Int("interval", cfg.MetricInterval).
		Send()

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
