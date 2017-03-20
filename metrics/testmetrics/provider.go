package testmetrics

import (
	"reflect"
	"testing"

	"github.com/go-kit/kit/metrics"
)

// Provider collects registered metrics for testing.
type Provider struct {
	t *testing.T

	counters   map[string]*Counter
	histograms map[string]*Histogram
}

// NewProvider constructs a test provider which can later be checked.
func NewProvider(t *testing.T) *Provider {
	return &Provider{
		t:          t,
		counters:   make(map[string]*Counter),
		histograms: make(map[string]*Histogram),
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
