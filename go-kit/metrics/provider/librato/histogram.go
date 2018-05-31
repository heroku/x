/* Copyright (c) 2018 Salesforce
 * All rights reserved.
 * Licensed under the BSD 3-Clause license.
 * For full license text, see LICENSE.txt file in the repo root  or https://opensource.org/licenses/BSD-3-Clause
 */

package librato

import (
	"sync"

	"github.com/VividCortex/gohistogram"
	kmetrics "github.com/go-kit/kit/metrics"
)

var (
	_ kmetrics.Histogram = &Histogram{}
)

// Histogram adapts go-kit/Heroku/Librato's ideas of histograms. It
// reports p99, p95 and p50 values as gauges in addition to a gauge for the
// histogram itself.
type Histogram struct {
	buckets          int
	name             string
	percentilePrefix string

	mu sync.RWMutex
	// I would prefer to use hdrhistogram, but that's incompatible with the
	// go-metrics Histogram interface (int64 vs float64).
	h                    gohistogram.Histogram
	sum, min, max, sumsq float64
	count                int64
}

// the json marshalers for the histograms 4 different gauges
func (h *Histogram) measures(period float64) []gauge {
	h.mu.Lock()
	if h.count == 0 {
		h.mu.Unlock()
		return nil
	}
	count := h.count
	sum := h.sum
	min := h.min
	max := h.max
	sumsq := h.sumsq
	name := h.name
	percs := []struct {
		n string
		v float64
	}{
		{name + h.percentilePrefix + "99", h.h.Quantile(.99)},
		{name + h.percentilePrefix + "95", h.h.Quantile(.95)},
		{name + h.percentilePrefix + "50", h.h.Quantile(.50)},
	}
	h.reset()
	h.mu.Unlock()

	m := make([]gauge, 0, 4)
	m = append(m,
		gauge{Name: name, Period: period, Count: count, Sum: sum, Min: min, Max: max, SumSq: sumsq},
	)

	for _, perc := range percs {
		m = append(m, gauge{Name: perc.n, Period: period, Count: 1, Sum: perc.v, Min: perc.v, Max: perc.v, SumSq: perc.v * perc.v})
	}
	return m
}

// Observe some data for the histogram
func (h *Histogram) Observe(value float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.count++
	h.sum += value
	if value < h.min || h.min == 0 {
		h.min = value
	}
	if value > h.max {
		h.max = value
	}
	h.sumsq += value * value
	h.h.Add(value)
}

// With returns a new LibratoHistogram with the same name / buckets but it is
// otherwise a noop
func (h *Histogram) With(lv ...string) kmetrics.Histogram {
	n := Histogram{name: h.name, buckets: h.buckets}
	n.reset()
	return &n
}

func (h *Histogram) reset() {
	// Not happy with this, but the existing histogram doesn't have a Reset.
	h.h = gohistogram.NewHistogram(h.buckets)
	h.count = 0
	h.sum = 0
	h.min = 0
	h.max = 0
	h.sumsq = 0
}

// Quantile percentage of reported Observations
func (h *Histogram) Quantile(q float64) float64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.h.Quantile(q)
}

// Count of Observations
func (h *Histogram) Count() int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.count
}

// Min Observation
func (h *Histogram) Min() float64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.min
}

// Max Observation
func (h *Histogram) Max() float64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.max
}

// Sum of the Observations
func (h *Histogram) Sum() float64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.sum
}

// SumSq of the Observations
func (h *Histogram) SumSq() float64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.sumsq
}
