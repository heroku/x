package metrics

import (
	"time"

	kitmetrics "github.com/go-kit/kit/metrics"
)

// DurationTimer acts as a stopwatch, sending observations to a wrapped histogram.
// It's a bit of helpful syntax sugar for h.Observe(time.Since(x)), with a specified
// time duration unit.
type DurationTimer struct {
	h kitmetrics.Histogram
	t time.Time
	d float64
}

// NewDurationTimer wraps the given histogram and records the current time.
// Observations will be made in units of d.
func NewDurationTimer(h kitmetrics.Histogram, d time.Duration) *DurationTimer {
	return &DurationTimer{
		h: h,
		t: time.Now(),
		d: float64(d),
	}
}

// ObserveDuration captures the number of time units since the timer was
// constructed, and forwards that observation to the histogram.
func (t *DurationTimer) ObserveDuration() {
	d := time.Since(t.t)
	if d < 0 {
		d = 0
	}
	t.h.Observe(float64(d) / t.d)
}

// setTime sets the timer's start time to n. Used in tests.
func (t *DurationTimer) setTime(n time.Time) {
	t.t = n
}
