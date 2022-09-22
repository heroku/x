// Package metrics provides helpers for setting up metrics reporting.
package metrics

import (
	"time"

	"github.com/heroku/x/cmdutil/metrics/otel"
)

// Config stores all the env related config to bootstrap metrics.
type Config struct {
	ReportInterval time.Duration `env:"METRICS_REPORT_INTERVAL,default=60s"`
	Source         string        `env:"METRICS_SOURCE"`
	Prefix         string        `env:"METRICS_PREFIX"`
	DefaultTags    []string      `env:"METRICS_DEFAULT_TAGS"`
	OTEL           otel.Config
}
