package metrics

import (
	"time"

	kitmetrics "github.com/go-kit/kit/metrics"
)

// defaultTimingUnit is the resolution we'll use for all duration measurements.
const defaultTimingUnit = time.Millisecond

// DurationTimer acts as a stopwatch, sending observations to a wrapped histogram.
// It's a bit of helpful syntax sugar for h.Observe(time.Since(x)), with a specified
// time duration unit.
type DurationTimer struct {
	h kitmetrics.Histogram
	t time.Time
	d float64
}

// NewDurationTimer wraps the given histogram and records the current time.
// It defaults to time.Millisecond units.
func NewDurationTimer(h kitmetrics.Histogram) *DurationTimer {
	return &DurationTimer{
		h: h,
		t: time.Now(),
		d: float64(defaultTimingUnit),
	}
}

// ObserveDuration captures the number of time units since the timer was
// constructed, and forwards that observation to the histogram.
func (t *DurationTimer) ObserveDuration() {
	measureSince(t.h, t.t, t.d)
}

// MeasureSince takes a Histogram and initial time and generates
// an observation for the total duration of the operation. It's
// intended to be called via defer, e.g. defer MeasureSince(h, time.Now()).
func MeasureSince(h kitmetrics.Histogram, t0 time.Time) {
	measureSince(h, t0, float64(defaultTimingUnit))
}

// measureSince is the underlying code for supporting both MeasureSince
// and DurationTimer.ObserveDuration.
func measureSince(h kitmetrics.Histogram, t0 time.Time, unit float64) {
	d := time.Since(t0)
	if d < 0 {
		d = 0
	}
	h.Observe(float64(d) / unit)
}
