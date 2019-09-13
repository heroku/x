// Package testmetrics is for testing provider metrics
// with a test Provider that adheres to the Provider interface
package testmetrics

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/go-kit/kit/metrics"

	xmetrics "github.com/heroku/x/go-kit/metrics"
)

// Provider collects registered metrics for testing.
type Provider struct {
	t *testing.T

	sync.Mutex
	counters     map[string]*Counter
	gauges       map[string]*Gauge
	histograms   map[string]*Histogram
	cardCounters map[string]*xmetrics.HLLCounter
	stopped      bool
}

// NewProvider constructs a test provider which can later be checked.
func NewProvider(t *testing.T) *Provider {
	return &Provider{
		t:            t,
		counters:     make(map[string]*Counter),
		histograms:   make(map[string]*Histogram),
		gauges:       make(map[string]*Gauge),
		cardCounters: make(map[string]*xmetrics.HLLCounter),
	}
}

// Stop makes it Provider compliant.
func (p *Provider) Stop() {
	p.stopped = true
}

// NewCounter implements go-kit's Provider interface.
func (p *Provider) NewCounter(name string) metrics.Counter {
	return p.newCounter(name)
}

func (p *Provider) newCounter(name string, labelValues ...string) metrics.Counter {
	p.Lock()
	defer p.Unlock()

	k := p.keyFor(name, labelValues...)
	if _, ok := p.counters[k]; !ok {
		p.counters[k] = &Counter{name: name, p: p, labelValues: labelValues}
	}
	return p.counters[k]
}

// NewGauge implements go-kit's Provider interface.
func (p *Provider) NewGauge(name string) metrics.Gauge {
	return p.newGauge(name)
}

func (p *Provider) newGauge(name string, labelValues ...string) metrics.Gauge {
	p.Lock()
	defer p.Unlock()

	k := p.keyFor(name, labelValues...)
	if _, ok := p.gauges[k]; !ok {
		p.gauges[k] = &Gauge{name: name, p: p, labelValues: labelValues}
	}
	return p.gauges[k]
}

// NewHistogram implements go-kit's Provider interface.
func (p *Provider) NewHistogram(name string, _ int) metrics.Histogram {
	return p.newHistogram(name)
}

func (p *Provider) newHistogram(name string, labelValues ...string) metrics.Histogram {
	p.Lock()
	defer p.Unlock()

	k := p.keyFor(name, labelValues...)
	if _, ok := p.histograms[k]; !ok {
		p.histograms[k] = &Histogram{name: name, p: p, labelValues: labelValues}
	}
	return p.histograms[k]
}

// NewCardinalityCounter implements metrics.Provider.
func (p *Provider) NewCardinalityCounter(name string) xmetrics.CardinalityCounter {
	p.Lock()
	defer p.Unlock()

	if _, ok := p.cardCounters[name]; !ok {
		p.cardCounters[name] = xmetrics.NewHLLCounter(name)
	}
	return p.cardCounters[name]
}

// CheckCounter checks that there is a registered counter
// with the name and value provided.
func (p *Provider) CheckCounter(name string, v float64, labelValues ...string) {
	p.t.Helper()

	p.Lock()
	defer p.Unlock()

	k := p.keyFor(name, labelValues...)
	c, ok := p.counters[k]
	if !ok {
		keys := make([]string, 0, len(p.counters))
		for k := range p.counters {
			keys = append(keys, k)
		}
		available := strings.Join(keys, "\n")
		p.t.Fatalf("no counter named %s out of available counters: \n%s", k, available)
	}

	if c.getValue() != v {
		p.t.Fatalf("%v = %v, want %v", name, c.value, v)
	}

	if len(labelValues) > 0 && !reflect.DeepEqual(labelValues, c.labelValues) {
		p.t.Fatalf("want counter label values: %#v, got %#v", labelValues, c.labelValues)
	}
}

// PrintCounterValue prints the value of the specified counter and their current counts
func (p *Provider) PrintCounterValue(name string) {
	p.t.Helper()

	p.Lock()
	defer p.Unlock()

	fmt.Printf("%s: %v\n", name, p.counters[name].getValue())
}

// CheckNoCounter checks that there is no registered counter with the name
// provided.
func (p *Provider) CheckNoCounter(name string, labelValues ...string) {
	p.t.Helper()

	p.Lock()
	defer p.Unlock()

	k := p.keyFor(name, labelValues...)
	_, ok := p.counters[k]
	if ok {
		p.t.Fatalf("a counter named %s was found", k)
	}
}

// CheckObservationsMinMax checks that there is a histogram
// with the name and that the values all fall within the min/max range.
func (p *Provider) CheckObservationsMinMax(name string, min, max float64, labelValues ...string) {
	p.t.Helper()

	for _, o := range p.getObservations(name, labelValues...) {
		if o < min || o > max {
			p.t.Fatalf("got %f want %f..%f ", o, min, max)
		}
	}
}

