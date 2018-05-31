/* Copyright (c) 2018 Salesforce
 * All rights reserved.
 * Licensed under the BSD 3-Clause license.
 * For full license text, see LICENSE.txt file in the repo root  or https://opensource.org/licenses/BSD-3-Clause
 */

package metrics

// UniqueCounter describes a metric that reports a count of the number of
// unique values seen.
type UniqueCounter interface {
	With(labelValues ...string) UniqueCounter
	Insert(b []byte)
}
