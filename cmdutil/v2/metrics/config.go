package metrics

import (
	"net/url"
	"time"
)

type Config struct {
	Enabled              bool          `env:"OTEL_METRICS_ENABLED,default=false"`
	Endpoint             *url.URL      `env:"OTEL_EXPORTER_OTLP_ENDPOINT"`
	Protocol             string        `env:"OTEL_EXPORTER_OTLP_PROTOCOL,default=http/protobuf"`
	Interval             time.Duration `env:"OTEL_METRIC_EXPORT_INTERVAL,default=60s"`
	EnableRuntimeMetrics bool          `env:"OTEL_ENABLE_RUNTIME_METRICS,default=false"`
}
