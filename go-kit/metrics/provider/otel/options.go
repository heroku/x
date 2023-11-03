package otel

import (
	"encoding/base64"
	"errors"
	"net/url"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/sdk/metric"

	"github.com/heroku/x/tlsconfig"
)

var (
	// ErrExporterNil is returned if an exporter is required, but is not passed in.
	ErrExporterNil = errors.New("exporter cannot be nil")
	// ErrAggregatorNil is returned if an aggregator is required, but is not passed in.
	ErrAggregatorNil = errors.New("aggregator cannot be nil")
	// ErrEndpointNil is returned if an endpoint is required, but is not passed in.
	ErrEndpointNil = errors.New("endpoint cannot be nil")
)

var (
	DefaultAggregationSelector = WithExponentialHistograms
	DefaultEndpointExporter    = WithHTTPExporter
)

const (
	// DefaultAgentEndpoint is a default exporter endpoint that points to a local otel collector.
	DefaultAgentEndpoint = "http://0.0.0.0:55680"
)

// Option is used for optional arguments when initializing Provider.
type Option func(*config) error

func WithPrefix(prefix string) Option {
	return func(cfg *config) error {
		cfg.prefix = prefix
		return nil
	}
}

// WithCollectPeriod initial
func WithCollectPeriod(collectPeriod time.Duration) Option {
	return func(c *config) error {
		c.collectPeriod = collectPeriod
		return nil
	}
}

func WithTemporalitySelector(temporality metric.TemporalitySelector) Option {
	return func(c *config) error {
		c.temporalitySelector = temporality

		return nil
	}
}

func WithExponentialHistograms() Option {
	return WithAggregationSelector(ExponentialAggregationSelector)
}

func WithExplicitHistograms() Option {
	return WithAggregationSelector(ExplicitAggregationSelector)
}

func WithAggregationSelector(selector metric.AggregationSelector) Option {
	return func(c *config) error {
		c.aggregationSelector = selector

		return nil
	}
}

func WithHTTPExporter(options ...otlpmetrichttp.Option) Option {
	return WithHTTPEndpointExporter(DefaultAgentEndpoint, options...)
}

func WithHTTPEndpointExporter(endpoint string, options ...otlpmetrichttp.Option) Option {
	return WithExporterFunc(func(cfg *config) (metric.Exporter, error) {
		u, err := url.Parse(endpoint)
		if err != nil {
			return nil, err
		}

		defaults := []otlpmetrichttp.Option{
			otlpmetrichttp.WithEndpoint(u.Host),
			otlpmetrichttp.WithAggregationSelector(cfg.aggregationSelector),
			otlpmetrichttp.WithTemporalitySelector(cfg.temporalitySelector),
		}

		if u.Scheme == "https" {
			defaults = append(defaults, otlpmetrichttp.WithTLSClientConfig(tlsconfig.New()))
		} else {
			defaults = append(defaults, otlpmetrichttp.WithInsecure())
		}

		if u.User.String() != "" {
			authHeader := make(map[string]string)
			authHeader["Authorization"] = "Basic" + base64.StdEncoding.EncodeToString([]byte(u.User.String()))

			defaults = append(defaults, otlpmetrichttp.WithHeaders(authHeader))
		}

		// finally append any passed in options
		options = append(defaults, options...)

		return otlpmetrichttp.New(cfg.ctx, options...)
	})
}

func WithGRPCExporter(endpoint string, options ...otlpmetricgrpc.Option) Option {
	return WithExporterFunc(func(cfg *config) (metric.Exporter, error) {
		defaults := []otlpmetricgrpc.Option{
			otlpmetricgrpc.WithEndpoint(endpoint),
			otlpmetricgrpc.WithInsecure(),
			otlpmetricgrpc.WithTemporalitySelector(cfg.temporalitySelector),
			otlpmetricgrpc.WithAggregationSelector(cfg.aggregationSelector),
		}
		options = append(defaults, options...)
		return otlpmetricgrpc.New(cfg.ctx, options...)
	})
}

func WithExporterFunc(fn exporterFactory) Option {
	return func(c *config) error {
		c.exporterFactory = fn

		return nil
	}
}
