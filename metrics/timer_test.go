package metrics

import (
	"math"
	"testing"
	"time"

	"github.com/go-kit/kit/metrics/generic"
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
		NewDurationTimer(h, d).ObserveDuration()

		tolerance := float64(100 * time.Microsecond)
		if want, have := 0.000, h.ApproximateMovingAverage(); math.Abs(want-have) > tolerance {
			t.Errorf("NewTimerDuration(h, %s): want %.3f +/- %.3f, have %.3f, diff %.3f", d, want, tolerance, have, math.Abs(want-have))
		}
	}
}

func TestTimerDurationSlow(t *testing.T) {
	for _, d := range timerDurations {
		h := generic.NewSimpleHistogram()
		timer := NewDurationTimer(h, d)
		timer.setTime(time.Now().Add(-250 * time.Millisecond))
		timer.ObserveDuration()

		tolerance := float64(100 * time.Microsecond)
		if want, have := float64(250*time.Millisecond)/float64(d), h.ApproximateMovingAverage(); math.Abs(want-have) > tolerance {
			t.Errorf("NewTimerDuration(h, %s): want %.3f +/- %.3f, have %.3f, diff %.3f", d, want, tolerance, have, math.Abs(want-have))
		}
	}
}
