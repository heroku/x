package otel

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"

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

const (
	// DefaultAgentEndpoint is a default exporter endpoint that points to a local otel collector.
	DefaultAgentEndpoint = "0.0.0.0:55680"

	// If you are encountering this error it means that you are attempting to
	// establish an aggregation selection (explicit, or exponential) after you have
	// created an exporter.
	exporterAlreadyCreatedPanic = "histogram aggregation selection must happen before exporter selection"
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

var DefaultAggregationSelector = WithExponentialHistogramAggregationSelector

func WithExponentialHistogramAggregationSelector() Option {
	return WithAggregationSelector(ExponentialAggregationSelector)
}

func WithExplicitHistogramAggregationSelector() Option {
	return WithAggregationSelector(ExplicitAggregationSelector)
}

func WithAggregationSelector(selector metric.AggregationSelector) Option {
	return func(c *config) error {
		c.aggregationSelector = selector

		return nil
	}
}

func WithService(name, namespace, instanceID string) Option {
	return WithResource(resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(name),
		semconv.ServiceNamespace(namespace),
		semconv.ServiceInstanceID(instanceID),
	))
}

func WithResource(res *resource.Resource) Option {
	return func(cfg *config) error {
		merged, err := resource.Merge(cfg.serviceResource, res)
		if err != nil {
			return fmt.Errorf("failed to merge resources: %w", err)
		}

		cfg.serviceResource = merged
		return nil
	}
}

// WithAttributes initializes a serviceNameResource with attributes.
// If a resource already exists, a new resource is created by merging the two resources.
func WithAttributes(attributes ...attribute.KeyValue) Option {
	return WithResource(resource.NewWithAttributes(semconv.SchemaURL, attributes...))
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
	return WithAttributes(semconv.ServiceNamespace(serviceNamespace))
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
	return WithAttributes(semconv.ServiceInstanceID(serviceInstanceID))
}

// WithDefaultEndpointExporter initializes the Provider with an exporter using a default endpoint.
func WithDefaultEndpointExporter() Option {
	return WithHTTPExporter(DefaultAgentEndpoint)
}

func WithHTTPExporter(endpoint string, options ...otlpmetrichttp.Option) Option {
	return WithExporterFunc(func(cfg *config) (metric.Exporter, error) {
		u, err := url.Parse(endpoint)
		if err != nil {
			return nil, err
		}

		authHeader := make(map[string]string)
		authHeader["Authorization"] = "Basic" + base64.StdEncoding.EncodeToString([]byte(u.User.String()))

		defaults := []otlpmetrichttp.Option{
			otlpmetrichttp.WithEndpoint(u.Hostname()),
			otlpmetrichttp.WithTLSClientConfig(tlsconfig.New()),
			otlpmetrichttp.WithHeaders(authHeader),
			otlpmetrichttp.WithAggregationSelector(cfg.aggregationSelector),
		}
		options = append(defaults, options...)

		return otlpmetrichttp.New(cfg.ctx, options...)
	})
}

func WithGRPCExporter(endpoint string, options ...otlpmetricgrpc.Option) Option {
	return WithExporterFunc(func(cfg *config) (metric.Exporter, error) {
		defaults := []otlpmetricgrpc.Option{
			otlpmetricgrpc.WithEndpoint(endpoint),
			otlpmetricgrpc.WithInsecure(),
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
