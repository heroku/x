/* Copyright (c) 2018 Salesforce
 * All rights reserved.
 * Licensed under the BSD 3-Clause license.
 * For full license text, see LICENSE.txt file in the repo root  or https://opensource.org/licenses/BSD-3-Clause
 */

package librato

import (
	"net/url"
	"strings"
	"sync"
	"time"

	kmetrics "github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/generic"
	"github.com/heroku/x/go-kit/metrics"
	xmetrics "github.com/heroku/x/go-kit/metrics"
	"gopkg.in/caio/go-tdigest.v2"
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

var (
	_ kmetrics.Histogram = &Histogram{}
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

	now         func() time.Time
	tagsEnabled bool
	defaultTags []string

	once          sync.Once
	done, stopped chan struct{}

	mu                  sync.Mutex
	counters            map[string]*Counter
	gauges              map[string]*Gauge
	histograms          map[string]*Histogram
	cardinalityCounters map[string]*CardinalityCounter

	measurements kmetrics.Gauge
	ratelimitAgg kmetrics.Gauge
	ratelimitStd kmetrics.Gauge
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

// WithTags allows the use of tags when submitting measurements. The default
// is to not allow it, and fall back to just sources.
func WithTags(labelValues ...string) OptionFunc {
	return func(p *Provider) {
		p.tagsEnabled = true
		p.defaultTags = append(p.defaultTags, labelValues...)
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

		counters:            make(map[string]*Counter),
		gauges:              make(map[string]*Gauge),
		histograms:          make(map[string]*Histogram),
		cardinalityCounters: make(map[string]*CardinalityCounter),

		now: time.Now,
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

	p.measurements = p.NewGauge("go-kit.measurements")
	p.ratelimitAgg = p.NewGauge("go-kit.ratelimit-aggregate")
	p.ratelimitStd = p.NewGauge("go-kit.ratelimit-standard")

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

// keyName is used as the map key for counters, gauges, and histograms
// and incorporates the name and the labelValues.
func keyName(name string, labelValues ...string) string {
	if len(labelValues) == 0 {
		return name
	}

	l := len(labelValues)
	parts := make([]string, 0, l/2)
	for i := 0; i < l; i += 2 {
		parts = append(parts, labelValues[i]+":"+labelValues[i+1])
	}
	return name + "." + strings.Join(parts, ".")
}

// metricName returns the name we'll use for the metric depending on
// whether we're using tags or not.
func (p *Provider) metricName(name string, labelValues ...string) string {
	if p.tagsEnabled {
		return name
	}
	return keyName(name, labelValues...)
}

// NewCounter that will be reported by the provider. Becuase of the way librato
// works, they are reported as gauges. If you require a counter reset every
// report use the WithResetCounters option function, otherwise the counter's
// value will increase until restart.
func (p *Provider) NewCounter(name string) kmetrics.Counter {
	return p.newCounter(prefixName(p.prefix, name), p.defaultTags...)
}

func (p *Provider) newCounter(name string, labelValues ...string) kmetrics.Counter {
	p.mu.Lock()
	defer p.mu.Unlock()

	k := keyName(name, labelValues...)
	if _, ok := p.counters[k]; !ok {
		gc := generic.NewCounter(name)

		if len(labelValues) > 0 {
			gc = gc.With(labelValues...).(*generic.Counter)
		}

		p.counters[k] = &Counter{
			Counter: gc,
			p:       p,
		}
	}

	return p.counters[k]
}

// NewGauge that will be reported by the provider.
func (p *Provider) NewGauge(name string) kmetrics.Gauge {
	return p.newGauge(prefixName(p.prefix, name), p.defaultTags...)
}

func (p *Provider) newGauge(name string, labelValues ...string) kmetrics.Gauge {
	p.mu.Lock()
	defer p.mu.Unlock()

	k := keyName(name, labelValues...)
	if _, ok := p.gauges[k]; !ok {
		gg := generic.NewGauge(name)

		if len(labelValues) > 0 {
			gg = gg.With(labelValues...).(*generic.Gauge)
		}

		p.gauges[k] = &Gauge{
			Gauge: gg,
			p:     p,
		}
	}

	return p.gauges[k]
}

// NewHistogram that will be reported by the provider.
func (p *Provider) NewHistogram(name string, buckets int) kmetrics.Histogram {
	return p.newHistogram(prefixName(p.prefix, name), buckets, p.percentilePrefix, p.defaultTags...)
}

func (p *Provider) newHistogram(name string, buckets int, percentilePrefix string, labelValues ...string) kmetrics.Histogram {
	p.mu.Lock()
	defer p.mu.Unlock()

	k := keyName(name, labelValues...)
	if _, ok := p.histograms[k]; !ok {
		h := &Histogram{
			p:                p,
			name:             name,
			buckets:          buckets,
			percentilePrefix: percentilePrefix,
			labelValues:      labelValues,
		}
		h.reset()

		p.histograms[k] = h
	}

	return p.histograms[k]
}

// NewCardinalityCounter that will be reported by the provider.
func (p *Provider) NewCardinalityCounter(name string) xmetrics.CardinalityCounter {
	return p.newCardinalityCounter(prefixName(p.prefix, name), p.defaultTags...)
}

func (p *Provider) newCardinalityCounter(name string, labelValues ...string) xmetrics.CardinalityCounter {
	p.mu.Lock()
	defer p.mu.Unlock()

	k := keyName(name, labelValues...)
	if _, ok := p.cardinalityCounters[k]; !ok {
		c := &CardinalityCounter{
			HLLCounter: xmetrics.NewHLLCounter(name).With(labelValues...).(*xmetrics.HLLCounter),
			p:          p,
		}

		p.cardinalityCounters[k] = c
	}

	return p.cardinalityCounters[k]
}

// Histogram adapts go-kit/Heroku/Librato's ideas of histograms. It
// reports p99, p95 and p50 values as gauges in addition to a gauge for the
// histogram itself.
type Histogram struct {
	p *Provider

	buckets          int
	name             string
	percentilePrefix string
	labelValues      []string

	mu sync.RWMutex

	h                          *tdigest.TDigest
	sum, min, max, sumsq, last float64
	count                      int64
}

func (h *Histogram) metricName() string {
	return h.p.metricName(h.name, h.labelValues...)
}

// Observe some data for the histogram
func (h *Histogram) Observe(value float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.count++
	h.sum += value
	if value < h.min || h.min == 0 {
		h.min = value
	}
	if value > h.max {
		h.max = value
	}
	h.sumsq += value * value
	h.last = value
	h.h.Add(value)
}

// With returns a new LibratoHistogram with the same name / buckets but it is
// otherwise a noop
func (h *Histogram) With(labelValues ...string) kmetrics.Histogram {
	lvs := append(append([]string(nil), h.labelValues...), labelValues...)
	return h.p.newHistogram(h.name, h.buckets, h.percentilePrefix, lvs...)
}

func (h *Histogram) reset() {
	// errors only happen if you pass in wrong options. We're passing no
	// options so there's zero chance of getting an error here.
	h.h, _ = tdigest.New()
	h.count = 0
	h.sum = 0
	h.min = 0
	h.max = 0
	h.sumsq = 0
}

// Quantile percentage of reported Observations
func (h *Histogram) Quantile(q float64) float64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.h.Quantile(q)
}

// Count of Observations
func (h *Histogram) Count() int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.count
}

// Min Observation
func (h *Histogram) Min() float64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.min
}

// Max Observation
func (h *Histogram) Max() float64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.max
}

