package otel

import (
	"context"
	"strings"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric"
	"go.opentelemetry.io/otel/sdk/export/metric"

	"github.com/heroku/x/go-kit/metrics"
	otelprovider "github.com/heroku/x/go-kit/metrics/provider/otel"
)

// MustProvider ensures setting up and starting a otel.Provider succeeds.
// nolint: lll
func MustProvider(ctx context.Context, logger logrus.FieldLogger, cfg Config, service, serviceNamespace, stage, serviceInstanceID string) metrics.Provider {
	// This provider is used for metrics reporting to the  collector.
	logger.WithField("metrics_destinations", strings.Join(cfg.MetricsDestinations, ",")).Info("setting up  provider")

	client := otelprovider.NewHTTPClient(cfg.CollectorURL)
	expOpts := otlpmetric.WithMetricExportKindSelector(metric.DeltaExportKindSelector())
	exporter := otlpmetric.NewUnstarted(client, expOpts)

	attrs := []attribute.KeyValue{}
	if cfg.Honeycomb.MetricsDataset != "" {
		attrs = append(attrs, attribute.String("dataset", cfg.Honeycomb.MetricsDataset))
	}
	for _, md := range cfg.MetricsDestinations {
		attrs = append(attrs, attribute.String(md, "true"))
	}

	otelProvider, err := otelprovider.New(ctx, service,
		otelprovider.WithExporter(exporter),
		otelprovider.WithAttributes(attrs...),
		otelprovider.WithServiceNamespaceAttribute(serviceNamespace),
		otelprovider.WithServiceInstanceIDAttribute(serviceInstanceID),
		otelprovider.WithStageAttribute(stage),
	)
	if err != nil {
		logger.Fatal(err)
	}

	if err := otelProvider.(*otelprovider.Provider).Start(); err != nil {
		logger.WithError(err).Fatal("failed to start  metrics provider")
	}

	return otelProvider
}
