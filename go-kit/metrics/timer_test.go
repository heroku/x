package metrics

import (
	"math"
	"sync"
	"testing"
	"time"

	"github.com/go-kit/kit/metrics"
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
		timer := &DurationTimer{h: h, t: time.Now(), d: d}
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
			d: d,
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
	h := generic.NewSimpleHistogram()

	t0 := time.Now().Add(-1 * time.Second)
	t1 := time.Now()
	measureSince(h, t0, t1, defaultTimingUnit)

	want := float64(t1.Sub(t0) / defaultTimingUnit)
	got := h.ApproximateMovingAverage()
	if want != got {
		t.Fatalf("wanted avg: %f, got %f", want, got)
	}
}

func TestMonotonicTimer(t *testing.T) {
	h := &testHistogram{}
	timer := newUnstartedMonotonicTimer(h, time.Millisecond)

	done := make(chan struct{})
	tick := make(chan time.Time)

	go timer.start(func() { close(done) }, tick)

	tick <- time.Now()
	tick <- time.Now()
	timer.Finish()

	<-done

	if want, got := 3, len(h.observations); want != got {
		t.Fatalf("wanted %d observations, got %d", want, got)
	}
}

type testHistogram struct {
	mu           sync.Mutex
	observations []float64
}

func (h *testHistogram) With(lvs ...string) metrics.Histogram {
	return h
}

func (h *testHistogram) Observe(v float64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.observations = append(h.observations, v)
}