// Sum of the Observations
func (h *Histogram) Sum() float64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.sum
}

// SumSq of the Observations
func (h *Histogram) SumSq() float64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.sumsq
}

// Counter is a wrapper on generic.Counter which stores a reference to the
// underlying Provider.
type Counter struct {
	*generic.Counter
	p *Provider
}

// Add implements Counter.
func (c *Counter) Add(delta float64) {
	c.Counter.Add(delta)
}

// With returns a Counter with the label values applied. Depending on whether
// you're using tags or not, the label values may be applied to the name.
func (c *Counter) With(labelValues ...string) kmetrics.Counter {
	lvs := append(append([]string(nil), c.LabelValues()...), labelValues...)
	return c.p.newCounter(c.Counter.Name, lvs...)
}

func (c *Counter) metricName() string {
	return c.p.metricName(c.Name, c.LabelValues()...)
}

// Gauge is a wrapper on generic.Gauge which stores a reference to the
// underlying Provider.
type Gauge struct {
	*generic.Gauge
	p *Provider
}

// Set implements Gauge.
func (g *Gauge) Set(value float64) {
	g.Gauge.Set(value)
}

// Add implements Gauge.
func (g *Gauge) Add(delta float64) {
	g.Gauge.Add(delta)
}

// With returns a Gauge with the label values applied. Depending on whether
// you're using tags or not, the label values may be applied to the name.
func (g *Gauge) With(labelValues ...string) kmetrics.Gauge {
	lvs := append(append([]string(nil), g.LabelValues()...), labelValues...)
	return g.p.newGauge(g.Gauge.Name, lvs...)
}

func (g *Gauge) metricName() string {
	return g.p.metricName(g.Name, g.LabelValues()...)
}

// CardinalityCounter is a wrapper on xmetrics.CardinalityCounter which stores
// a reference to the underlying Provider.
type CardinalityCounter struct {
	*xmetrics.HLLCounter
	p *Provider
}

// With returns a CardinalityCounter with the label values applied. Depending
// on whether you're using tags or not, the label values may be applied to the
// name.
func (c *CardinalityCounter) With(labelValues ...string) xmetrics.CardinalityCounter {
	lvs := append(append([]string(nil), c.LabelValues()...), labelValues...)
	return c.p.newCardinalityCounter(c.HLLCounter.Name, lvs...)
}

// Insert implements CardinalityCounter.
func (c *CardinalityCounter) Insert(b []byte) {
	c.HLLCounter.Insert(b)
}

func (c *CardinalityCounter) metricName() string {
	return c.p.metricName(c.Name, c.LabelValues()...)
}
