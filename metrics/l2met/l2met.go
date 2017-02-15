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
	"time"

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
	logger     *logrus.Logger
}

// New returns a L2met object that may be used to create metrics. Prefix is
// applied to all created metrics. Callers must ensure that regular calls to
// WriteTo are performed, either manually or with one of the helper methods.
func New(prefix string, logger *logrus.Logger) *L2met {
	return &L2met{
		prefix:     prefix,
		counters:   map[string]*Counter{},
		gauges:     map[string]*Gauge{},
		histograms: map[string]*Histogram{},
		logger:     logger,
	}
}

// NewCounter returns a counter. Observations are aggregated and emitted once
// per write invocation.
func (l *L2met) NewCounter(name string) *Counter {
	c := NewCounter(l.prefix + name)
	l.mu.Lock()
	l.counters[l.prefix+name] = c
	l.mu.Unlock()
	return c
}

// NewGauge returns a gauge. Observations are aggregated and emitted once per
// write invocation.
func (l *L2met) NewGauge(name string) *Gauge {
	ga := NewGauge(l.prefix + name)
	l.mu.Lock()
	l.gauges[l.prefix+name] = ga
	l.mu.Unlock()
	return ga
}

// NewHistogram returns a histogram. Observations are aggregated and emitted
// as per-quantile gauges, once per write invocation. 50 is a good default
// value for buckets.
func (l *L2met) NewHistogram(name string, buckets int) *Histogram {
	h := NewHistogram(l.prefix+name, buckets)
	l.mu.Lock()
	l.histograms[l.prefix+name] = h
	l.mu.Unlock()
	return h
}

// WriteLoop is a helper method that invokes WriteTo to the passed writer every
// time the passed channel fires. This method blocks until the channel is
// closed, so clients probably want to run it in its own goroutine. For typical
// usage, create a time.Ticker and pass its C channel to this method.
func (l *L2met) WriteLoop(c <-chan time.Time, w io.Writer) {
	for range c {
		if _, err := l.WriteTo(w); err != nil {
			l.logger.WithError(err).Error()
		}
	}
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
		n, err := fmt.Fprintf(w, "sample#%s=%s\n", name, formatFloat(g.g.Value()))
		if err != nil {
			return count, err
		}
		count += int64(n)
	}

	for name, h := range l.histograms {
		for _, p := range []struct {
			s string
			f float64
		}{
			{"50", 0.50},
			{"90", 0.90},
			{"95", 0.95},
			{"99", 0.99},
		} {
			n, err := fmt.Fprintf(w, "sample#%s.perc%s=%s\n", name, p.s, formatFloat(h.h.Quantile(p.f)))
			if err != nil {
				return count, err
			}
			count += int64(n)
		}
	}

	return count, err
}

// formatFloat formats f as an exponentless value with the minimum precision
// necessary to identify the value uniquely.
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
	h *generic.Histogram
}

// NewHistogram returns a new usable Histogram metric.
func NewHistogram(name string, buckets int) *Histogram {
	return &Histogram{generic.NewHistogram(name, buckets)}
}

// With is a no-op.
func (h *Histogram) With(...string) metrics.Histogram {
	return h
}

// Observe implements histogram.
func (h *Histogram) Observe(value float64) {
	h.h.Observe(value)
}
