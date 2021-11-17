package otel

import (
	"net/url"

	"github.com/heroku/x/cmdutil/metrics/honeycomb"
)

// Config is a reusable configuration struct that contains the necessary
// environment variables to setup an metrics.Provider
type Config struct {
	Enabled             bool     `env:"ENABLE_OTEL_COLLECTION"`
	CollectorURL        *url.URL `env:"OTEL_COLLECTOR_URL"`
	MetricsDestinations []string `env:"OTEL_METRICS_DESTINATIONS,default=honeycomb,argus"`
	Honeycomb           honeycomb.Config
}
