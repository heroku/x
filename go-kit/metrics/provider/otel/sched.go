package otel

import (
	"context"
	"math"
	"runtime/metrics"
	"sync"

	otelmetric "go.opentelemetry.io/otel/metric"
)

const (
	schedLatenciesSample = "/sched/latencies:seconds"
	schedScopeName       = "github.com/heroku/x/go-kit/metrics/provider/otel"
)

// schedLatencyReader returns the current cumulative scheduler latency histogram,
// or nil if the runtime does not provide it.
type schedLatencyReader func() *metrics.Float64Histogram

func readSchedLatencies() *metrics.Float64Histogram {
	s := []metrics.Sample{{Name: schedLatenciesSample}}
	metrics.Read(s)
	if s[0].Value.Kind() != metrics.KindFloat64Histogram {
		return nil
	}
	return s[0].Value.Float64Histogram()
}

// registerSchedulerLatency registers observable gauges for the time goroutines
// spend runnable before they run, computed over each collection interval.
//
// The runtime histogram is cumulative since process start, so the callback
// subtracts the previous reading to get the counts for the interval. The
// gauges are bucket bounds, so they overestimate the true value by at most
// one bucket width. Nothing is observed for an interval with no samples.
func registerSchedulerLatency(meter otelmetric.Meter, read schedLatencyReader) error {
	p50, err := meter.Float64ObservableGauge(
		"process.runtime.go.sched.latency.p50",
		otelmetric.WithUnit("s"),
		otelmetric.WithDescription("Median time goroutines spent runnable before running, over the collection interval."),
	)
	if err != nil {
		return err
	}
	p99, err := meter.Float64ObservableGauge(
		"process.runtime.go.sched.latency.p99",
		otelmetric.WithUnit("s"),
		otelmetric.WithDescription("99th percentile time goroutines spent runnable before running, over the collection interval."),
	)
	if err != nil {
		return err
	}
	maxLatency, err := meter.Float64ObservableGauge(
		"process.runtime.go.sched.latency.max",
		otelmetric.WithUnit("s"),
		otelmetric.WithDescription("Longest time a goroutine spent runnable before running, over the collection interval."),
	)
	if err != nil {
		return err
	}

	var (
		mu   sync.Mutex
		prev []uint64
	)
	_, err = meter.RegisterCallback(func(_ context.Context, o otelmetric.Observer) error {
		h := read()
		if h == nil {
			return nil
		}

		mu.Lock()
		delta := bucketDelta(prev, h.Counts)
		prev = append(prev[:0], h.Counts...)
		mu.Unlock()

		if q50, ok := bucketQuantile(h.Buckets, delta, 0.50); ok {
			o.ObserveFloat64(p50, q50)
		}
		if q99, ok := bucketQuantile(h.Buckets, delta, 0.99); ok {
			o.ObserveFloat64(p99, q99)
		}
		if qmax, ok := bucketQuantile(h.Buckets, delta, 1); ok {
			o.ObserveFloat64(maxLatency, qmax)
		}
		return nil
	}, p50, p99, maxLatency)
	return err
}

// bucketDelta returns cur - prev per bucket. The first reading (or one whose
// bucket count changed) is returned as is, which covers the time since process start.
func bucketDelta(prev, cur []uint64) []uint64 {
	d := make([]uint64, len(cur))
	for i, c := range cur {
		d[i] = c
		if len(prev) == len(cur) && prev[i] <= c {
			d[i] = c - prev[i]
		}
	}
	return d
}

// bucketQuantile returns the upper bound of the bucket holding the q-th
// quantile of counts. buckets holds len(counts)+1 boundaries; bucket i spans
// [buckets[i], buckets[i+1]). It reports false when counts is empty. An
// infinite upper bound is replaced by the bucket's lower bound.
func bucketQuantile(buckets []float64, counts []uint64, q float64) (float64, bool) {
	if len(buckets) != len(counts)+1 {
		return 0, false
	}
	var total uint64
	for _, c := range counts {
		total += c
	}
	if total == 0 {
		return 0, false
	}

	rank := uint64(math.Ceil(q * float64(total)))
	if rank < 1 {
		rank = 1
	}
	var seen uint64
	for i, c := range counts {
		seen += c
		if c == 0 || seen < rank {
			continue
		}
		if upper := buckets[i+1]; !math.IsInf(upper, 1) {
			return upper, true
		}
		return buckets[i], true
	}
	return 0, false
}
