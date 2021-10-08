package otel

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/generic"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	metricexport "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/export/metric/aggregation"
	metriccontroller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/resource"

	xmetrics "github.com/heroku/x/go-kit/metrics"
)

var _ metrics.Counter = (*Counter)(nil)
var _ metrics.Gauge = (*Gauge)(nil)
var _ metrics.Histogram = (*Histogram)(nil)

const (
	// The values of these attributes should be the service name.
	serviceKey     = "_service"
	serviceNameKey = "service.name"
	componentKey   = "component"

	// The "service.namespace" attribute will be "heroku".
	serviceNamespaceKey = "service.namespace" // always "heroku"

	// The "service.instance.id" attribute will be an identifier for this specific instance of the service (e.g. "web.1").
	serviceInstanceIDKey = "service.instance.id"

	// The values of these attributes should be the stage (e.g. "production").
	stageKey      = "stage"
	subserviceKey = "_subservice"

	// The value of the "deploy" attribute should be the cloud (e.g. "eu")
	deployKey = "deploy"

	// The value of the "cloud" attribute should be the cloud (e.g. "heroku.com")
	cloudKey = "cloud"
)

// Provider initializes a global otlp meter provider that can collect metrics and
// use a collector to push those metrics to various backends (e.g. Argus, Honeycomb).
// Initialize with New(...). An initialized Provider must be started before
// it can be used to provide a Meter (i.e. p.Start()).
type Provider struct {
	ctx                 context.Context // used for init and shutdown of the otlp exporter and other bits of this Provider
	serviceNameResource *resource.Resource
	aggregator          metricexport.AggregatorSelector
	exporter            exporter
	controller          controller

	defaultTags []string
	prefix      string

	mu         sync.Mutex
	counters   map[string]*Counter
	gauges     map[string]*Gauge
	histograms map[string]*Histogram
}

// New returns a new, unstarted Provider. Use its Start() method to start
// and establish a connection with its exporter's collector agent.
func New(ctx context.Context, serviceName string, opts ...Option) (xmetrics.Provider, error) {
	p := Provider{
		ctx:        ctx,
		counters:   make(map[string]*Counter),
		gauges:     make(map[string]*Gauge),
		histograms: make(map[string]*Histogram),
	}

	defaultOpts := []Option{
		WithAttributes(
			attribute.String(serviceKey, serviceName),
			attribute.String(serviceNameKey, serviceName),
			attribute.String(componentKey, serviceName),
			attribute.String(serviceNamespaceKey, "heroku"),
		),
		WithDefaultAggregator(),
		WithDefaultEndpointExporter(),
	}

	opts = append(defaultOpts, opts...)
	for _, opt := range opts {
		if err := opt(&p); err != nil {
			return nil, fmt.Errorf("failed to apply options: %w", err)
		}
	}

	// initialize the controller
	p.controller = metriccontroller.New(
		processor.New(p.aggregator, p.exporter),
		metriccontroller.WithExporter(p.exporter),
		metriccontroller.WithResource(p.serviceNameResource),
	)
	global.SetMeterProvider(p.controller.MeterProvider())

	return &p, nil
}

type exporter interface {
	Start(ctx context.Context) error
	Shutdown(ctx context.Context) error
	Export(parent context.Context, resource *resource.Resource, cps metricexport.CheckpointSet) error
	ExportKindFor(desc *metric.Descriptor, kind aggregation.Kind) metricexport.ExportKind
}

