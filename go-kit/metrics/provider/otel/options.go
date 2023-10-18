package otel

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/sdk/metric"
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
func WithDefaultExporter(ctx context.Context) Option {
	return WithGRPCExporter(ctx, DefaultAgentEndpoint)
}

func WithGRPCExporter(ctx context.Context, endpoint string, options ...otlpmetricgrpc.Option) Option {
	return func(p *Provider) error {
		defaults := []otlpmetricgrpc.Option{
			otlpmetricgrpc.WithEndpoint(endpoint),
			otlpmetricgrpc.WithInsecure(),
		}
		options = append(defaults, options...)
		exp, err := otlpmetricgrpc.New(ctx, options...)
		if err != nil {
			return err
		}

		p.exporter = exp
		return nil
	}
}

func WithReader(reader *metric.PeriodicReader) Option {
	return func(p *Provider) error {
		p.reader = reader

		return nil
	}
}

func WithExporter(exporter metric.Exporter) Option {
	return func(p *Provider) error {
		p.exporter = exporter

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
