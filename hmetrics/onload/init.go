/* Copyright (c) 2018 Salesforce
 * All rights reserved.
 * Licensed under the BSD 3-Clause license.
 * For full license text, see LICENSE.txt file in the repo root  or https://opensource.org/licenses/BSD-3-Clause
 */

/*
Package onload automatically starts hmetrics reporting, ignoring errors and
retrying reporting, backing off in 10 second increments.

Use this package when you don't care about stopping reporting, specifying the
endpoint, or being notified of any reporting errors.

usage:

import (
	_ "github.com/heroku/x/hmetrics/onload"
)

See github.com/heroku/x/hmetrics documentation for more info about Heroku Go
metrics and advanced usage.

*/
package onload

import (
	"context"
	"time"

	"github.com/heroku/x/hmetrics"
)

const (
	interval = 10
)

func init() {
	go func() {
		var backoff int64
		for backoff = 1; ; backoff++ {
			start := time.Now()
			hmetrics.Report(context.Background(), hmetrics.DefaultEndpoint, nil)
			if time.Since(start) > 5*time.Minute {
				backoff = 1
			}
			time.Sleep(time.Duration(backoff*interval) * time.Second)
		}
	}()
}