type controller interface {
	MeterProvider() metric.MeterProvider
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// Start starts the provider's controller and exporter.
func (p *Provider) Start() error {
	if err := p.controller.Start(p.ctx); err != nil {
		return fmt.Errorf("failed to start controller: %w", err)
	}
	if err := p.exporter.Start(p.ctx); err != nil {
		return fmt.Errorf("failed to start exporter: %w", err)
	}
	return nil
}

// Stop shuts down the provider's controller and exporter.
// It should be used to ensure the metrics provider drains all metrics before exiting the program.
func (p *Provider) Stop() {
	_ = p.controller.Stop(p.ctx)
	_ = p.exporter.Shutdown(p.ctx)
}

// Meter returns an oltp meter used for the creation of metric instruments.
// This relies on the Provider's controller having been properly initialized.
func (p *Provider) Meter(name string) metric.Meter {
	m := global.Meter(name)
	return m
}

// Counter is a counter.
type Counter struct {
	metric.Float64Counter
	name       string
	labels     []string
	attributes []attribute.KeyValue
	p          *Provider
}

// Add implements metrics.Counter.
func (c *Counter) Add(delta float64) {
	c.Float64Counter.Add(c.p.ctx, delta, c.attributes...)
}

// With implements metrics.Counter.
func (c *Counter) With(labelValues ...string) metrics.Counter {
	lvs := append(append([]string(nil), c.labels...), labelValues...)
	return c.p.newCounter(c.name, lvs...)
}

// NewCounter creates a new Counter.
func (p *Provider) NewCounter(name string) metrics.Counter {
	return p.newCounter(prefixName(p.prefix, name), p.defaultTags...)
}

func (p *Provider) newCounter(name string, labelValues ...string) metrics.Counter {
	p.mu.Lock()
	defer p.mu.Unlock()

	k := keyName(name, labelValues...)
	m := p.Meter(name)

	if _, ok := p.counters[k]; !ok {
		c := metric.Must(m).NewFloat64Counter(name)

		p.counters[k] = &Counter{
			Float64Counter: c,
			labels:         labelValues,
			attributes:     makeAttributes(labelValues),
			p:              p,
			name:           name,
		}
	}

	return p.counters[k]
}

// Gauge is a gauge.
type Gauge struct {
	*generic.Gauge
	observer   *metric.Float64GaugeObserver
	name       string
	labels     []string
	attributes []attribute.KeyValue
	p          *Provider
}

// NewGauge implements metrics.Provider.
func (p *Provider) NewGauge(name string) metrics.Gauge {
	return p.newGauge(prefixName(p.prefix, name), p.defaultTags...)
}

func (p *Provider) newGauge(name string, labelValues ...string) metrics.Gauge {
	p.mu.Lock()
	defer p.mu.Unlock()

	k := keyName(name, labelValues...)
	m := p.Meter(name)

	attributes := makeAttributes(labelValues)

	if _, ok := p.gauges[k]; !ok {
		gg := generic.NewGauge(name)

		callback := func(ctx context.Context, result metric.Float64ObserverResult) {
			result.Observe(gg.Value(), attributes...)
		}

		g := metric.Must(m).NewFloat64GaugeObserver(name, callback)

		p.gauges[k] = &Gauge{
			Gauge:      gg,
			observer:   &g,
			labels:     labelValues,
			attributes: attributes,
			name:       name,
			p:          p,
		}
	}

	return p.gauges[k]
}

// With implements metrics.Gauge.
func (g *Gauge) With(labelValues ...string) metrics.Gauge {
	lvs := append(append([]string(nil), g.labels...), labelValues...)
	return g.p.newGauge(g.name, lvs...)
}

// Set implements metrics.Gauge.
func (g *Gauge) Set(value float64) {
	g.Gauge.Set(value)
}

// Add implements metrics.Gauge.
func (g *Gauge) Add(delta float64) {
	g.Gauge.Set(delta)
}

// Histogram is a histogram.
type Histogram struct {
	metric.Float64Histogram
	name       string
	labels     []string
	attributes []attribute.KeyValue
	p          *Provider
}

// NewHistogram implements metrics.Provider.
func (p *Provider) NewHistogram(name string, buckets int) metrics.Histogram {
	return p.newHistogram(prefixName(p.prefix, name), p.defaultTags...)
}

func (p *Provider) newHistogram(name string, labelValues ...string) metrics.Histogram {
	p.mu.Lock()
	defer p.mu.Unlock()

	k := keyName(name, labelValues...)
	m := p.Meter(name)

	if _, ok := p.histograms[k]; !ok {
		h := metric.Must(m).NewFloat64Histogram(name)

		p.histograms[k] = &Histogram{
			Float64Histogram: h,
			name:             name,
			labels:           labelValues,
			attributes:       makeAttributes(labelValues),
			p:                p,
		}
	}

	return p.histograms[k]
}

// With implements metrics.Histogram.
func (h *Histogram) With(labelValues ...string) metrics.Histogram {
	lvs := append(append([]string(nil), h.labels...), labelValues...)
	return h.p.newHistogram(h.name, lvs...)
}

// Observe implements metrics.Histogram.
func (h *Histogram) Observe(value float64) {
	h.Float64Histogram.Record(h.p.ctx, value, h.attributes...)
}

// NewCardinalityCounter implements metrics.Provider.
func (p *Provider) NewCardinalityCounter(name string) xmetrics.CardinalityCounter {
	return &xmetrics.HLLCounter{}
}

func prefixName(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}

// keyName is used as the map key for counters, gauges, and histograms
// and incorporates the name and the labelValues.
func keyName(name string, labelValues ...string) string {
	if len(labelValues) == 0 {
		return name
	}

	l := len(labelValues)
	parts := make([]string, 0, l/2)
	for i := 0; i < l; i += 2 {
		parts = append(parts, labelValues[i]+":"+labelValues[i+1])
	}
	return name + "." + strings.Join(parts, ".")
}

// makeAttributes is used to convert labels into attribute.KeyValues.
func makeAttributes(labels []string) (attributes []attribute.KeyValue) {
	attributes = make([]attribute.KeyValue, len(labels))
	if len(labels)%2 != 0 {
		labels = append(labels, "unknown")
	}

	for i := 0; i < len(labels); i += 2 {
		attributes = append(attributes, attribute.String(labels[i], labels[i+1]))
	}
	return
}
