package metrics

import (
	"math"
	"testing"
	"time"

	"github.com/go-kit/kit/metrics/generic"
	"github.com/heroku/cedar/lib/kit/metrics/testmetrics"
)

var timerDurations = []time.Duration{
	time.Nanosecond,
	time.Microsecond,
	time.Millisecond,
	time.Second,
}

func TestTimerDurationFast(t *testing.T) {
	for _, d := range timerDurations {
		h := generic.NewSimpleHistogram()
		timer := &DurationTimer{h: h, t: time.Now(), d: float64(d)}
		timer.ObserveDuration()

		tolerance := float64(100 * time.Microsecond)
		if want, have := 0.000, h.ApproximateMovingAverage(); math.Abs(want-have) > tolerance {
			t.Errorf("NewTimerDuration(h, %s): want %.3f +/- %.3f, have %.3f, diff %.3f", d, want, tolerance, have, math.Abs(want-have))
		}
	}
}

func TestTimerDurationSlow(t *testing.T) {
	for _, d := range timerDurations {
		h := generic.NewSimpleHistogram()
		timer := &DurationTimer{
			h: h,
			d: float64(d),
			t: time.Now().Add(-250 * time.Millisecond),
		}
		timer.ObserveDuration()

		tolerance := float64(100 * time.Microsecond)
		if want, have := float64(250*time.Millisecond)/float64(d), h.ApproximateMovingAverage(); math.Abs(want-have) > tolerance {
			t.Errorf("NewTimerDuration(h, %s): want %.3f +/- %.3f, have %.3f, diff %.3f", d, want, tolerance, have, math.Abs(want-have))
		}
	}
}

func TestMeasureSince(t *testing.T) {
	p := testmetrics.NewProvider(t)
	h := p.NewHistogram("timer", 50)

	done := make(chan struct{})
	go func() {
		defer MeasureSince(h, time.Now())
		time.Sleep(10 * time.Millisecond)
		close(done)
	}()

	<-done
	p.CheckObservationsMinMax("timer", 0, 11)
}

func TestMonotonicTimer(t *testing.T) {
	name := "monotonic-test"
	provider := testmetrics.NewProvider(t)
	h := provider.NewHistogram(name, 50)
	timer := NewMonotonicTimer(h, time.Millisecond, 5*time.Millisecond)
	time.Sleep(28 * time.Millisecond)
	timer.Finish()
	provider.CheckObservationCount(name, 6)
	for i := 0; i < 5; i++ {
		provider.CheckObservationAlmostEqual(name, i, float64((i+1)*5), 1)
	}
	provider.CheckObservationAlmostEqual(name, 5, 28, 1)
}
