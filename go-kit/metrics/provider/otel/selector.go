package otel

import (
	"sync"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/sdkapi"
	export "go.opentelemetry.io/otel/sdk/export/metric"

	"go.opentelemetry.io/otel/sdk/metric/aggregator/histogram"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/lastvalue"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/sum"
)

type (
	config struct {
		lock sync.RWMutex
		opts map[string][]histogram.Option
	}

	selectorHistogram struct {
		cfg *config
	}
)

func (s selectorHistogram) StoreOptions(name string, opts ...histogram.Option) {
	s.cfg.lock.Lock()
	defer s.cfg.lock.Unlock()

	s.cfg.opts[name] = opts
}

func (s selectorHistogram) FetchOptions(name string) []histogram.Option {
	s.cfg.lock.RLock()
	defer s.cfg.lock.RUnlock()

	return s.cfg.opts[name]
}

func (s selectorHistogram) AggregatorFor(desc *metric.Descriptor, aggPtrs ...*export.Aggregator) {
	switch desc.InstrumentKind() {
	case sdkapi.GaugeObserverInstrumentKind:
		lastValueAggs(aggPtrs)
	case sdkapi.HistogramInstrumentKind:
		opts := s.FetchOptions(desc.Name())

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
