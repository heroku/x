// Package health provides cmdutil-compatible healthcheck utilities.
package health

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/heroku/x/cmdutil"
	"github.com/heroku/x/go-kit/metrics"
	"github.com/heroku/x/tickgroup"
	"github.com/heroku/x/v2/healthcheck"
)

// NewTCPServer returns a cmdutil.Server which emits a health metric whenever a TCP
// connection is opened on the configured port.
func NewTCPServer(logger *slog.Logger, provider metrics.Provider, cfg Config) cmdutil.Server {
	healthLogger := logger.With(slog.String("service", "healthcheck"))
	healthLogger.With(
		slog.String("at", "binding"),
		slog.Int("port", cfg.Port),
	).Info("")

	return healthcheck.NewTCPServer(healthLogger, provider, fmt.Sprintf(":%d", cfg.Port))
}

// NewTickingServer returns a cmdutil.Server which emits a health metric every
// cfg.MetricInterval seconds.
func NewTickingServer(logger *slog.Logger, provider metrics.Provider, cfg Config) cmdutil.Server {
	logger.With(
		slog.String("service", "healthcheck-worker"),
		slog.String("at", "starting"),
		slog.Int("interval", cfg.MetricInterval),
	).Info("")

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
