package explicit

import (
	"reflect"
	"testing"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/number"
	"go.opentelemetry.io/otel/metric/sdkapi"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/export/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/histogram"
)

var (
	defaultFloat64ExplicitBoundaries = []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}
)

func TestSelector_HistogramConfig(t *testing.T) {
	_, cache := NewExplicitHistogramDistribution()

	t.Run("when empty", func(t *testing.T) {

		got := cache.Fetch("test")
		if len(got) != 0 {
			t.Fatalf("expected empty options, got %v", got)
		}
	})

	t.Run("with single option", func(t *testing.T) {
		_, cache := NewExplicitHistogramDistribution()

		boundaries := defaultFloat64ExplicitBoundaries

		want := []histogram.Option{
			histogram.WithExplicitBoundaries(boundaries),
		}

		// store and fetch the default boundaries
		cache.Store("test", want...)
		got := cache.Fetch("test")

		if len(got) != 1 {
			t.Fatalf("unexpected options length, want %v, got %v", 1, len(got))
		}

		if !reflect.DeepEqual(want, got) {
			t.Fatalf("unexpected options, want %v and got %v", want, got)
		}
	})

	t.Run("replaces single option", func(t *testing.T) {
		_, cache := NewExplicitHistogramDistribution()

		// store the default boundaries
		cache.Store("test", histogram.WithExplicitBoundaries(defaultFloat64ExplicitBoundaries))

		boundaries := []float64{1, 2, 3, 4, 5}
		want := []histogram.Option{
			histogram.WithExplicitBoundaries(boundaries),
		}

		// store and fetch some updated boundaries
		cache.Store("test", want...)
		got := cache.Fetch("test")

		if len(got) != 1 {
			t.Fatalf("unexpected options length, want %v, got %v", 1, len(got))
		}

		if !reflect.DeepEqual(want, got) {
			t.Fatalf("unexpected options, want %v and got %v", want, got)
		}
	})

	t.Run("many named metrics", func(t *testing.T) {
		_, cache := NewExplicitHistogramDistribution()

		metrics := []struct {
			name   string
			option []float64
		}{
			{
				name:   "test1",
				option: []float64{1, 2, 3, 4, 5},
			},
			{
				name:   "test2",
				option: []float64{10, 20, 30, 40, 50},
			},
		}

		for _, metric := range metrics {
			want := []histogram.Option{
				histogram.WithExplicitBoundaries(metric.option),
			}

			// store and fetch some updated boundaries
			cache.Store(metric.name, want...)
			got := cache.Fetch(metric.name)

			if len(got) != 1 {
				t.Fatalf("expected empty options, got %v", got)
			}

			if !reflect.DeepEqual(want, got) {
				t.Fatalf("unexpected options, want %v and got %v", want, got)
			}
		}

		c, ok := cache.(*selectorCache)
		if !ok {
			t.Errorf("unexpected casting problem err: %v", ok)
		}

		if len(c.opts) != 2 {
			t.Fatalf("unexpected opts map, want 2 keys and got %v", c.opts)
		}
	})

}

func TestSelector_Histogram(t *testing.T) {
	var aggregator export.Aggregator
	desc := metric.NewDescriptor("test", sdkapi.HistogramInstrumentKind, number.Float64Kind)

	t.Run("no options", func(t *testing.T) {
		selector, _ := NewExplicitHistogramDistribution()

		selector.AggregatorFor(&desc, &aggregator)

		if aggregator == nil {
			t.Fatal("expected aggregator to be initialized, got nil")
		}

		agg := aggregator.Aggregation()
		if agg.Kind() != aggregation.HistogramKind {
			t.Fatalf("expected kind to be %v, got %v", aggregation.HistogramKind, agg.Kind())
		}

		buckets, err := agg.(aggregation.Histogram).Histogram()
		if err != nil {
			t.Error(err)
		}

		boundaries := buckets.Boundaries
		if !reflect.DeepEqual(boundaries, defaultFloat64ExplicitBoundaries) {
			t.Fatalf("expected boundaries to match, want: %v, got: %v", defaultFloat64ExplicitBoundaries, boundaries)
		}
	})

	t.Run("custom options", func(t *testing.T) {
		want := []float64{1, 2, 3, 4, 5}

		selector, cache := NewExplicitHistogramDistribution()
		cache.Store(desc.Name(), histogram.WithExplicitBoundaries(want))

		selector.AggregatorFor(&desc, &aggregator)

		if aggregator == nil {
			t.Fatal("expected aggregator to be initialized, got nil")
		}

		agg := aggregator.Aggregation()
		if agg.Kind() != aggregation.HistogramKind {
			t.Fatalf("expected kind to be %v, got %v", aggregation.HistogramKind, agg.Kind())
		}

		buckets, err := agg.(aggregation.Histogram).Histogram()
		if err != nil {
			t.Error(err)
		}

		got := buckets.Boundaries
		if !reflect.DeepEqual(want, got) {
			t.Fatalf("expected boundaries to match, want: %v, got: %v", want, got)
		}

	})
}
