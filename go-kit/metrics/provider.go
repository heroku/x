/* Copyright (c) 2018 Salesforce
 * All rights reserved.
 * Licensed under the BSD 3-Clause license.
 * For full license text, see LICENSE.txt file in the repo root  or https://opensource.org/licenses/BSD-3-Clause
 */

// Package metrics is a small wrapper around the go-kit metrics Provider type.
//
// It is extracted like this for convenience. See the Provider documentation
// for more information.
package metrics

import (
	"github.com/go-kit/kit/metrics"
)

// Provider represents all the kinds of metrics a provider must expose. This is
// here for 2 reasons: (1) go-kit/metrics/provider imports all the providers in
// the world supported by go-kit cluttering up your vendor folder; and (2)
// provider.Provider (hmmmmm stutter)!
type Provider interface {
	NewCounter(name string) metrics.Counter
	NewGauge(name string) metrics.Gauge
	NewHistogram(name string, buckets int) metrics.Histogram
	NewUniqueCounter(name string) UniqueCounter
	Stop()
}
