package metrics

import (
	"sync"

	kitmetrics "github.com/go-kit/kit/metrics"
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
	_ Registry = &registry{}
	_ Registry = &prefixedRegistry{}
)

// registry is the base implementation of a Registry.
type registry struct {
	sync.Mutex
	p          Provider
	counters   map[string]kitmetrics.Counter
	gauges     map[string]kitmetrics.Gauge
	histograms map[string]kitmetrics.Histogram
}

// NewRegistry creates a Registry given a metrics.Provider.
func NewRegistry(p Provider) Registry {
	return &registry{
		p:          p,
		counters:   make(map[string]kitmetrics.Counter),
		gauges:     make(map[string]kitmetrics.Gauge),
		histograms: make(map[string]kitmetrics.Histogram),
	}
}

// GetOrRegisterCounter creates or finds the Counter given a name.
func (r *registry) GetOrRegisterCounter(name string) kitmetrics.Counter {
	r.Lock()
	defer r.Unlock()

	if r.counters[name] == nil {
		r.counters[name] = r.p.NewCounter(name)
	}
	return r.counters[name]
}

// GetOrRegisterGauge creates or finds the Gauge given a name.
func (r *registry) GetOrRegisterGauge(name string) kitmetrics.Gauge {
	r.Lock()
	defer r.Unlock()

	if r.gauges[name] == nil {
		r.gauges[name] = r.p.NewGauge(name)
	}
	return r.gauges[name]
}

// GetOrRegisterHistogram creates or finds the Histogram given a name.
func (r *registry) GetOrRegisterHistogram(name string, buckets int) kitmetrics.Histogram {
	r.Lock()
	defer r.Unlock()

	if r.histograms[name] == nil {
		r.histograms[name] = r.p.NewHistogram(name, buckets)
	}
	return r.histograms[name]
}

// RegistryWithPrefix wraps an existing registry and returns a new registry
// with all the GetOrRegister functions automatically prefixed.
func RegistryWithPrefix(r Registry, prefix string) Registry {
	return &prefixedRegistry{r: r, prefix: prefix}
}

// prefixedRegistry contains a reference to the original Registry and thus
// shares the same state with the parent registry.
type prefixedRegistry struct {
	r      Registry
	prefix string
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
