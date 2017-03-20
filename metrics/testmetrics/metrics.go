package testmetrics

import "github.com/go-kit/kit/metrics"

// Counter accumulates a value based on Add calls.
type Counter struct{ value float64 }

// Add implements the metrics.Counter interface.
func (c *Counter) Add(delta float64) { c.value += delta }

// With implements the metrics.Counter interface.
func (c *Counter) With(...string) metrics.Counter { return c }

// Histogram collects observations without computing quantiles
// so the observations can be checked by tests.
type Histogram struct{ observations []float64 }

// Observe implements the metrics.Histogram interface.
func (h *Histogram) Observe(v float64) { h.observations = append(h.observations, v) }

// With implements the metrics.Histogram interface.
func (h *Histogram) With(...string) metrics.Histogram { return h }
