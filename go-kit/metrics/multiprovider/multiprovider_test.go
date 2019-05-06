package multiprovider

import (
	"testing"

	"github.com/heroku/x/go-kit/metrics/testmetrics"
)

func TestCounters(t *testing.T) {
	p1 := testmetrics.NewProvider(t)
	p2 := testmetrics.NewProvider(t)

	p := New(p1, p2)
	p.NewCounter("foo").Add(1)

	p1.CheckCounter("foo", 1)
	p2.CheckCounter("foo", 1)
}

func TestGauges(t *testing.T) {
	p1 := testmetrics.NewProvider(t)
	p2 := testmetrics.NewProvider(t)

	p := New(p1, p2)
	p.NewGauge("foo").Add(1)

	p1.CheckGauge("foo", 1)
	p2.CheckGauge("foo", 1)
}

func TestHistograms(t *testing.T) {
	p1 := testmetrics.NewProvider(t)
	p2 := testmetrics.NewProvider(t)

	p := New(p1, p2)
	p.NewHistogram("foo", 1).Observe(1)

	p1.CheckObservationCount("foo", 1)
	p2.CheckObservationCount("foo", 1)
}

func TestStop(t *testing.T) {
	p1 := testmetrics.NewProvider(t)
	p2 := testmetrics.NewProvider(t)

	p := New(p1, p2)
	p.Stop()

	p1.CheckStopped()
	p2.CheckStopped()
}

func TestMultiCardinalityCounter(t *testing.T) {
	p1 := testmetrics.NewProvider(t)
	p2 := testmetrics.NewProvider(t)

	p := New(p1, p2)
	p.NewCardinalityCounter("foo").Insert([]byte("bar"))

	p1.CheckCardinalityCounter("foo", 1)
	p2.CheckCardinalityCounter("foo", 1)
}
