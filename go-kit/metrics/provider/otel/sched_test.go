package otel

import (
	"context"
	"math"
	"runtime/metrics"
	"testing"

	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestBucketQuantile(t *testing.T) {
	buckets := []float64{0, 1, 2, 4, 8}

	tests := []struct {
		name   string
		counts []uint64
		q      float64
		want   float64
		wantOK bool
	}{
		{"empty", []uint64{0, 0, 0, 0}, 0.5, 0, false},
		{"median", []uint64{5, 5, 0, 0}, 0.5, 1, true},
		{"p99", []uint64{98, 1, 1, 0}, 0.99, 2, true},
		{"max", []uint64{98, 1, 0, 1}, 1, 8, true},
		{"mismatched lengths", []uint64{1, 1}, 0.5, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := bucketQuantile(buckets, tt.counts, tt.q)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("bucketQuantile() = %v, %v; want %v, %v", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestBucketQuantile_InfiniteUpperBound(t *testing.T) {
	inf := []float64{0, 1, math.Inf(1)}
	got, ok := bucketQuantile(inf, []uint64{0, 3}, 1)
	if !ok || got != 1 {
		t.Errorf("bucketQuantile() = %v, %v; want 1, true", got, ok)
	}
}

func TestBucketDelta(t *testing.T) {
	cur := []uint64{5, 7, 9}
	if got := bucketDelta(nil, cur); got[0] != 5 || got[1] != 7 || got[2] != 9 {
		t.Errorf("first reading = %v; want %v", got, cur)
	}
	if got := bucketDelta([]uint64{5, 3, 9}, cur); got[0] != 0 || got[1] != 4 || got[2] != 0 {
		t.Errorf("delta = %v; want [0 4 0]", got)
	}
	if got := bucketDelta([]uint64{1, 1}, cur); got[0] != 5 {
		t.Errorf("mismatched length = %v; want cur", got)
	}
}

func TestRegisterSchedulerLatency(t *testing.T) {
	buckets := []float64{0, 1, 2, 4, 8}
	counts := []uint64{0, 0, 0, 0}
	read := func() *metrics.Float64Histogram {
		return &metrics.Float64Histogram{Buckets: buckets, Counts: append([]uint64(nil), counts...)}
	}

	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))
	if err := registerSchedulerLatency(mp.Meter("test"), read); err != nil {
		t.Fatal(err)
	}

	collect := func() map[string]float64 {
		t.Helper()
		var rm metricdata.ResourceMetrics
		if err := reader.Collect(context.Background(), &rm); err != nil {
			t.Fatal(err)
		}
		got := map[string]float64{}
		for _, sm := range rm.ScopeMetrics {
			for _, m := range sm.Metrics {
				g := m.Data.(metricdata.Gauge[float64])
				got[m.Name] = g.DataPoints[0].Value
			}
		}
		return got
	}

	// No samples: nothing observed.
	if got := collect(); len(got) != 0 {
		t.Errorf("empty interval observed %v", got)
	}

	// First interval: 98 fast, 1 mid, 1 slow.
	counts = []uint64{98, 1, 0, 1}
	got := collect()
	want := map[string]float64{
		"process.runtime.go.sched.latency.p50": 1,
		"process.runtime.go.sched.latency.p99": 2,
		"process.runtime.go.sched.latency.max": 8,
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("interval 1 %s = %v; want %v", k, got[k], v)
		}
	}

	// Second interval: only 10 new samples in the 2-4 bucket; the cumulative
	// counts from interval 1 must not leak in.
	counts = []uint64{98, 1, 10, 1}
	got = collect()
	for _, k := range []string{"p50", "p99", "max"} {
		if v := got["process.runtime.go.sched.latency."+k]; v != 4 {
			t.Errorf("interval 2 %s = %v; want 4", k, v)
		}
	}
}

func TestWithSchedulerLatencyMetrics(t *testing.T) {
	p, err := New(context.Background(), "my-service", WithSchedulerLatencyMetrics(true))
	if err != nil {
		t.Fatal(err)
	}
	if err := p.(*Provider).Start(); err != nil {
		t.Fatalf("Start() = %v", err)
	}
	p.Stop()
}

func TestReadSchedLatencies(t *testing.T) {
	h := readSchedLatencies()
	if h == nil || len(h.Buckets) != len(h.Counts)+1 {
		t.Fatalf("unexpected histogram: %+v", h)
	}
}
