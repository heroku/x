// Package runtimemetrics exposes a go-kit metrics collector for Go runtime
// metrics.
//
// It collects the following metrics:
//
//		go.goroutines - number of goroutines
// 		go.mem.alloc-bytes - allocated bytes for heap objects
// 		go.mem.sys-bytes - bytes requested from OS (may not all be used)
// 		go.mem.total-alloc-bytes - cumulative total allocated bytes for heap objects
// 		go.mem.mallocs - cumultative number of heap allocations
// 		go.mem.frees - cumultaive number of freed heap objects
// 		go.gc.pause-duration - histogram of GC pause durations
// 		go.gc.next-target-heap-size-bytes - target heap size of the next GC cycle
//
package runtimemetrics
