package metrics

import (
	kitmetrics "github.com/go-kit/kit/metrics"
)

// Provider represents all the kinds of metrics a provider must expose
type Provider interface {
	NewCounter(name string) kitmetrics.Counter
	NewGauge(name string) kitmetrics.Gauge
	NewHistogram(name string, buckets int) kitmetrics.Histogram
	Stop()
}
