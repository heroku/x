package otel

import (
	"context"
	"strings"

	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/generic"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdk "go.opentelemetry.io/otel/sdk/metric"

	xmetrics "github.com/heroku/x/go-kit/metrics"
)

var (
	_ metrics.Counter   = (*Counter)(nil)
	_ metrics.Gauge     = (*Gauge)(nil)
	_ metrics.Histogram = (*Histogram)(nil)
)

// Counter is a counter.
type Counter struct {
	metric.Float64Counter
	name       string
	labels     []string
	attributes attribute.Set
	p          *Provider
}

// Add implements metrics.Counter.
func (c *Counter) Add(delta float64) {
	c.Float64Counter.Add(c.p.cfg.ctx, delta, metric.WithAttributeSet(c.attributes))
}

// With implements metrics.Counter.
func (c *Counter) With(labelValues ...string) metrics.Counter {
	lvs := append(append([]string(nil), c.labels...), labelValues...)
	return c.p.newCounter(c.name, lvs...)
}

// NewCounter creates a new Counter.
func (p *Provider) NewCounter(name string) metrics.Counter {
	return p.newCounter(prefixName(p.cfg.prefix, name))
}

func (p *Provider) newCounter(name string, labelValues ...string) metrics.Counter {
	p.mu.Lock()
	defer p.mu.Unlock()

	k := keyName(name, labelValues...)
	m := p.meterProvider.Meter(name)

	if _, ok := p.counters[k]; !ok {
		c, _ := m.Float64Counter(name)

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
	observer   metric.Float64Observable
	name       string
	labels     []string
	attributes attribute.Set
	p          *Provider
}

// NewGauge implements metrics.Provider.
func (p *Provider) NewGauge(name string) metrics.Gauge {
	return p.newGauge(prefixName(p.cfg.prefix, name))
}

func (p *Provider) newGauge(name string, labelValues ...string) metrics.Gauge {
	p.mu.Lock()
	defer p.mu.Unlock()

	k := keyName(name, labelValues...)
	m := p.meterProvider.Meter(name)

	attributes := makeAttributes(labelValues)

	if _, ok := p.gauges[k]; !ok {
		gg := generic.NewGauge(name)

		callback := func(_ context.Context, result metric.Float64Observer) error {
			result.Observe(gg.Value(), metric.WithAttributeSet(attributes))

			return nil
		}

		g, _ := m.Float64ObservableGauge(name, metric.WithFloat64Callback(callback))

		p.gauges[k] = &Gauge{
			Gauge:      gg,
			observer:   g,
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
	g.Gauge.Add(delta)
}

// Histogram is a histogram.
type Histogram struct {
	metric.Float64Histogram
	stream     sdk.Stream
	labels     []string
	attributes attribute.Set
	p          *Provider
}

func (p *Provider) NewExplicitHistogram(name string, fn xmetrics.DistributionFunc) metrics.Histogram {
	stream := sdk.Stream{
		Name: prefixName(p.cfg.prefix, name),
		Aggregation: sdk.AggregationExplicitBucketHistogram{
			Boundaries: fn(),
		},
	}

	return p.newHistogram(stream)
}

// NewHistogram implements metrics.Provider.
func (p *Provider) NewHistogram(name string, buckets int) metrics.Histogram {
	if buckets <= 0 {
		buckets = defaultExponentialHistogramMaxSize
	}
	stream := sdk.Stream{
		Name: prefixName(p.cfg.prefix, name),
		Aggregation: sdk.AggregationBase2ExponentialHistogram{
			MaxSize:  int32(buckets),
			MaxScale: defaultExponentialHistogramMaxScale,
		},
	}
	return p.newHistogram(stream)
}

func (p *Provider) newHistogram(stream sdk.Stream, labelValues ...string) metrics.Histogram {
	p.mu.Lock()
	defer p.mu.Unlock()

	name := stream.Name
	k := keyName(name, labelValues...)
	m := p.meterProvider.Meter(name)

	if _, ok := p.histograms[k]; !ok {
		h, _ := m.Float64Histogram(name)
		p.viewCache.Store(stream)

		p.histograms[k] = &Histogram{
			Float64Histogram: h,
			stream:           stream,
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
	return h.p.newHistogram(h.stream, lvs...)
}

// Observe implements metrics.Histogram.
func (h *Histogram) Observe(value float64) {
	h.Record(h.p.cfg.ctx, value, metric.WithAttributeSet(h.attributes))
}

// NewCardinalityCounter implements metrics.Provider.
func (p *Provider) NewCardinalityCounter(name string) xmetrics.CardinalityCounter {
	return xmetrics.NewHLLCounter(name)
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
func makeAttributes(labels []string) attribute.Set {
	attributes := make([]attribute.KeyValue, len(labels))
	if len(labels)%2 != 0 {
		labels = append(labels, "unknown")
	}

	for i := 0; i < len(labels); i += 2 {
		attributes = append(attributes, attribute.String(labels[i], labels[i+1]))
	}

	return attribute.NewSet(attributes...)
}
