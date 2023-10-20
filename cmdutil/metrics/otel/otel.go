package otel

import (
	"context"
	"strings"

	"github.com/sirupsen/logrus"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"

	"github.com/heroku/x/go-kit/metrics"
	otel "github.com/heroku/x/go-kit/metrics/provider/otel"
)

// MustProvider ensures setting up and starting a otel.Provider succeeds.
// nolint: lll
func MustProvider(ctx context.Context, logger logrus.FieldLogger, cfg Config, service, serviceNamespace, stage, serviceInstanceID string, opts ...otel.Option) metrics.Provider {
	// This provider is used for metrics reporting to the  collector.
	logger.WithField("metrics_destinations", strings.Join(cfg.MetricsDestinations, ",")).Info("setting up  provider")

	if cfg.CollectorURL == nil {
		logger.Fatal("provider collectorURL cannot be nil")
	}

	attrs := []attribute.KeyValue{}
	if cfg.Honeycomb.MetricsDataset != "" {
		attrs = append(attrs, attribute.String("dataset", cfg.Honeycomb.MetricsDataset))
	}
	for _, md := range cfg.MetricsDestinations {
		attrs = append(attrs, attribute.String(md, "true"))
	}

	res := resource.NewSchemaless(attrs...)

	allOpts := []otel.Option{
		otel.WithOpenTelemetryStandardService(service, serviceNamespace, serviceInstanceID),
		otel.WithServiceStandard(service),
		otel.WithEnvironmentStandard(stage),
		otel.WithResource(res),
		otel.WithExponentialHistograms(),
		otel.WithHTTPEndpointExporter(cfg.CollectorURL.String()),
	}
	allOpts = append(allOpts, opts...)

	otelProvider, err := otel.New(ctx, service, allOpts...)
	if err != nil {
		logger.Fatal(err)
	}

	if err := otelProvider.(*otel.Provider).Start(); err != nil {
		logger.WithError(err).Fatal("failed to start  metrics provider")
	}

	return otelProvider
}
