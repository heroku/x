package otel

import (
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.opentelemetry.io/otel/sdk/resource"
)

var (
	// ErrExporterNil is returned if an exporter is required, but is not passed in.
	ErrExporterNil = errors.New("exporter cannot be nil")
	// ErrAggregatorNil is returned if an aggregator is required, but is not passed in.
	ErrAggregatorNil = errors.New("aggregator cannot be nil")
	// ErrEndpointNil is returned if an endpoint is required, but is not passed in.
	ErrEndpointNil = errors.New("endpoint cannot be nil")
)

// DefaultAgentEndpoint is a default exporter endpoint that points to a local otel collector.
const DefaultAgentEndpoint = "0.0.0.0:55680"

// Option is used for optional arguments when initializing Provider.
type Option func(*Provider) error

// WithDefaultAggregator initializes the Provider with a default aggregator.
func WithDefaultAggregator() Option {
	return WithAggregator(simple.NewWithExactDistribution())
}

// WithAggregator initializes the Provider with an aggregator used by its controller.
func WithAggregator(agg metric.AggregatorSelector) Option {
	return func(p *Provider) error {
		if agg == nil {
			return ErrAggregatorNil
		}

		p.selector = agg
		return nil
	}
}

// WithAttributes initializes a serviceNameResource with attributes.
// If a resource already exists, a new resource is created by merging the two resources.
func WithAttributes(attributes ...attribute.KeyValue) Option {
	return func(p *Provider) error {
		res, err := resource.New(p.ctx, resource.WithAttributes(attributes...))
		if err != nil {
			return err
		}

		mergedRes, err := resource.Merge(p.serviceNameResource, res)
		if err != nil {
			return fmt.Errorf("failed to merge resources: %w", err)
		}
		p.serviceNameResource = mergedRes
		return nil
	}
}

// WithStageAttribute adds the "stage" and "_subservice" attributes.
func WithStageAttribute(stage string) Option {
	attrs := []attribute.KeyValue{
		attribute.String(stageKey, stage),
		attribute.String(subserviceKey, stage),
	}
	return WithAttributes(attrs...)
}

// WithServiceNamespaceAttribute adds the "service.namespace" attribute.
func WithServiceNamespaceAttribute(serviceNamespace string) Option {
	attrs := []attribute.KeyValue{
		attribute.String(serviceNamespaceKey, serviceNamespace),
	}
	return WithAttributes(attrs...)
}

// WithCloudAttribute adds the "cloud" attribute.
func WithCloudAttribute(cloud string) Option {
	attrs := []attribute.KeyValue{
		attribute.String(cloudKey, cloud),
	}
	return WithAttributes(attrs...)
}

// WithServiceInstanceIDAttribute adds the "service.instance.id" attribute.
func WithServiceInstanceIDAttribute(serviceInstanceID string) Option {
	attrs := []attribute.KeyValue{
		attribute.String(serviceInstanceIDKey, serviceInstanceID),
	}
	return WithAttributes(attrs...)
}

// WithDefaultEndpointExporter initializes the Provider with an exporter using a default endpoint.
func WithDefaultEndpointExporter() Option {
	return WithEndpointExporter(DefaultAgentEndpoint)
}

// WithEndpointExporter initializes the Provider with a default exporter.
func WithEndpointExporter(endpoint string) Option {
	return func(p *Provider) error {
		if endpoint == "" {
			return ErrEndpointNil
		}
		p.exporter = defaultExporter(endpoint)
		return nil
	}
}

// WithExporter initializes the Provider with an exporter.
func WithExporter(exp exporter) Option {
	return func(p *Provider) error {
		if exp == nil {
			return ErrExporterNil
		}
		p.exporter = exp
		return nil
	}
}

// WithCollectPeriod initializes the controller with the collectPeriod.
func WithCollectPeriod(collectPeriod time.Duration) Option {
	return func(p *Provider) error {
		p.collectPeriod = collectPeriod
		return nil
	}
}

// defaultExporter returns a new otlp exporter that uses a gRPC driver.
// A collector agent endpoint (host:port) is required as the addr.
func defaultExporter(addr string) exporter {
	c := otlpmetricgrpc.NewClient(
		otlpmetricgrpc.WithEndpoint(addr),
		otlpmetricgrpc.WithInsecure(),
	)
	eo := otlpmetric.WithMetricExportKindSelector(metric.DeltaExportKindSelector())
	return otlpmetric.NewUnstarted(c, eo)
}
