/* Copyright (c) 2018 Salesforce
 * All rights reserved.
 * Licensed under the BSD 3-Clause license.
 * For full license text, see LICENSE.txt file in the repo root  or https://opensource.org/licenses/BSD-3-Clause
 */

// Package metrics is a largely a wrapper around the standard go-kit
// Provider type, with an extension for Cardinality estimators, for use
// on large sets.
//
// It is extracted like this for convenience. See the Provider documentation
// for more information.
package metrics

import (
	"github.com/go-kit/kit/metrics"
)

// Provider represents the different types of metrics that a provider
// can expose. We duplicate the definition from go-kit for 2 reasons:
//
//  1. A little copying never hurt anyone (and in copying, we avoid the
//     need to import and vendor all of go-kit's supported providers
//  2. It provides us an extension mechanism for our own custom metric
//     types that we can implement without go-kit's approval.
type Provider interface {
	NewCounter(name string) metrics.Counter
	NewGauge(name string) metrics.Gauge
	NewHistogram(name string, buckets int) metrics.Histogram
	NewCardinalityCounter(name string) CardinalityCounter
	Stop()
}
