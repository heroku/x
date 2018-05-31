/* Copyright (c) 2018 Salesforce
 * All rights reserved.
 * Licensed under the BSD 3-Clause license.
 * For full license text, see LICENSE.txt file in the repo root  or https://opensource.org/licenses/BSD-3-Clause
 */

package librato

import (
	"net/url"
	"sync"
	"time"

	kmetrics "github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/generic"
	"github.com/heroku/x/go-kit/metrics"
	xmetrics "github.com/heroku/x/go-kit/metrics"
)

const (
	// DefaultBucketCount is a reasonable default for the number of buckets a
	// Histogram should use.
	DefaultBucketCount = 50
	// DefaultURL for reporting metrics.
	DefaultURL = "https://metrics-api.librato.com/v1/metrics"
	// DefaultPercentilePrefix if WithPercentilePrefix isn't used to set a different prefix
	DefaultPercentilePrefix = ".p"
	// DefaultNumRetries if WithRetries isn't used to set a different value
	DefaultNumRetries = 3
	// DefaultBatchSize of metric batches sent to librato. This value was taken out of the librato docs at:
	// http://api-docs-archive.librato.com/?shell#measurement-properties
	DefaultBatchSize = 300
)

// Provider works with Librato's older source based
// metrics (http://api-docs-archive.librato.com/?shell#create-a-metric, not the
// new tag based metrics). The generated metric's With methods return new
// metrics, but are otherwise noops as the LabelValues are not applied in any
// meaningful way.
type Provider struct {
	errorHandler                     func(err error)
	backoff                          func(r int) error
	source, prefix, percentilePrefix string
	numRetries                       int
	batchSize                        int
	resetCounters                    bool
	ssa                              bool
	requestDebugging                 bool

	once          sync.Once
	done, stopped chan struct{}

	mu             sync.Mutex
	counters       []*generic.Counter
	gauges         []*generic.Gauge
	histograms     []*Histogram
	uniqueCounters []*HLLCounter
}

// OptionFunc used to set options on a librato provider
type OptionFunc func(*Provider)

// WithBatchSize sets the number of metrics sent in a single request to librato
func WithBatchSize(n int) OptionFunc {
	return func(p *Provider) {
		p.batchSize = n
	}
}

// WithRetries sets the max number of retries during reporting.
func WithRetries(n int) OptionFunc {
	return func(p *Provider) {
		p.numRetries = n
	}
}

// WithRequestDebugging enables request debugging which exposes the original
// request in the Error.
func WithRequestDebugging() OptionFunc {
	return func(p *Provider) {
		p.requestDebugging = true
	}
}

// WithSSA turns on Server Side Aggreggation for all gauges submitted.
func WithSSA() OptionFunc {
	return func(p *Provider) {
		p.ssa = true
	}
}

// WithPercentilePrefix sets the optional percentile prefix used for the
// different percentile gauges reported for each histogram. The default is `.p`,
// meaning the name of those gauges will be:
//
//  <histogram metric name>.p50
//  <histogram metric name>.p95
//  <histogram metric name>.p99
func WithPercentilePrefix(prefix string) OptionFunc {
	return func(p *Provider) {
		p.percentilePrefix = prefix
	}
}

// WithResetCounters makes the reporting behavior reset all the counters every
// reporting interval. Use this option if you're trying to be compatible with
// l2met (e.g. you previously had l2met metrics which exhibited the same
// behavior).
func WithResetCounters() OptionFunc {
	return func(p *Provider) {
		p.resetCounters = true
	}
}

// WithSource sets the optional provided source for the librato provider
func WithSource(source string) OptionFunc {
	return func(p *Provider) {
		p.source = source
	}
}

// WithPrefix sets the optional metrics prefix for the librato provider. If the
// prefix is != "" then it is prefixed to each metric name when reported.
func WithPrefix(prefix string) OptionFunc {
	return func(p *Provider) {
		p.prefix = prefix
	}
}

// WithErrorHandler sets the optional error handler used to report errors. Use
// this to log, or otherwise handle reporting errors in your application.
func WithErrorHandler(eh func(err error)) OptionFunc {
	return func(p *Provider) {
		p.errorHandler = eh

	}
}

// WithBackoff sets the optional backoff handler. The backoffhandler should sleep
// for the amount of time required between retries. The backoff func receives the
// current number of retries remaining. Returning an error from the backoff func
// stops additional retries for that request.
//
// The default backoff strategy is 100ms * (total # of tries - retries remaining)
func WithBackoff(b func(r int) error) OptionFunc {
	return func(p *Provider) {
		p.backoff = b
	}
}

func defaultErrorHandler(err error) {}

// New librato metrics provider that reports metrics to the URL every interval
// with the provided options.
func New(URL *url.URL, interval time.Duration, opts ...OptionFunc) metrics.Provider {
	p := Provider{
		errorHandler:     defaultErrorHandler,
		done:             make(chan struct{}),
		stopped:          make(chan struct{}),
		percentilePrefix: DefaultPercentilePrefix,
		numRetries:       DefaultNumRetries,
		batchSize:        DefaultBatchSize,
	}

	for _, opt := range opts {
		opt(&p)
	}

	if p.backoff == nil {
		p.backoff = func(r int) error {
			time.Sleep((time.Duration(p.numRetries-r) * 100) * time.Millisecond)
			return nil
		}
	}

	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				p.reportWithRetry(URL, interval)
			case <-p.done:
				p.reportWithRetry(URL, interval)
				close(p.stopped)
				return
			}
		}
	}()

	return &p
}

// Stop reporting metrics
func (p *Provider) Stop() {
	p.once.Do(func() {
		close(p.done)
		<-p.stopped
	})
}

func prefixName(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}

// NewCounter that will be reported by the provider. Becuase of the way librato
// works, they are reported as gauges. If you require a counter reset every
// report use the WithResetCounters option function, otherwise the counter's
// value will increase until restart.
func (p *Provider) NewCounter(name string) kmetrics.Counter {
	c := generic.NewCounter(prefixName(p.prefix, name))
	p.mu.Lock()
	defer p.mu.Unlock()
	p.counters = append(p.counters, c)
	return c
}

// NewGauge that will be reported by the provider.
func (p *Provider) NewGauge(name string) kmetrics.Gauge {
	g := generic.NewGauge(prefixName(p.prefix, name))
	p.mu.Lock()
	defer p.mu.Unlock()
	p.gauges = append(p.gauges, g)
	return g
}

// NewHistogram that will be reported by the provider.
func (p *Provider) NewHistogram(name string, buckets int) kmetrics.Histogram {
	h := Histogram{name: prefixName(p.prefix, name), buckets: buckets, percentilePrefix: p.percentilePrefix}
	h.reset()
	p.mu.Lock()
	defer p.mu.Unlock()
	p.histograms = append(p.histograms, &h)
	return &h
}

// NewUniqueCounter that will be reported by the provider.
func (p *Provider) NewUniqueCounter(name string) xmetrics.UniqueCounter {
	c := NewHLLCounter(name)
	p.mu.Lock()
	defer p.mu.Unlock()
	p.uniqueCounters = append(p.uniqueCounters, c)
	return c
}
