package otel

import (
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"

	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// Service level resource attributes
const (
	// ServiceKey is the attribute key for the `_service` convention.
	ServiceKey = attribute.Key("_service")

	// ComponentKey is the attribute key for the `component` convention
	ComponentKey = attribute.Key("component")
)

func Service(name string) attribute.KeyValue {
	return ServiceKey.String(name)
}

func Component(name string) attribute.KeyValue {
	return ComponentKey.String(name)
}

// Environment level resource attributes
const (
	// StageKey is the attribute key for the `stage` convention.
	StageKey = attribute.Key("stage")

	// SubServiceKey is the attribute key for the `_subservice` convention.
	SubServiceKey = attribute.Key("_subservice")

	// CloudKey is the attribute key for the `cloud` convention.
	CloudKey = attribute.Key("_subservice")
)

func Stage(val string) attribute.KeyValue {
	return StageKey.String(val)
}

func SubService(val string) attribute.KeyValue {
	return SubServiceKey.String(val)
}

func Cloud(val string) attribute.KeyValue {
	return CloudKey.String(val)
}

const (
	HoneycombDatasetKey  = attribute.Key("dataset")
	MetricDestinationVal = "true"
)

func HoneycombDataset(val string) attribute.KeyValue {
	return HoneycombDatasetKey.String(val)
}

func MetricDestinations(destinations []string) []attribute.KeyValue {
	attrs := []attribute.KeyValue{}

	for _, md := range destinations {
		attrs = append(attrs, attribute.String(md, MetricDestinationVal))
	}
	return attrs
}

func OtelSemConv(name, namespace, instanceID string) []attribute.KeyValue {
	return []attribute.KeyValue{
		semconv.ServiceName(name),
		semconv.ServiceNamespace(namespace),
		semconv.ServiceInstanceID(instanceID),
	}
}

func ServiceSemConv(name string) []attribute.KeyValue {
	return []attribute.KeyValue{
		Service(name),
		Component(name),
	}
}

func EnvironmentSemConv(stage string) []attribute.KeyValue {
	return []attribute.KeyValue{
		Stage(stage),
		SubService(stage),
	}
}

func WithOpenTelemetryStandardService(name, namespace, instanceID string) Option {
	return WithResource(resource.NewWithAttributes(
		semconv.SchemaURL,
		OtelSemConv(name, namespace, instanceID)...,
	))
}

func WithServiceStandard(name string) Option {
	return WithResource(resource.NewSchemaless(
		ServiceSemConv(name)...,
	))
}

func WithEnvironmentStandard(env string) Option {
	return WithResource(resource.NewSchemaless(
		EnvironmentSemConv(env)...,
	))
}

// WithResource will merge the given resource with the provided default, any duplicate attributes will be overwritten.
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
// This is a deprecated function provided for convenence.
func WithAttributes(attributes ...attribute.KeyValue) Option {
	return WithResource(resource.NewWithAttributes(semconv.SchemaURL, attributes...))
}
