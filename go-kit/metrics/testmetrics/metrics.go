package testmetrics

import (
	"sync"

	"github.com/go-kit/kit/metrics"
)

// Counter accumulates a value based on Add calls.
type Counter struct {
	name        string
	p           *Provider
	labelValues []string
	value       float64
	sync.RWMutex
}

// Add implements the metrics.Counter interface.
func (c *Counter) Add(delta float64) {
	c.Lock()
	defer c.Unlock()
	c.value += delta
}

// With implements the metrics.Counter interface.
func (c *Counter) With(labelValues ...string) metrics.Counter {
	lvs := append(append([]string(nil), c.labelValues...), labelValues...)
	return c.p.newCounter(c.name, lvs...)
}

func (c *Counter) getValue() float64 {
	c.RLock()
	defer c.RUnlock()
	return c.value
}

// Gauge stores a value based on Add/Set calls.
type Gauge struct {
	name        string
	p           *Provider
	labelValues []string
	value       float64
	sync.RWMutex
}

// Add implements the metrics.Gauge interface.
func (g *Gauge) Add(delta float64) {
	g.Lock()
	defer g.Unlock()
	g.value += delta
}

// Set implements the metrics.Gauge interface.
func (g *Gauge) Set(v float64) {
	g.Lock()
	defer g.Unlock()
	g.value = v
}

// With implements the metrics.Gauge interface.
func (g *Gauge) With(labelValues ...string) metrics.Gauge {
	lvs := append(append([]string(nil), g.labelValues...), labelValues...)
	return g.p.newGauge(g.name, lvs...)
}

func (g *Gauge) getValue() float64 {
	g.RLock()
	defer g.RUnlock()
	return g.value
}

// Histogram collects observations without computing quantiles
// so the observations can be checked by tests.
type Histogram struct {
	name         string
	p            *Provider
	labelValues  []string
	observations []float64
	sync.RWMutex
}

func (h *Histogram) getObservations() []float64 {
	h.RLock()
	defer h.RUnlock()

	o := h.observations
	return o
}

// Observe implements the metrics.Histogram interface.
func (h *Histogram) Observe(v float64) {
	h.Lock()
	defer h.Unlock()
	h.observations = append(h.observations, v)
}

// With implements the metrics.Histogram interface.
func (h *Histogram) With(labelValues ...string) metrics.Histogram {
	lvs := append(append([]string(nil), h.labelValues...), labelValues...)
	return h.p.newHistogram(h.name, lvs...)
}
