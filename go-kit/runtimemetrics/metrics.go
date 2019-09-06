package runtimemetrics

import (
	"runtime"
	"runtime/debug"
	"time"

	kitmetrics "github.com/go-kit/kit/metrics"

	"github.com/heroku/x/go-kit/metrics"
)

// Collector collects metrics about the Go runtime into go-kit metrics.
type Collector struct {
	// Goroutines counts the number of goroutines.
	Goroutines kitmetrics.Gauge

	// AllocBytes counts the bytes of allocated heap objects.
	AllocBytes kitmetrics.Gauge

	// SysBytes is the total bytes of memory obtained from the OS.
	//
	// It's likely that not all of this virtual address space reserved by the Go
	// runtime is backed by physical memory at any given moment.
	SysBytes kitmetrics.Gauge

	// TotalAllocBytes is the cumulative total of bytes allocated for heap
	// objects.
	TotalAllocBytes kitmetrics.Gauge

	// Mallocs is the total count of heap objects ever allocated.
	Mallocs kitmetrics.Gauge

	// Frees is the total count of heap objects ever freed.
	Frees kitmetrics.Gauge

	// GCPauseDuration reports observed GC pause times.
	GCPauseDuration kitmetrics.Histogram

	// NextGCBytes is the target heap size of the next GC cycle.
	//
	// The garbage collector's goal is to keep AllocBytes ≤ NextGCBytes.
	NextGCBytes kitmetrics.Gauge

	// lastGCNum tracks the last GC cycle number so Collect can update
	// GCPauseDuration with only new observations.
	lastGCNum int64
}

// NewCollector returns a collector whose metrics are registered with p.
func NewCollector(p metrics.Provider) *Collector {
	return &Collector{
		Goroutines:      p.NewGauge("go.goroutines"),
		AllocBytes:      p.NewGauge("go.mem.alloc-bytes"),
		SysBytes:        p.NewGauge("go.mem.sys-bytes"),
		TotalAllocBytes: p.NewGauge("go.mem.total-alloc-bytes"),
		Mallocs:         p.NewGauge("go.mem.mallocs"),
		Frees:           p.NewGauge("go.mem.frees"),
		GCPauseDuration: p.NewHistogram("go.gc.pause-duration.ms", 50),
		NextGCBytes:     p.NewGauge("go.gc.next-target-heap-size-bytes"),
	}
}

// Collect calls into the runtime to update its internal metrics.
func (c *Collector) Collect() {
	c.Goroutines.Set(float64(runtime.NumGoroutine()))

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	c.AllocBytes.Set(float64(ms.Alloc))
	c.SysBytes.Set(float64(ms.Sys))
	c.TotalAllocBytes.Set(float64(ms.TotalAlloc))
	c.Mallocs.Set(float64(ms.Mallocs))
	c.Frees.Set(float64(ms.Frees))
	c.NextGCBytes.Set(float64(ms.NextGC))

	var gs debug.GCStats
	debug.ReadGCStats(&gs)

	// It's possible that more GCs have occurred since we last collected than the
	// runtime stores pause data for. In that case, we observe all the pauses
	// available.
	unobserved := int(gs.NumGC - c.lastGCNum)
	if unobserved > len(gs.Pause) {
		unobserved = len(gs.Pause)
	}

	for i := 0; i < unobserved; i++ {
		c.GCPauseDuration.Observe(float64(gs.Pause[i]) / float64(time.Millisecond))
	}

	c.lastGCNum = gs.NumGC
}
