package otel

import (
	"context"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/heroku/x/go-kit/metrics"
	"github.com/heroku/x/go-kit/metrics/provider/otel"
)

// MustProvider ensures setting up and starting a otel.Provider succeeds.
// nolint: lll
func MustProvider(ctx context.Context, logger logrus.FieldLogger, cfg Config, service, serviceNamespace, stage, serviceInstanceID string, opts ...otel.Option) metrics.Provider {
	// This provider is used for metrics reporting to the  collector.
	logger.WithField("metrics_destinations", strings.Join(cfg.MetricsDestinations, ",")).Info("setting up  provider")

	// for backwards compatiblity prefer the CollectorURL over EndpointURL
	endpoint := cfg.CollectorURL
	if endpoint == nil {
		endpoint = cfg.EndpointURL
	}

	if endpoint == nil {
		logger.Fatal("provider collectorURL cannot be nil")
	}

	// configure some optional resource attributes
	attrs := otel.MetricsDestinations(cfg.MetricsDestinations)
	if cfg.Honeycomb.MetricsDataset != "" {
		attrs = append(attrs, otel.HoneycombDataset(cfg.Honeycomb.MetricsDataset))
	}

	allOpts := []otel.Option{
		// ensure we have service.id, service.namespace, and service.instance.id attributes
		otel.WithOpenTelemetryStandardService(service, serviceNamespace, serviceInstanceID),

		// ensure we have _service and component attributes
		otel.WithServiceStandard(service),

		// ensure we have stage and _subservice attributes
		otel.WithEnvironmentStandard(stage),

		// if set, ensure we have honeycomb dataset and metrics destination attributes set
		otel.WithAttributes(attrs...),

		// use the exponential aggregation selector, exponential histograms are generally easier to use than explicit
		otel.WithExponentialAggregation(),

		// use Delta Temporality by default
		otel.WithDeltaTemporality(),

		// ensure we use the http exporter
		otel.WithHTTPEndpointExporter(endpoint.String()),
	}
	allOpts = append(allOpts, opts...)

	otelProvider, err := otel.New(ctx, service, allOpts...)
	if err != nil {
		logger.Fatal(err)
	}

	return otelProvider
}
