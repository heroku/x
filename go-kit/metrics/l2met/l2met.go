// Package l2met provides a basic log-based metrics provider for cases where a real provider is not available.
package l2met

import (
	"context"
	"sync"
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/generic"
	"github.com/sirupsen/logrus"

	xmetrics "github.com/heroku/x/go-kit/metrics"
)

// Provider provides constructors for creating, tracking, and logging metrics.
type Provider struct {
	logger     logrus.FieldLogger
	mu         sync.Mutex
	counters   map[string]*generic.Counter
	gauges     map[string]*generic.Gauge
	histograms map[string]*generic.Histogram
}

// New returns a metrics provider for constructing and tracing metrics to
// report.
func New(l logrus.FieldLogger) *Provider {
	return &Provider{
		logger:     l,
		counters:   map[string]*generic.Counter{},
		gauges:     map[string]*generic.Gauge{},
		histograms: map[string]*generic.Histogram{},
	}
}

// NewCounter implements Provider.
func (p *Provider) NewCounter(name string) metrics.Counter {
	p.mu.Lock()
	defer p.mu.Unlock()

	if c, ok := p.counters[name]; ok {
		return c
	}

	p.counters[name] = generic.NewCounter(name)
	return p.counters[name]
}

// NewGauge implements Provider.
func (p *Provider) NewGauge(name string) metrics.Gauge {
	p.mu.Lock()
	defer p.mu.Unlock()

	if g, ok := p.gauges[name]; ok {
		return g
	}

	p.gauges[name] = generic.NewGauge(name)
	return p.gauges[name]
}

// NewHistogram implements Provider.
func (p *Provider) NewHistogram(name string, buckets int) metrics.Histogram {
	p.mu.Lock()
	defer p.mu.Unlock()

	if h, ok := p.histograms[name]; ok {
		return h
	}

	p.histograms[name] = generic.NewHistogram(name, buckets)
	return p.histograms[name]
}

// NewCardinalityCounter implements the heroku/x metrics Provider interface. It
// is not implemented for the log-based provider.
func (*Provider) NewCardinalityCounter(string) xmetrics.CardinalityCounter {
	panic("unimplemented")
}

// Run starts the provider, logging metrics once per minute until the context
// is canceled.
func (p *Provider) Run(ctx context.Context) error {
	tick := time.NewTicker(time.Minute)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
			p.log()
		}
	}
}

func (p *Provider) log() {
	p.mu.Lock()
	defer p.mu.Unlock()

	data := logrus.Fields{"at": "metrics"}

	for name, c := range p.counters {
		data["count#"+name] = c.ValueReset()
	}

	for name, g := range p.gauges {
		data["measure#"+name] = g.Value()
	}

	for name, h := range p.histograms {
		v := h.Quantile(0.99)
		// no measurement to report
		if v < 0 {
			continue
		}

		data["measure#"+name+".p99"] = v
	}

	p.logger.WithFields(data).Info()
}

// Stop implements Provider.
func (p *Provider) Stop() {}
