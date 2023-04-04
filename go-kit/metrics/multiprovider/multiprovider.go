// Package multiprovider allows multiple metrics.Providers to be composed together to report metrics to multiple places.
package multiprovider

import (
	"strings"

	kitmetrics "github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/multi"
	"github.com/pkg/errors"

	"github.com/heroku/x/go-kit/metrics"
)

// New takes any number of providers and returns a metrics.Provider that fans
// out all constructor calls to all the providers.
func New(providers ...metrics.Provider) metrics.Provider {
	return &multiProvider{providers: providers}
}

// multiProvider is also a metrics.Provider
var _ metrics.Provider = &multiProvider{}

type multiProvider struct {
	providers []metrics.Provider
}

// NewCounter returns a multi.Counter composed from all the given providers.
func (m *multiProvider) NewCounter(name string) kitmetrics.Counter {
	counters := make([]kitmetrics.Counter, 0, len(m.providers))

	for _, p := range m.providers {
		counters = append(counters, p.NewCounter(name))
	}
	return multi.NewCounter(counters...)
}

// NewGauge returns a multi.Gauge composed from all the given providers.
func (m *multiProvider) NewGauge(name string) kitmetrics.Gauge {
	gauges := make([]kitmetrics.Gauge, 0, len(m.providers))

	for _, p := range m.providers {
		gauges = append(gauges, p.NewGauge(name))
	}
	return multi.NewGauge(gauges...)
}

// NewHistogram returns a multi.Histogram composed from all the given providers.
func (m *multiProvider) NewHistogram(name string, buckets int) kitmetrics.Histogram {
	histograms := make([]kitmetrics.Histogram, 0, len(m.providers))

	for _, p := range m.providers {
		histograms = append(histograms, p.NewHistogram(name, buckets))
	}
	return multi.NewHistogram(histograms...)
}

// NewExplicitHistogram returns a multi.Histogram composed from all the given providers.
func (m *multiProvider) NewExplicitHistogram(name string, fn metrics.DistributionFunc) kitmetrics.Histogram {
	histograms := make([]kitmetrics.Histogram, 0, len(m.providers))

	for _, p := range m.providers {
		histograms = append(histograms, p.NewExplicitHistogram(name, fn))
	}
	return multi.NewHistogram(histograms...)
}

// NewCardinalityCounter implements metrics.CardinalityCounter.
func (m *multiProvider) NewCardinalityCounter(name string) metrics.CardinalityCounter {
	cardCounters := make([]metrics.CardinalityCounter, 0, len(m.providers))

	for _, p := range m.providers {
		cardCounters = append(cardCounters, p.NewCardinalityCounter(name))
	}
	return MultiCardinalityCounter(cardCounters)
}

// Stop calls stop on all the underlying providers.
func (m *multiProvider) Stop() {
	for _, p := range m.providers {
		p.Stop()
	}
}

// Flush calls flush on all the underlying providers.
func (m *multiProvider) Flush() error {
	errMsgs := []string{}
	for _, p := range m.providers {
		if err := p.Flush(); err != nil {
			// Don't immediately return the error.
			// Record error msg and continue trying to
			// flush to other providers.
			errMsgs = append(errMsgs, err.Error())
		}
	}
	if len(errMsgs) == 0 {
		return nil
	}

	return errors.Errorf("flush failed for at least one provider: %s", strings.Join(errMsgs, ";"))
}

type MultiCardinalityCounter []metrics.CardinalityCounter

func (cc MultiCardinalityCounter) With(labelValues ...string) metrics.CardinalityCounter {
	cardCounters := make([]metrics.CardinalityCounter, 0, len(cc))
	for _, cardCounter := range cc {
		cardCounters = append(cardCounters, cardCounter.With(labelValues...))
	}
	return MultiCardinalityCounter(cardCounters)
}

func (cc MultiCardinalityCounter) Insert(b []byte) {
	for _, cardCounter := range cc {
		cardCounter.Insert(b)
	}
}
