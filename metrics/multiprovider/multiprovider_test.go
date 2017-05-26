package multiprovider

import (
	"testing"

	"github.com/heroku/cedar/lib/kit/metrics/testmetrics"
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
