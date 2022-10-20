package lowcard

import (
	"github.com/go-kit/kit/metrics"
	xmetrics "github.com/heroku/x/go-kit/metrics"
)

// NewLowCardinalityWrappedProvider is a wrapper around metrics.Provider
// that ignores any added labels via metric.With() call.
func NewLowCardinalityWrappedProvider(p xmetrics.Provider, labels []string) xmetrics.Provider {
	return lowCardinalityWrappedProvider{
		Provider:        p,
		labelsToInclude: labels,
	}
}

type lowCardinalityWrappedProvider struct {
	xmetrics.Provider
	// Labels that will be added to instruments.
	labelsToInclude []string
}

type counter struct {
	metrics.Counter
	// Labels that will be added to counters.
	labelsToInclude []string
}

// NewCounter wraps counter around metrics.Counter.
func (p lowCardinalityWrappedProvider) NewCounter(name string) metrics.Counter {
	return counter{
		Counter:         p.Provider.NewCounter(name),
		labelsToInclude: p.labelsToInclude,
	}
}

// With noops and returns counter if the passed labelValues
// do not start with a label that has been designated for inclusion.
func (c counter) With(labelValues ...string) metrics.Counter {
	c.Counter = c.Counter.With(sanitizeLabels(c.labelsToInclude, labelValues)...)
	return c
}

type gauge struct {
	metrics.Gauge
	// Labels that will be added to gauges.
	labelsToInclude []string
}

// NewGauge wraps gauge around metrics.Gauge.
func (p lowCardinalityWrappedProvider) NewGauge(name string) metrics.Gauge {
	return gauge{
		Gauge:           p.Provider.NewGauge(name),
		labelsToInclude: p.labelsToInclude,
	}
}

// With noops and returns gauge if the passed labelValues
// do not start with a label that has been designated for inclusion.
func (g gauge) With(labelValues ...string) metrics.Gauge {
	g.Gauge = g.Gauge.With(sanitizeLabels(g.labelsToInclude, labelValues)...)
	return g
}

type histogram struct {
	metrics.Histogram
	// Labels that will be added to histograms.
	labelsToInclude []string
}

// With noops and returns histogram if the passed labelValues
// do not start with a label that has been designated for inclusion.
func (p lowCardinalityWrappedProvider) NewHistogram(name string, buckets int) metrics.Histogram {
	return histogram{
		Histogram:       p.Provider.NewHistogram(name, buckets),
		labelsToInclude: p.labelsToInclude,
	}
}

// With noops and returns histogram if the passed labelValues
// do not start with a label that has been designated for inclusion.
func (p lowCardinalityWrappedProvider) NewExplicitHistogram(name string, fn xmetrics.DistributionFunc) metrics.Histogram {
	return histogram{
		Histogram:       p.Provider.NewExplicitHistogram(name, fn),
		labelsToInclude: p.labelsToInclude,
	}
}

// With noops and returns histogram.
func (h histogram) With(labelValues ...string) metrics.Histogram {
	h.Histogram = h.Histogram.With(sanitizeLabels(h.labelsToInclude, labelValues)...)
	return h
}

func sanitizeLabels(labelsToInclude, labelValues []string) []string {
	sanitized := []string{}

	// The instrument expects an even number of values
	// since they represent key-value pairs. If there
	// are not an even number, we assume labels are malformed.
	// If malformed, the labels are not added.
	if len(labelValues)%2 != 0 {
		return []string{}
	}
	for i := 0; i <= len(labelValues)-2; i += 2 {
		if contains(labelsToInclude, labelValues[i]) {
			sanitized = append(sanitized, labelValues[i], labelValues[i+1])
		}
	}
	return sanitized
}

func contains(strs []string, str string) bool {
	for _, s := range strs {
		if s == str {
			return true
		}
	}
	return false
}
