// Package l2met provides an l2met backend for metrics. Metrics are batched
// and emitted. For more information, see
// https://github.com/ryandotsmith/l2met/wiki/Usage
//
// L2met does not have a native understanding of metric parameterization, so
// label values not supported. Use distinct metrics for each unique
// combination of label values.
package l2met

import (
	"fmt"
	"io"
	"strconv"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/generic"
)

// L2met receives metrics observations and writes them to the log. Create a
// L2met object, use it to create metrics, and pass those metrics as
// dependencies to the components that will use them.
//
// All metrics are buffered until WriteTo is called. Counters and gauges are
// aggregated into a single observation per timeseries per write. Histograms
// are exploded into per-quantile gauges and reported once per write.
//
// To regularly report metrics to an io.Writer, use the WriteLoop helper
// method.
type L2met struct {
	mu         sync.RWMutex
	prefix     string
	counters   map[string]*Counter
	gauges     map[string]*Gauge
	histograms map[string]*Histogram
	logger     logrus.FieldLogger
}

// New returns a L2met object that may be used to create metrics. Prefix is
// applied to all created metrics. Callers must ensure that regular calls to
// WriteTo are performed, either manually or with one of the helper methods.
//
// The provided logger is used to log errors encountered while writing metrics
// in WriteLoop.
func New(prefix string) *L2met {
	return &L2met{
		prefix:     prefix,
		counters:   map[string]*Counter{},
		gauges:     map[string]*Gauge{},
		histograms: map[string]*Histogram{},
	}
}

// NewCounter returns a counter. Observations are aggregated and emitted once
// per write invocation.
func (l *L2met) NewCounter(name string) metrics.Counter {
	fullName := l.prefix + name

	l.mu.Lock()
	defer l.mu.Unlock()

	if c, ok := l.counters[fullName]; ok {
		return c
	}

	c := NewCounter(fullName)
	l.counters[fullName] = c
	return c
}

// NewGauge returns a gauge. Observations are aggregated and emitted once per
// write invocation.
func (l *L2met) NewGauge(name string) metrics.Gauge {
	fullName := l.prefix + name

	l.mu.Lock()
	defer l.mu.Unlock()

	if g, ok := l.gauges[fullName]; ok {
		return g
	}

	g := NewGauge(fullName)
	l.gauges[fullName] = g
	return g
}

// NewHistogram returns a histogram. Observations are aggregated and emitted
// as per-quantile gauges, once per write invocation. 50 is a good default
// value for buckets.
func (l *L2met) NewHistogram(name string, buckets int) metrics.Histogram {
	fullName := l.prefix + name

	l.mu.Lock()
	defer l.mu.Unlock()

	if h, ok := l.histograms[fullName]; ok {
		return h
	}

	h := NewHistogram(fullName, buckets)
	l.histograms[fullName] = h
	return h
}

// WriteTo flushes the buffered content of the metrics to the writer, in
// Graphite plaintext format. WriteTo abides best-effort semantics, so
// observations are lost if there is a problem with the write. Clients should
// be sure to call WriteTo regularly, typically through the WriteLoop.
func (l *L2met) WriteTo(w io.Writer) (count int64, err error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	for name, c := range l.counters {
		n, err := fmt.Fprintf(w, "count#%s=%s\n", name, formatFloat(c.c.ValueReset()))
		if err != nil {
			return count, err
		}
		count += int64(n)
	}

	for name, g := range l.gauges {
		n, err := fmt.Fprintf(w, "measure#%s=%s\n", name, formatFloat(g.g.Value()))
		if err != nil {
			return count, err
		}
		count += int64(n)
	}

	for name, h := range l.histograms {
		oh := h.reset()

		for _, p := range []struct {
			s string
			f float64
		}{
			{"50", 0.50},
			{"90", 0.90},
			{"95", 0.95},
			{"99", 0.99},
		} {
			v := oh.Quantile(p.f)
			// no measurement to report
			if v < 0 {
				continue
			}

			n, err := fmt.Fprintf(w, "measure#%s.perc%s=%s\n", name, p.s, formatFloat(v))
			if err != nil {
				return count, err
			}
			count += int64(n)
		}
	}

	return count, err
}

// formatFloat formats f as an exponentless value with 9 decimal points of
// precision.
func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', 9, 64)
}

// Counter is a l2met counter metric.
type Counter struct {
	c *generic.Counter
}

// NewCounter returns a new usable counter metric.
func NewCounter(name string) *Counter {
	return &Counter{generic.NewCounter(name)}
}

// With is a no-op.
func (c *Counter) With(...string) metrics.Counter {
	return c
}

// Add implements counter.
func (c *Counter) Add(delta float64) {
	c.c.Add(delta)
}

// Gauge is a l2met gauge metric.
type Gauge struct {
	g *generic.Gauge
}

// NewGauge returns a new usable Gauge metric.
func NewGauge(name string) *Gauge {
	return &Gauge{generic.NewGauge(name)}
}

// With is a no-op.
func (g *Gauge) With(...string) metrics.Gauge {
	return g
}

// Set implements gauge.
func (g *Gauge) Set(value float64) {
	g.g.Set(value)
}

// Add implements metrics.Gauge.
func (g *Gauge) Add(delta float64) {
	g.g.Add(delta)
}

// Histogram is a l2met histogram metric. Observations are bucketed into
// per-quantile gauges.
type Histogram struct {
	name    string
	buckets int

	mu sync.RWMutex
	h  *generic.Histogram
}

// NewHistogram returns a new usable Histogram metric.
func NewHistogram(name string, buckets int) *Histogram {
	h := &Histogram{
		name:    name,
		buckets: buckets,
	}
	h.reset()
	return h
}

// reset creates a new generic.Histogram and replaces the underlying
// histogram with it. It returns the previous histogram.
func (h *Histogram) reset() *generic.Histogram {
	h.mu.Lock()
	defer h.mu.Unlock()
	oh := h.h
	h.h = generic.NewHistogram(h.name, h.buckets)
	return oh
}

// With is a no-op.
func (h *Histogram) With(...string) metrics.Histogram {
	return h
}

// Observe implements histogram.
func (h *Histogram) Observe(value float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.h.Observe(value)
}

// Quantile returns the value of the quantile q, 0.0 < q < 1.0.
func (h *Histogram) Quantile(q float64) float64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.h.Quantile(q)
}
