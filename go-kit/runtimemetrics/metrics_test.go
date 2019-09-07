package runtimemetrics

import (
	"runtime"
	"runtime/debug"
	"testing"

	"github.com/heroku/x/go-kit/metrics/testmetrics"
)

func TestCollectGoroutines(t *testing.T) {
	p := testmetrics.NewProvider(t)
	c := NewCollector(p)

	n := runtime.NumGoroutine()
	c.Collect()

	p.CheckGauge("go.goroutines", float64(n))
}

func TestCollectMemStats(t *testing.T) {
	p := testmetrics.NewProvider(t)
	c := NewCollector(p)

	c.Collect()

	p.CheckGaugeNonZero("go.mem.alloc-bytes")
	p.CheckGaugeNonZero("go.mem.total-alloc-bytes")
	p.CheckGaugeNonZero("go.mem.mallocs")
	p.CheckGaugeNonZero("go.mem.frees")
	p.CheckGaugeNonZero("go.mem.sys-bytes")
}

func TestCollectGCStats(t *testing.T) {
	p := testmetrics.NewProvider(t)
	c := NewCollector(p)

	prev := debug.SetGCPercent(-1)
	defer debug.SetGCPercent(prev)

	var gs debug.GCStats

	runtime.GC()
	c.Collect()

	debug.ReadGCStats(&gs)
	p.CheckObservationsMatch("go.gc.pause-duration.ms", pauseDurations(gs))
	p.CheckGaugeNonZero("go.gc.next-target-heap-size-bytes")

	runtime.GC()
	c.Collect()

	debug.ReadGCStats(&gs)
	p.CheckObservationsMatch("go.gc.pause-duration.ms", pauseDurations(gs))

	runtime.GC()
	runtime.GC()

	c.Collect()

	debug.ReadGCStats(&gs)
	p.CheckObservationsMatch("go.gc.pause-duration.ms", pauseDurations(gs))

	// Trigger more GCs than the runtime stores in its buffer, to ensure the
	// collector handles that case well.
	maxGCHistory := len(runtime.MemStats{}.PauseNs)
	for i := 0; i < maxGCHistory+1; i++ {
		runtime.GC()
	}

	c.Collect()
}

func pauseDurations(gs debug.GCStats) []float64 {
	ds := make([]float64, 0, len(gs.Pause))
	for _, pause := range gs.Pause {
		ds = append(ds, float64(pause)/1e6)
	}

	return ds
}
