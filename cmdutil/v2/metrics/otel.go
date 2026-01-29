package metrics

import (
	"context"
	"crypto/tls"
	"fmt"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"google.golang.org/grpc/credentials"
)

const (
	ServiceKey    = attribute.Key("_service")
	ComponentKey  = attribute.Key("component")
	StageKey      = attribute.Key("stage")
	SubServiceKey = attribute.Key("_subservice")
)

func Setup(ctx context.Context, cfg Config, serviceName, serviceNamespace, deploy, serviceInstanceID string, opts ...Option) (*metric.MeterProvider, func(context.Context) error, error) {
	if !cfg.Enabled {
		return metric.NewMeterProvider(), func(context.Context) error { return nil }, nil
	}

	if cfg.Endpoint == nil {
		return nil, nil, fmt.Errorf("OTEL_EXPORTER_OTLP_ENDPOINT is required when metrics enabled")
	}

	var options setupOptions
	for _, opt := range opts {
		opt(&options)
	}

	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
		semconv.ServiceNamespace(serviceNamespace),
		semconv.ServiceInstanceID(serviceInstanceID),
		ServiceKey.String(serviceName),
		ComponentKey.String(serviceName),
		StageKey.String(deploy),
		SubServiceKey.String(deploy),
	)
	if options.resource != nil {
		res, _ = resource.Merge(res, options.resource)
	}

	// Default to Delta temporality to match v1 behavior
	temporality := options.temporalitySelector
	if temporality == nil {
		temporality = func(metric.InstrumentKind) metricdata.Temporality {
			return metricdata.DeltaTemporality
		}
	}

	// Default to exponential histogram aggregation to match v1 behavior
	aggregation := options.aggregationSelector
	if aggregation == nil {
		aggregation = func(kind metric.InstrumentKind) metric.Aggregation {
			if kind == metric.InstrumentKindHistogram {
				return metric.AggregationBase2ExponentialHistogram{
					MaxSize:  160,
					MaxScale: 20,
				}
			}
			return metric.DefaultAggregationSelector(kind)
		}
	}

	var exporter metric.Exporter
	var err error

	switch cfg.Protocol {
	case "grpc":
		opts := []otlpmetricgrpc.Option{
			otlpmetricgrpc.WithEndpoint(cfg.Endpoint.Host),
			otlpmetricgrpc.WithTemporalitySelector(temporality),
			otlpmetricgrpc.WithAggregationSelector(aggregation),
		}
		if options.tlsConfig != nil {
			opts = append(opts, otlpmetricgrpc.WithTLSCredentials(credentials.NewTLS(options.tlsConfig)))
		}
		exporter, err = otlpmetricgrpc.New(ctx, opts...)
	case "http/protobuf":
		opts := []otlpmetrichttp.Option{
			otlpmetrichttp.WithEndpoint(cfg.Endpoint.Host),
			otlpmetrichttp.WithTemporalitySelector(temporality),
			otlpmetrichttp.WithAggregationSelector(aggregation),
		}
		if options.tlsConfig != nil {
			opts = append(opts, otlpmetrichttp.WithTLSClientConfig(options.tlsConfig))
		} else if cfg.Endpoint.Scheme == "http" {
			opts = append(opts, otlpmetrichttp.WithInsecure())
		}
		exporter, err = otlpmetrichttp.New(ctx, opts...)
	default:
		return nil, nil, fmt.Errorf("unsupported protocol: %s", cfg.Protocol)
	}

	if err != nil {
		return nil, nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	reader := metric.NewPeriodicReader(exporter, metric.WithInterval(cfg.Interval))

	provider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(reader),
	)

	if cfg.EnableRuntimeMetrics {
		if err := runtime.Start(runtime.WithMeterProvider(provider)); err != nil {
			return nil, nil, fmt.Errorf("failed to start runtime metrics: %w", err)
		}
	}

	shutdown := func(ctx context.Context) error {
		return provider.Shutdown(ctx)
	}

	return provider, shutdown, nil
}

type setupOptions struct {
	resource            *resource.Resource
	tlsConfig           *tls.Config
	temporalitySelector metric.TemporalitySelector
	aggregationSelector metric.AggregationSelector
}

type Option func(*setupOptions)

func WithResource(r *resource.Resource) Option {
	return func(o *setupOptions) {
		o.resource = r
	}
}

func WithTLSConfig(cfg *tls.Config) Option {
	return func(o *setupOptions) {
		o.tlsConfig = cfg
	}
}

func WithTemporality(selector metric.TemporalitySelector) Option {
	return func(o *setupOptions) {
		o.temporalitySelector = selector
	}
}

func WithAggregation(selector metric.AggregationSelector) Option {
	return func(o *setupOptions) {
		o.aggregationSelector = selector
	}
}
