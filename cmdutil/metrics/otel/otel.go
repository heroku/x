package otel

import (
	"context"
	"strings"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric"
	"go.opentelemetry.io/otel/sdk/export/metric"

	"github.com/heroku/x/go-kit/metrics"
	"github.com/heroku/x/go-kit/metrics/provider/otel"
)

// MustProvider ensures setting up and starting a otel.Provider succeeds.
// nolint: lll
func MustProvider(ctx context.Context, logger logrus.FieldLogger, cfg Config, service, serviceNamespace, stage, serviceInstanceID string, opts ...otel.Option) metrics.Provider {
	// This provider is used for metrics reporting to the  collector.
	logger.WithField("metrics_destinations", strings.Join(cfg.MetricsDestinations, ",")).Info("setting up  provider")

	if cfg.CollectorURL == nil {
		logger.Fatal("provider collectorURL cannot be nil")
	}

	client := otel.NewHTTPClient(*cfg.CollectorURL)
	expOpts := otlpmetric.WithMetricExportKindSelector(metric.DeltaExportKindSelector())
	exporter := otlpmetric.NewUnstarted(client, expOpts)

	attrs := []attribute.KeyValue{}
	if cfg.Honeycomb.MetricsDataset != "" {
		attrs = append(attrs, attribute.String("dataset", cfg.Honeycomb.MetricsDataset))
	}
	for _, md := range cfg.MetricsDestinations {
		attrs = append(attrs, attribute.String(md, "true"))
	}

	aggrOpt := otel.WithDefaultAggregator()
	if cfg.UseExactAggregator {
		aggrOpt = otel.WithExactAggregator()
	}

	allOpts := []otel.Option{
		aggrOpt,
		otel.WithExporter(exporter),
		otel.WithAttributes(attrs...),
		otel.WithServiceNamespaceAttribute(serviceNamespace),
		otel.WithServiceInstanceIDAttribute(serviceInstanceID),
		otel.WithStageAttribute(stage),
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