// CheckObservations checks that there is a histogram
// with the name and observations provided.
func (p *Provider) CheckObservations(name string, obs []float64, labelValues ...string) {
	p.t.Helper()

	observations := p.getObservations(name, labelValues...)
	if !reflect.DeepEqual(observations, obs) {
		p.t.Fatalf("%v = %v, want %v", p.keyFor(name, labelValues...), observations, obs)
	}
}

// CheckObservationsMatch checks that there is a histogram with the name and
// observations provided, ignoring order.
func (p *Provider) CheckObservationsMatch(name string, obs []float64, labelValues ...string) {
	p.t.Helper()

	observations := p.getObservations(name, labelValues...)

	got := make([]float64, len(observations))
	copy(got, observations)

	want := make([]float64, len(obs))
	copy(want, obs)

	sort.Float64s(got)
	sort.Float64s(want)

	if !reflect.DeepEqual(want, got) {
		p.t.Fatalf("%v = %v, want %v", p.keyFor(name, labelValues...), want, got)
	}
}

// CheckObservationCount checks that there is a histogram
// with the name and number of observations provided.
func (p *Provider) CheckObservationCount(name string, n int, labelValues ...string) {
	p.t.Helper()

	observations := p.getObservations(name, labelValues...)

	if len(observations) != n {
		p.t.Fatalf("len(%v) = %v, want %v", p.keyFor(name, labelValues...), len(observations), n)
	}
}

func (p *Provider) getObservations(name string, labelValues ...string) []float64 {
	p.t.Helper()

	p.Lock()
	defer p.Unlock()

	k := p.keyFor(name, labelValues...)
	h, ok := p.histograms[k]
	if !ok {
		keys := make([]string, 0, len(p.histograms))
		for k := range p.histograms {
			keys = append(keys, k)
		}
		available := strings.Join(keys, "\n")
		p.t.Fatalf("no histogram named %s out available histograms: \n%s", k, available)
	}

	return h.getObservations()
}

// CheckGauge checks that there is a registered gauge
// with the name and value provided.
func (p *Provider) CheckGauge(name string, v float64, labelValues ...string) {
	p.t.Helper()

	p.Lock()
	defer p.Unlock()

	k := p.keyFor(name, labelValues...)
	g, ok := p.gauges[k]
	if !ok {
		keys := make([]string, 0, len(p.gauges))
		for k := range p.gauges {
			keys = append(keys, k)
		}
		available := strings.Join(keys, "\n")
		p.t.Fatalf("no gauge named %s out of available gauges: \n%s", k, available)
	}
	actualV := g.getValue()
	if actualV != v {
		p.t.Fatalf("%v = %v, want %v", k, actualV, v)
	}
}

// CheckGaugeNonZero checks that there is a registered gauge
// with the name provided whose value is != 0.
func (p *Provider) CheckGaugeNonZero(name string, labelValues ...string) {
	p.t.Helper()

	p.Lock()
	defer p.Unlock()

	k := p.keyFor(name, labelValues...)
	g, ok := p.gauges[k]
	if !ok {
		p.t.Fatalf("no gauge named %v", k)
	}

	if g.value == 0 {
		p.t.Fatalf("%v = %v, want non-zero", k, g.value)
	}
}

// CheckNoGauge checks that there is no registered gauge with the name
// provided.
func (p *Provider) CheckNoGauge(name string, labelValues ...string) {
	p.t.Helper()

	p.Lock()
	defer p.Unlock()

	k := p.keyFor(name, labelValues...)
	_, ok := p.gauges[k]
	if ok {
		p.t.Fatalf("a gauge named %s was found", k)
	}
}

// CheckStopped verifies that a provider has been Stop'd.
func (p *Provider) CheckStopped() {
	p.t.Helper()

	if !p.stopped {
		p.t.Fatal("provider is not stopped")
	}
}

// CheckCardinalityCounter checks that there is a registered cardinality
// counter with the name and estimate provided.
func (p *Provider) CheckCardinalityCounter(name string, estimate uint64) {
	p.t.Helper()

	p.Lock()
	defer p.Unlock()

	cc, ok := p.cardCounters[name]
	if !ok {
		keys := make([]string, 0, len(p.cardCounters))
		for k := range p.cardCounters {
			keys = append(keys, k)
		}
		available := strings.Join(keys, "\n")
		p.t.Fatalf("no cardinality counter named %s out of available cardinality counter: \n%s", name, available)
	}
	actualEstimate := cc.Estimate()
	if actualEstimate != estimate {
		p.t.Fatalf("%v = %v, want %v", name, actualEstimate, estimate)
	}
}

func (p *Provider) keyFor(name string, labelValues ...string) string {
	if len(labelValues) == 0 {
		return name
	}
	return name + "." + strings.Join(labelValues, ":")
}
