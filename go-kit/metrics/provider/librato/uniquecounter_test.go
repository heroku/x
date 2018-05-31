/* Copyright (c) 2018 Salesforce
 * All rights reserved.
 * Licensed under the BSD 3-Clause license.
 * For full license text, see LICENSE.txt file in the repo root  or https://opensource.org/licenses/BSD-3-Clause
 */

package librato

import (
	"testing"
)

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
