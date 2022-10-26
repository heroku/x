package clocktest

import (
	"time"

	"github.com/heroku/router/clock"
)

// New returns a test clock that will respond to
// Now() calls using the times provided.
func New(ts ...time.Time) clock.Clock {
	return clock.Func(func() (t time.Time) {
		if len(ts) == 0 {
			return
		}
		if len(ts) == 1 {
			t, ts = ts[0], ts[:0]
			return
		}

		t, ts = ts[0], ts[1:]
		return
	})
}

// NewFromDurations returns a test clock that will respond to
// Now() calls with times that are after the current time by
// the passed durations.
func NewFromDurations(ds ...time.Duration) clock.Clock {
	t0 := time.Now()
	ts := make([]time.Time, 0, len(ds))
	for _, d := range ds {
		ts = append(ts, t0.Add(d))
	}
	return New(ts...)
}
