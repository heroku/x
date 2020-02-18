// Package metricsregistry provides utilities for working with dynamically created metrics.
package metricsregistry

import (
	"sync"

	kitmetrics "github.com/go-kit/kit/metrics"

	"github.com/heroku/x/go-kit/metrics"
)

// A Registry holds references to a set of metrics by name. It's guaranteed
// to keep returning the same metric given the same name and type. All
// implementations are also required to be thread safe.
type Registry interface {
	GetOrRegisterCounter(name string) kitmetrics.Counter
	GetOrRegisterGauge(name string) kitmetrics.Gauge
	GetOrRegisterHistogram(name string, buckets int) kitmetrics.Histogram
}

// simple compile time checks for interface compliance.
var (
	_ Registry = &basicRegistry{}
	_ Registry = &prefixedRegistry{}
)

// registry is the base implementation of a Registry.
type basicRegistry struct {
	sync.Mutex
	p          metrics.Provider
	counters   map[string]kitmetrics.Counter
	gauges     map[string]kitmetrics.Gauge
	histograms map[string]kitmetrics.Histogram
}

// New creates a Registry given a metrics.Provider.
func New(p metrics.Provider) Registry {
	return &basicRegistry{
		p:          p,
		counters:   make(map[string]kitmetrics.Counter),
		gauges:     make(map[string]kitmetrics.Gauge),
		histograms: make(map[string]kitmetrics.Histogram),
	}
}

// GetOrRegisterCounter creates or finds the Counter given a name.
func (r *basicRegistry) GetOrRegisterCounter(name string) kitmetrics.Counter {
	r.Lock()
	defer r.Unlock()

	if r.counters[name] == nil {
		r.counters[name] = r.p.NewCounter(name)
	}
	return r.counters[name]
}

// GetOrRegisterGauge creates or finds the Gauge given a name.
func (r *basicRegistry) GetOrRegisterGauge(name string) kitmetrics.Gauge {
	r.Lock()
	defer r.Unlock()

	if r.gauges[name] == nil {
		r.gauges[name] = r.p.NewGauge(name)
	}
	return r.gauges[name]
}

// GetOrRegisterHistogram creates or finds the Histogram given a name.
func (r *basicRegistry) GetOrRegisterHistogram(name string, buckets int) kitmetrics.Histogram {
	r.Lock()
	defer r.Unlock()

	if r.histograms[name] == nil {
		r.histograms[name] = r.p.NewHistogram(name, buckets)
	}
	return r.histograms[name]
}

// prefixedRegistry contains a reference to the original Registry and thus
// shares the same state with the parent registry.
type prefixedRegistry struct {
	r      Registry
	prefix string
}

// NewPrefixed creates a new Registry backed by r
// with all created metric names prefixed with prefix + ".".
func NewPrefixed(r Registry, prefix string) Registry {
	return &prefixedRegistry{
		r:      r,
		prefix: prefix,
	}
}

// GetOrRegisterCounter creates or finds the Counter given a name.
func (r *prefixedRegistry) GetOrRegisterCounter(name string) kitmetrics.Counter {
	return r.r.GetOrRegisterCounter(r.prefixedName(name))
}

// GetOrRegisterGauge creates or finds the Gauge given a name.
func (r *prefixedRegistry) GetOrRegisterGauge(name string) kitmetrics.Gauge {
	return r.r.GetOrRegisterGauge(r.prefixedName(name))
}

// GetOrRegisterHistogram creates or finds the Histogram given a name.
func (r *prefixedRegistry) GetOrRegisterHistogram(name string, buckets int) kitmetrics.Histogram {
	return r.r.GetOrRegisterHistogram(r.prefixedName(name), buckets)
}

func (r *prefixedRegistry) prefixedName(name string) string {
	return r.prefix + "." + name
}
