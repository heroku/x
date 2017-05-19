package grpcmetrics

import (
	kitmetrics "github.com/go-kit/kit/metrics"
	"github.com/heroku/cedar/lib/kit/metrics"
)

type prefixProvider struct {
	prefix string
	p      metrics.Provider
}

func (p *prefixProvider) NewCounter(name string) kitmetrics.Counter {
	return p.p.NewCounter(p.prefix + name)
}

func (p *prefixProvider) NewGauge(name string) kitmetrics.Gauge {
	return p.p.NewGauge(p.prefix + name)
}

func (p *prefixProvider) NewHistogram(name string, buckets int) kitmetrics.Histogram {
	return p.p.NewHistogram(p.prefix+name, buckets)
}

func (p *prefixProvider) Stop() {
	p.p.Stop()
}
