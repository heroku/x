/* Copyright (c) 2018 Salesforce
 * All rights reserved.
 * Licensed under the BSD 3-Clause license.
 * For full license text, see LICENSE.txt file in the repo root  or https://opensource.org/licenses/BSD-3-Clause
 */

package librato

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/heroku/x/scrub"
)

// Error is used to report information from a non 200 error returned by Librato.
type Error struct {
	code, retries                    int // return code and retries remaining
	body, rateLimitAgg, rateLimitStd string

	// Used to debug things on occasion if inspection of the original
	// request is necessary.
	dumpedRequest string
}

// Code returned by librato
func (e Error) Code() int {
	return e.code
}

// Temporary error that will be retried?
func (e Error) Temporary() bool {
	return e.retries > 0
}

// Request that generated the error
func (e Error) Request() string {
	return e.dumpedRequest
}

// RateLimit info returned by librato in the X-Librato-RateLimit-Agg and
// X-Librato-RateLimit-Std headers
func (e Error) RateLimit() (string, string) {
	return e.rateLimitAgg, e.rateLimitStd
}

// Body returned by librato.
func (e Error) Body() string {
	return e.body
}

// Error interface
func (e Error) Error() string {
	return fmt.Sprintf("code: %d, retries remaining: %d, body: %s, rate-limit-agg: %s, rate-limit-std: %s", e.code, e.retries, e.body, e.rateLimitAgg, e.rateLimitStd)
}

// reportWithRetry the metrics to the url, every interval, with max retries.
func (p *Provider) reportWithRetry(u *url.URL, interval time.Duration) {
	nu := *u // copy the url
	requests, err := p.Batch(&nu, interval)
	if err != nil {
		p.errorHandler(err)
		return
	}
	var wg sync.WaitGroup
	for _, req := range requests {
		wg.Add(1)
		go func(req *http.Request) {
			defer wg.Done()
			for r := p.numRetries; r > 0; r-- {
				err := p.report(req)
				if err == nil {
					return
				}
				if terr, ok := err.(Error); ok {
					terr.retries = r - 1
					err = error(terr)
				}
				p.errorHandler(err)
				if err := p.backoff(r - 1); err != nil {
					return
				}
				// Not required with go1.9rc1
				if b, err := req.GetBody(); err == nil {
					req.Body = b
				}
			}
		}(req)
	}
	wg.Wait()
}

// report the request, which already has a Body containing metrics
func (p *Provider) report(req *http.Request) error {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if v := remainingRateLimit(resp.Header.Get("X-Librato-RateLimit-Agg")); v >= 0 {
		p.ratelimitAgg.Set(float64(v))
	}

	if v := remainingRateLimit(resp.Header.Get("X-Librato-RateLimit-Std")); v >= 0 {
		p.ratelimitStd.Set(float64(v))
	}

	if resp.StatusCode/100 != 2 {
		// Best effort, but don't fail on error
		d, _ := ioutil.ReadAll(resp.Body)

		e := Error{
			code:         resp.StatusCode,
			body:         string(d),
			rateLimitAgg: resp.Header.Get("X-Librato-RateLimit-Agg"),
			rateLimitStd: resp.Header.Get("X-Librato-RateLimit-Std"),
		}
		if p.requestDebugging {
			req.Header = scrub.Header(req.Header)

			// Best effort, but don't fail on error
			if b, err := req.GetBody(); err == nil {
				req.Body = b
			}
			d, _ := httputil.DumpRequestOut(req, true)
			e.dumpedRequest = string(d)
		}

		return e
	}
	return nil
}

func remainingRateLimit(s string) int {
	tuples := strings.Split(s, ",")
	for _, t := range tuples {
		chunks := strings.Split(t, "=")
		if len(chunks) == 2 && chunks[0] == "remaining" {
			n, err := strconv.Atoi(chunks[1])
			if err == nil {
				return n
			}
		}
	}
	return -1
}
