/* Copyright (c) 2018 Salesforce
 * All rights reserved.
 * Licensed under the BSD 3-Clause license.
 * For full license text, see LICENSE.txt file in the repo root  or https://opensource.org/licenses/BSD-3-Clause
 */

package metrics

import (
	"testing"
)

func TestHLLCounterWith(t *testing.T) {
	c := NewHLLCounter("foo").With("bar", "baz")
	c.Insert([]byte("foo"))
}

func TestHLLCounterEstimate(t *testing.T) {
	c := NewHLLCounter("foo")
	c.Insert([]byte("foo"))

	val := c.Estimate()
	if val != 1 {
		t.Errorf("got %d, want 1", val)
	}
}

func TestHLLCounterEstimateReset(t *testing.T) {
	c := NewHLLCounter("foo")
	c.Insert([]byte("foo"))

	val := c.EstimateReset()
	if val != 1 {
		t.Errorf("got %d, want 1", val)
	}

	val = c.Estimate()
	if val != 0 {
		t.Errorf("got %d, want 0", val)
	}
}

// Benchmarks how we write Estimate.
// Because Estimate is wrapped in a mutex, it may actually
// be cheaper to clone the underlying HLL counter and estimate
// on that, instead of staying locked against the expensive
// Estimate call.
//
// BenchmarkEstimateViaClone-4     10000000               197 ns/op             128 B/op          3 allocs/op
// BenchmarkEstimate-4             20000000                80.0 ns/op             0 B/op          0 allocs/op
func BenchmarkEstimateViaClone(b *testing.B) {
	hc := NewHLLCounter("foo")

	for i := 0; i < b.N; i++ {
		hc.mu.Lock()
		d := hc.counter.Clone()
		hc.mu.Unlock()

		d.Estimate()
	}
}

func BenchmarkEstimate(b *testing.B) {
	hc := NewHLLCounter("foo")

	for i := 0; i < b.N; i++ {
		hc.Estimate()
	}
}
