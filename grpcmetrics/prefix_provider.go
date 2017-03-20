package grpcmetrics

import "github.com/go-kit/kit/metrics"

type prefixProvider struct {
	prefix string
	p      Provider
}

func (p *prefixProvider) NewCounter(name string) metrics.Counter {
	return p.p.NewCounter(p.prefix + name)
}

func (p *prefixProvider) NewHistogram(name string, buckets int) metrics.Histogram {
	return p.p.NewHistogram(p.prefix+name, buckets)
}
