package metricsregistry

import (
	"testing"

	"github.com/heroku/x/go-kit/metrics/testmetrics"
)

func TestGetOrRegisterCounter(t *testing.T) {
	t.Run("basic registry", func(t *testing.T) {
		p := testmetrics.NewProvider(t)
		r := New(p)
		runCounterTests(t, r, p, "")
	})

	t.Run("with prefix", func(t *testing.T) {
		p := testmetrics.NewProvider(t)
		r := New(p)
		runCounterTests(t, NewPrefixed(r, "prefix"), p, "prefix.")
	})
}

func TestGetOrRegisterGauge(t *testing.T) {
	t.Run("basic registry", func(t *testing.T) {
		p := testmetrics.NewProvider(t)
		r := New(p)
		runGaugeTests(t, r, p, "")
	})

	t.Run("with prefix", func(t *testing.T) {
		p := testmetrics.NewProvider(t)
		r := New(p)
		runGaugeTests(t, NewPrefixed(r, "prefix"), p, "prefix.")
	})
}

func TestGetOrRegisterHistogram(t *testing.T) {
	t.Run("basic registry", func(t *testing.T) {
		p := testmetrics.NewProvider(t)
		r := New(p)
		runHistogramTests(t, r, p, "")
	})

	t.Run("with prefix", func(t *testing.T) {
		p := testmetrics.NewProvider(t)
		r := New(p)
		runHistogramTests(t, NewPrefixed(r, "prefix"), p, "prefix.")
	})
}

func runCounterTests(t *testing.T, r Registry, p *testmetrics.Provider, prefix string) {
	t.Helper()
	r.GetOrRegisterCounter("foo").Add(1)
	r.GetOrRegisterCounter("foo").Add(1)
	p.CheckCounter(prefix+"foo", 2)

	r.GetOrRegisterCounter("bar").Add(1)
	p.CheckCounter(prefix+"bar", 1)
}

func runHistogramTests(t *testing.T, r Registry, p *testmetrics.Provider, prefix string) {
	t.Helper()
	r.GetOrRegisterHistogram("foo", 1).Observe(1)
	r.GetOrRegisterHistogram("foo", 1).Observe(1)
	p.CheckObservationCount(prefix+"foo", 2)

	r.GetOrRegisterHistogram("bar", 1).Observe(1)
	p.CheckObservationCount(prefix+"bar", 1)
}

func runGaugeTests(t *testing.T, r Registry, p *testmetrics.Provider, prefix string) {
	t.Helper()
	r.GetOrRegisterGauge("foo").Add(1)
	r.GetOrRegisterGauge("foo").Add(1)
	p.CheckGauge(prefix+"foo", 2)

	r.GetOrRegisterGauge("bar").Add(1)
	p.CheckGauge(prefix+"bar", 1)
}
