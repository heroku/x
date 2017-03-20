package grpcmetrics

import "github.com/go-kit/kit/metrics"

// Provider defines the subset of go-kit's Provider interface
// required by grpcmetrics.
type Provider interface {
	NewCounter(name string) metrics.Counter
	NewHistogram(name string, buckets int) metrics.Histogram
}
