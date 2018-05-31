/* Copyright (c) 2018 Salesforce
 * All rights reserved.
 * Licensed under the BSD 3-Clause license.
 * For full license text, see LICENSE.txt file in the repo root  or https://opensource.org/licenses/BSD-3-Clause
 */

package metrics

import (
	"sync"

	hll "github.com/axiomhq/hyperloglog"
)

var (
	_ CardinalityCounter = &HLLCounter{}
)

// HLLCounter provides a wrapper around a HyperLogLog probabalistic
// counter, capable of being reported to Librato.
type HLLCounter struct {
	Name    string
	lvs     []string
	mu      sync.RWMutex
	counter *hll.Sketch
}

func NewHLLCounter(name string) *HLLCounter {
	return &HLLCounter{
		Name:    name,
		counter: hll.New(),
	}
}

// With returns a new UniqueCounter with the passed in label values merged
// with the previous label values. The counter's values are copied.
func (c *HLLCounter) With(labelValues ...string) CardinalityCounter {
	nlv := make([]string, len(c.lvs)+len(labelValues), 0)
	nlv = append(nlv, c.lvs...)
	nlv = append(nlv, labelValues...)

	c.mu.RLock()
	defer c.mu.RUnlock()
	return &HLLCounter{
		Name:    c.Name,
		lvs:     nlv,
		counter: c.counter.Clone(),
	}
}

// Insert counts x as a unique value.
func (c *HLLCounter) Insert(x []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.counter.Insert(x)
}

// Estimate the cardinality of the inserted items.
// Safe for concurrent use.
func (c *HLLCounter) Estimate() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.counter.Estimate()
}

// EstimateReset calculates the final estimate, and resets the counter
func (c *HLLCounter) EstimateReset() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	val := c.counter.Estimate()
	c.counter = hll.New()
	return val
}