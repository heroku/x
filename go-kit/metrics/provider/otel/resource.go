package otel

import (
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"

	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

// Service level resource attributes
const (
	// ServiceKey is the attribute key for the `_service` convention.
	ServiceKey = attribute.Key("_service")

	// ComponentKey is the attribute key for the `component` convention
	ComponentKey = attribute.Key("component")
)

// Service returns an attribute.KeyValue for the `_service` attribute.
func Service(name string) attribute.KeyValue {
	return ServiceKey.String(name)
}

// Component returns an attribute.KeyValue for the `component` attribute.
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
	CloudKey = attribute.Key("cloud")
)

// Stage returns an attribute.KeyValue for the `stage` attribute.
func Stage(val string) attribute.KeyValue {
	return StageKey.String(val)
}

// SubService returns an attribute.KeyValue for the `_subservice` attribute.
func SubService(val string) attribute.KeyValue {
	return SubServiceKey.String(val)
}

// Cloud returns an attribute.KeyValue for the `cloud` attribute.
func Cloud(val string) attribute.KeyValue {
	return CloudKey.String(val)
}

const (
	// HoneycombDatasetKey maps to the honeycomb specific dataset attribute
	HoneycombDatasetKey  = attribute.Key("dataset")
	MetricDestinationVal = "true"
)

// HoneycombDataset sets the value for the honeycomb `dataset` attribute.
func HoneycombDataset(val string) attribute.KeyValue {
	return HoneycombDatasetKey.String(val)
}

// MetricsDestinations appends attributes for each destination in the provided list.
func MetricsDestinations(destinations []string) []attribute.KeyValue {
	attrs := []attribute.KeyValue{}

	for _, md := range destinations {
		attrs = append(attrs, attribute.String(md, MetricDestinationVal))
	}
	return attrs
}

// SemConv returns a list of semconv attribute tags for `service.name`, `service.namespace` and `service.instance.id`.
func SemConv(name, namespace, instanceID string) []attribute.KeyValue {
	return []attribute.KeyValue{
		semconv.ServiceName(name),
		semconv.ServiceNamespace(namespace),
		semconv.ServiceInstanceID(instanceID),
	}
}

// ServiceConv returns an attriubte list for the `_service` and `component` convention.
func ServiceConv(name string) []attribute.KeyValue {
	return []attribute.KeyValue{
		Service(name),
		Component(name),
	}
}

// EnvironmentConv returns an attribute list for the `stage` and `_subservice` convention.
func EnvironmentConv(stage string) []attribute.KeyValue {
	return []attribute.KeyValue{
		Stage(stage),
		SubService(stage),
	}
}

// WithOpenTelemetryStandardService returns an Option that configures service
// attributes according to open-telemetry semconv conventions.
func WithOpenTelemetryStandardService(name, namespace, instanceID string) Option {
	return WithResource(resource.NewWithAttributes(
		semconv.SchemaURL,
		SemConv(name, namespace, instanceID)...,
	))
}

// WithServiceStandard returns an Option that configures service attributes for the ServiceConv convention.
func WithServiceStandard(name string) Option {
	return WithResource(resource.NewSchemaless(
		ServiceConv(name)...,
	))
}

// WithEnvironmentStandard returns an Option that configures service attributes for the EnvironmentConv convention.
func WithEnvironmentStandard(env string) Option {
	return WithResource(resource.NewSchemaless(
		EnvironmentConv(env)...,
	))
}

// WithResource will merge the given resource with the provided default, any
// duplicate attributes will be overwritten.
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
