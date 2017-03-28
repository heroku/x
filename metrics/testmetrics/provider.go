package testmetrics

import (
	"math"
	"reflect"
	"testing"

	"github.com/go-kit/kit/metrics"
)

// Provider collects registered metrics for testing.
type Provider struct {
	t *testing.T

	counters   map[string]*Counter
	gauges     map[string]*Gauge
	histograms map[string]*Histogram
}

// NewProvider constructs a test provider which can later be checked.
func NewProvider(t *testing.T) *Provider {
	return &Provider{
		t:          t,
		counters:   make(map[string]*Counter),
		histograms: make(map[string]*Histogram),
		gauges:     make(map[string]*Gauge),
	}
}

// NewCounter implements go-kit's Provider interface.
func (p *Provider) NewCounter(name string) metrics.Counter {
	if c, ok := p.counters[name]; ok {
		return c
	}

	c := &Counter{}
	p.counters[name] = c
	return c
}

// NewGauge implements go-kit's Provider interface.
func (p *Provider) NewGauge(name string) metrics.Gauge {
	if g, ok := p.gauges[name]; ok {
		return g
	}

	g := &Gauge{}
	p.gauges[name] = g
	return g
}

// NewHistogram implements go-kit's Provider interface.
func (p *Provider) NewHistogram(name string, _ int) metrics.Histogram {
	if h, ok := p.histograms[name]; ok {
		return h
	}

	h := &Histogram{}
	p.histograms[name] = h
	return h
}

// CheckCounter checks that there is a registered counter
// with the name and value provided.
func (p *Provider) CheckCounter(name string, v float64) {
	c, ok := p.counters[name]
	if !ok {
		p.t.Fatalf("no counter named %v", name)
	}

	if c.value != v {
		p.t.Fatalf("%v = %v, want %v", name, c.value, v)
	}
}

// CheckObservationsMinMax checks that there is a histogram
// with the name and that the values all fall within the min/max range.
func (p *Provider) CheckObservationsMinMax(name string, min, max float64) {
	h, ok := p.histograms[name]
	if !ok {
		p.t.Fatalf("no histogram named %v", name)
	}

	for _, o := range h.getObservations() {
		if o < min || o > max {
			p.t.Fatalf("Got %f want %f..%f ", o, min, max)
		}
	}
}

// CheckObservations checks that there is a histogram
// with the name and observations provided.
func (p *Provider) CheckObservations(name string, obs ...float64) {
	h, ok := p.histograms[name]
	if !ok {
		p.t.Fatalf("no histogram named %v", name)
	}

	if !reflect.DeepEqual(h.observations, obs) {
		p.t.Fatalf("%v = %v, want %v", name, h.observations, obs)
	}
}

// CheckObservationCount checks that there is a histogram
// with the name and number of observations provided.
func (p *Provider) CheckObservationCount(name string, n int) {
	h, ok := p.histograms[name]
	if !ok {
		p.t.Fatalf("no histogram named %v", name)
	}

	if len(h.observations) != n {
		p.t.Fatalf("len(%v) = %v, want %v", name, len(h.observations), n)
	}
}

// CheckObservationAlmostEqual is used to compare a specific element in a histogram.
// An epsilon is used because exactly matching floating point numbers is usually quite difficult.
func (p *Provider) CheckObservationAlmostEqual(name string, n int, value, epsilon float64) {
	h, ok := p.histograms[name]
	if !ok {
		p.t.Fatalf("no histogram named %v", name)
	}
	if len(h.observations) <= n {
		p.t.Fatalf("len(%v) = %v, want < %v", name, len(h.observations), n)
	}

	if math.Abs(h.observations[n]-value) >= epsilon {
		p.t.Fatalf("%v = %v, want %v", name, h.observations[n], value)
	}
}

// CheckGauge checks that there is a registered counter
// with the name and value provided.
func (p *Provider) CheckGauge(name string, v float64) {
	g, ok := p.gauges[name]
	if !ok {
		p.t.Fatalf("no gauge named %v", name)
	}

	if g.value != v {
		p.t.Fatalf("%v = %v, want %v", name, g.value, v)
	}
}
