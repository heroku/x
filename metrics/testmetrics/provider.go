package testmetrics

import (
	"math"
	"reflect"
	"sort"
	"sync"
	"testing"

	"github.com/go-kit/kit/metrics"
)

// Provider collects registered metrics for testing.
type Provider struct {
	t *testing.T

	sync.Mutex
	counters   map[string]*Counter
	gauges     map[string]*Gauge
	histograms map[string]*Histogram
	stopped    bool
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

// Stop makes it Provider compliant.
func (p *Provider) Stop() {
	p.stopped = true
}

// NewCounter implements go-kit's Provider interface.
func (p *Provider) NewCounter(name string) metrics.Counter {
	p.Lock()
	defer p.Unlock()

	if _, ok := p.counters[name]; ok {
		p.t.Errorf("NewCounter(%s) called, already existing", name)
	}

	c := &Counter{}
	p.counters[name] = c
	return c
}

// NewGauge implements go-kit's Provider interface.
func (p *Provider) NewGauge(name string) metrics.Gauge {
	p.Lock()
	defer p.Unlock()

	if _, ok := p.gauges[name]; ok {
		p.t.Errorf("NewGauge(%s) called, already existing", name)
	}

	g := &Gauge{}
	p.gauges[name] = g
	return g
}

// NewHistogram implements go-kit's Provider interface.
func (p *Provider) NewHistogram(name string, _ int) metrics.Histogram {
	p.Lock()
	defer p.Unlock()

	if _, ok := p.histograms[name]; ok {
		p.t.Errorf("NewHistogram(%s) called, already existing", name)
	}

	h := &Histogram{}
	p.histograms[name] = h
	return h
}

// CheckCounter checks that there is a registered counter
// with the name and value provided.
func (p *Provider) CheckCounter(name string, v float64) {
	p.Lock()
	defer p.Unlock()

	c, ok := p.counters[name]
	if !ok {
		p.t.Fatalf("no counter named %v", name)
	}

	if c.getValue() != v {
		p.t.Fatalf("%v = %v, want %v", name, c.value, v)
	}
}

// CheckObservationsMinMax checks that there is a histogram
// with the name and that the values all fall within the min/max range.
func (p *Provider) CheckObservationsMinMax(name string, min, max float64) {
	for _, o := range p.getObservations(name) {
		if o < min || o > max {
			p.t.Fatalf("Got %f want %f..%f ", o, min, max)
		}
	}
}

// CheckObservations checks that there is a histogram
// with the name and observations provided.
func (p *Provider) CheckObservations(name string, obs ...float64) {
	observations := p.getObservations(name)
	if !reflect.DeepEqual(observations, obs) {
		p.t.Fatalf("%v = %v, want %v", name, observations, obs)
	}
}

// CheckObservationsMatch checks that there is a histogram with the name and
// observations provided, ignoring order.
func (p *Provider) CheckObservationsMatch(name string, obs ...float64) {
	observations := p.getObservations(name)

	got := make([]float64, len(observations))
	copy(got, observations)

	want := make([]float64, len(obs))
	copy(want, obs)

	sort.Float64s(got)
	sort.Float64s(want)

	if !reflect.DeepEqual(want, got) {
		p.t.Fatalf("%v = %v, want %v", name, want, got)
	}
}

// CheckObservationCount checks that there is a histogram
// with the name and number of observations provided.
func (p *Provider) CheckObservationCount(name string, n int) {
	observations := p.getObservations(name)

	if len(observations) != n {
		p.t.Fatalf("len(%v) = %v, want %v", name, len(observations), n)
	}
}

// CheckObservationAlmostEqual is used to compare a specific element in a histogram.
// An epsilon is used because exactly matching floating point numbers is usually quite difficult.
func (p *Provider) CheckObservationAlmostEqual(name string, n int, value, epsilon float64) {
	observations := p.getObservations(name)
	if len(observations) <= n {
		p.t.Fatalf("len(%v) = %v, want < %v", name, len(observations), n)
	}

	if math.Abs(observations[n]-value) >= epsilon {
		p.t.Fatalf("%v = %v, want %v", name, observations[n], value)
	}
}

func (p *Provider) getObservations(name string) []float64 {
	p.Lock()
	defer p.Unlock()

	h, ok := p.histograms[name]
	if !ok {
		p.t.Fatalf("no histogram named %v", name)
	}

	return h.getObservations()
}

// CheckGauge checks that there is a registered gauge
// with the name and value provided.
func (p *Provider) CheckGauge(name string, v float64) {
	p.Lock()
	defer p.Unlock()

	g, ok := p.gauges[name]
	if !ok {
		p.t.Fatalf("no gauge named %v", name)
	}
	actualV := g.getValue()
	if actualV != v {
		p.t.Fatalf("%v = %v, want %v", name, actualV, v)
	}
}

// CheckGaugeNonZero checks that there is a registered gauge
// with the name provided whose value is != 0.
func (p *Provider) CheckGaugeNonZero(name string) {
	p.Lock()
	defer p.Unlock()

	g, ok := p.gauges[name]
	if !ok {
		p.t.Fatalf("no gauge named %v", name)
	}

	if g.value == 0 {
		p.t.Fatalf("%v = %v, want non-zero", name, g.value)
	}
}

// CheckStopped verifies that a provider has been Stop'd.
func (p *Provider) CheckStopped() {
	if !p.stopped {
		p.t.Fatal("provider is not stopped")
	}
}
