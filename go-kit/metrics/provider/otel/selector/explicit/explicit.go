package explicit

import (
	"sync"

	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/metric/sdkapi"

	"go.opentelemetry.io/otel/sdk/metric/aggregator/histogram"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/lastvalue"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/sum"
)

type (
	OptionCache interface {
		Store(name string, opts ...histogram.Option)
		Fetch(name string) []histogram.Option
	}

	selectorCache struct {
		lock sync.RWMutex
		opts map[string][]histogram.Option
	}

	selectorHistogram struct {
		cache OptionCache
	}
)

func NewExplicitHistogramDistribution() (export.AggregatorSelector, OptionCache) {
	cache := &selectorCache{
		opts: make(map[string][]histogram.Option),
	}

	return selectorHistogram{
		cache: cache,
	}, cache
}

func (c *selectorCache) Store(name string, opts ...histogram.Option) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.opts[name] = opts
}

func (c *selectorCache) Fetch(name string) []histogram.Option {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.opts[name]
}

func (s selectorHistogram) AggregatorFor(desc *sdkapi.Descriptor, aggPtrs ...*export.Aggregator) {
	switch desc.InstrumentKind() {
	case sdkapi.GaugeObserverInstrumentKind:
		lastValueAggs(aggPtrs)
	case sdkapi.HistogramInstrumentKind:
		opts := s.cache.Fetch(desc.Name())

		aggs := histogram.New(len(aggPtrs), desc, opts...)
		for i := range aggPtrs {
			*aggPtrs[i] = &aggs[i]
		}

	default:
		sumAggs(aggPtrs)
	}
}

func sumAggs(aggPtrs []*export.Aggregator) {
	aggs := sum.New(len(aggPtrs))
	for i := range aggPtrs {
		*aggPtrs[i] = &aggs[i]
	}
}

func lastValueAggs(aggPtrs []*export.Aggregator) {
	aggs := lastvalue.New(len(aggPtrs))
	for i := range aggPtrs {
		*aggPtrs[i] = &aggs[i]
	}
}
