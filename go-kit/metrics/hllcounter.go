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
	mu      sync.Mutex
	counter *hll.Sketch
}

// NewHLLCounter creates a new HyperLogLog based counter.
func NewHLLCounter(name string) *HLLCounter {
	return &HLLCounter{
		Name:    name,
		counter: hll.New(),
	}
}

// With returns a new UniqueCounter with the passed in label values merged
// with the previous label values. The counter's values are copied.
func (c *HLLCounter) With(labelValues ...string) CardinalityCounter {
	nlv := make([]string, 0, len(c.lvs)+len(labelValues))
	nlv = append(nlv, c.lvs...)
	nlv = append(nlv, labelValues...)

	c.mu.Lock()
	defer c.mu.Unlock()
	return &HLLCounter{
		Name:    c.Name,
		lvs:     nlv,
		counter: c.counter.Clone(),
	}
}

// Insert adds the item to the set to be counted.
func (c *HLLCounter) Insert(i []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.counter.Insert(i)
}

// Estimate the cardinality of the inserted items.
// Safe for concurrent use.
func (c *HLLCounter) Estimate() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.counter.Estimate()
}

// EstimateReset returns the cardinality estimate, and resets the estimate to zero allowing a new set to be counted.
// Safe for concurrent use.
func (c *HLLCounter) EstimateReset() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	val := c.counter.Estimate()
	c.counter = hll.New()
	return val
}

// LabelValues returns the label values for this HLLCounter.
func (c *HLLCounter) LabelValues() []string {
	return c.lvs
}
