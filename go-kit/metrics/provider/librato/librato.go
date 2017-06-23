package librato

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/VividCortex/gohistogram"
	kmetrics "github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/generic"
	"github.com/heroku/x/go-kit/metrics"
)

const (
	// DefaultBucketCount is a reasonable default for the number of buckets a
	// Histogram should use.
	DefaultBucketCount = 50
	// DefaultURL for reporting metrics.
	DefaultURL = "https://metrics-api.librato.com/v1/metrics"
)

var (
	_ kmetrics.Histogram = &Histogram{}
)

// Error is used to report information from a non 200 error returned by Librato.
type Error struct {
	code                             int
	body, rateLimitAgg, rateLimitStd string
}

// Code returned by librato
func (e Error) Code() int {
	return e.code
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
	return fmt.Sprintf("code: %d, body: %s, rate-limit-agg: %s, rate-limit-std: %s", e.code, e.body, e.rateLimitAgg, e.rateLimitStd)
}

// Provider works with Librato's older source based
// metrics (http://api-docs-archive.librato.com/?shell#create-a-metric, not the
// new tag based metrcis). The generated metric's With methods return new
// metrics, but are otherwise noops as the LabelValues are not applied in any
// meaningful way.
type Provider struct {
	errorHandler                     func(err error)
	source, prefix, percentilePrefix string
	resetCounters, ssa               bool
	numRetries                       int

	once sync.Once
	done chan struct{}

	mu         sync.Mutex
	counters   []*generic.Counter
	gauges     []*generic.Gauge
	histograms []*Histogram
}

// with the given source. If the prefix is != "" then it is prefixed to each
// reported metric.
// the context to stop reporting. The returned channel can be used to monitor
// errors, of which there will be a max of 1 every interval. If Librato responds
// with a non 2XX response code a LibratoError is returned. Callers need to
// drain the error channel or it will block reporting. The error channel is
// closed after a final report is sent. Callers should ensure there is only one
// Report operating at a time.

// OptionFunc used to set options on a librato provider
type OptionFunc func(*Provider)

// WithRetries sets the max number of retries during reporting.
func WithRetries(n int) OptionFunc {
	return func(p *Provider) {
		p.numRetries = n
	}
}

// WithSSA turns on SSA for all gauges submitted.
func WithSSA() OptionFunc {
	return func(p *Provider) {
		p.ssa = true
	}
}

// WithPercentilePrefix sets the optional percentile prefix.
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

// WithPrefix sets the optional metrics prefix for the librato provider
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

const (
	defaultPercentilePrefix = ".p"
	defaultNumRetries       = 3
)

// New metrics provider that reports metrics to the URL every interval.
func New(URL *url.URL, interval time.Duration, opts ...OptionFunc) metrics.Provider {
	p := Provider{
		done:             make(chan struct{}),
		percentilePrefix: defaultPercentilePrefix,
		numRetries:       defaultNumRetries,
	}

	for _, opt := range opts {
		opt(&p)
	}

	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				err := p.reportWithRetry(URL, interval)
				if err != nil && p.errorHandler != nil {
					p.errorHandler(err)
				}
			case <-p.done:
				err := p.reportWithRetry(URL, interval)
				if err != nil && p.errorHandler != nil {
					p.errorHandler(err)
				}
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
	})
}

func prefixName(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}

// NewCounter for this librato provider.
func (p *Provider) NewCounter(name string) kmetrics.Counter {
	c := generic.NewCounter(prefixName(p.prefix, name))
	p.mu.Lock()
	defer p.mu.Unlock()
	p.counters = append(p.counters, c)
	return c
}

// NewGauge for this librato provider.
func (p *Provider) NewGauge(name string) kmetrics.Gauge {
	g := generic.NewGauge(prefixName(p.prefix, name))
	p.mu.Lock()
	defer p.mu.Unlock()
	p.gauges = append(p.gauges, g)
	return g
}

// NewHistogram for this librato provider.
func (p *Provider) NewHistogram(name string, buckets int) kmetrics.Histogram {
	h := Histogram{name: prefixName(p.prefix, name), buckets: buckets, percentilePrefix: p.percentilePrefix}
	h.reset()
	p.mu.Lock()
	defer p.mu.Unlock()
	p.histograms = append(p.histograms, &h)
	return &h
}

// librato counter and gauge structs for json Marshaling
type counter struct {
	Name   string  `json:"name"`
	Period float64 `json:"period"`
	Value  float64 `json:"value"`
}

// extended librato gauge format is used for all gauges in order to keep the coe
// base simple
type gauge struct {
	Name   string  `json:"name"`
	Period float64 `json:"period"`
	Count  int64   `json:"count"`
	Sum    float64 `json:"sum"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	SumSq  float64 `json:"sum_squares"`
}

// attributes are top level things which you can use to affect newly created
// metrics.
type attributes struct {
	Aggregate bool `json:"aggregate,omitempty"`
}

// reportWithRetry the metrics to the url, every interval, with max retries.
func (p *Provider) reportWithRetry(u *url.URL, interval time.Duration) error {
	var err error

	for i := p.numRetries; i > 0; i-- {
		if err = p.report(u, interval); err == nil {
			return nil
		}
	}

	return err
}

// report the metrics to the url, every interval
func (p *Provider) report(u *url.URL, interval time.Duration) error {
	p.mu.Lock()
	defer p.mu.Unlock() // should only block New{Histogram,Counter,Gauge}

	if len(p.counters) == 0 && len(p.histograms) == 0 && len(p.gauges) == 0 {
		return nil
	}

	var buf bytes.Buffer
	e := json.NewEncoder(&buf)
	r := struct {
		Source      string      `json:"source,omitempty"`
		MeasureTime int64       `json:"measure_time"`
		Counters    []counter   `json:"counters"`
		Gauges      []gauge     `json:"gauges"`
		Attributes  *attributes `json:"attributes,omitempty"`
	}{}

	r.Source = p.source
	ivSec := int64(interval / time.Second)
	r.MeasureTime = (time.Now().Unix() / ivSec) * ivSec
	if p.ssa {
		r.Attributes = &attributes{Aggregate: true}
	}
	period := interval.Seconds()

	for _, c := range p.counters {
		var v float64
		if p.resetCounters {
			v = c.ValueReset()
		} else {
			v = c.Value()
		}
		r.Gauges = append(r.Gauges, gauge{Name: c.Name, Period: period, Count: 1, Sum: v, Min: v, Max: v, SumSq: v * v})
	}
	for _, g := range p.gauges {
		v := g.Value()
		r.Gauges = append(r.Gauges, gauge{Name: g.Name, Period: period, Count: 1, Sum: v, Min: v, Max: v, SumSq: v * v})
	}
	for _, h := range p.histograms {
		r.Gauges = append(r.Gauges, h.measures(period)...)
	}

	if err := e.Encode(r); err != nil {
		return err
	}
	resp, err := http.Post(u.String(), "application/json", &buf)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		b, _ := ioutil.ReadAll(resp.Body)
		return Error{
			resp.StatusCode,
			string(b),
			resp.Header.Get("X-Librato-RateLimit-Agg"),
			resp.Header.Get("X-Librato-RateLimit-Std"),
		}
	}
	return nil
}

// Histogram adapts go-kit/Heroku/Librato's ideas of histograms. It
// reports p99, p95 and p50 values as gauges in addition to a gauge for the
// histogram itself.
type Histogram struct {
	buckets          int
	name             string
	percentilePrefix string

	mu sync.RWMutex
	// I would prefer to use hdrhistogram, but that's incompatible with the
	// go-metrics Histogram interface (int64 vs float64).
	h                    gohistogram.Histogram
	sum, min, max, sumsq float64
	count                int64
}

// the json marshalers for the histograms 4 different gauges
func (h *Histogram) measures(period float64) []gauge {
	h.mu.Lock()
	if h.count == 0 {
		h.mu.Unlock()
		return nil
	}
	count := h.count
	sum := h.sum
	min := h.min
	max := h.max
	sumsq := h.sumsq
	name := h.name
	percs := []struct {
		n string
		v float64
	}{
		{name + h.percentilePrefix + "99", h.h.Quantile(.99)},
		{name + h.percentilePrefix + "95", h.h.Quantile(.95)},
		{name + h.percentilePrefix + "50", h.h.Quantile(.50)},
	}
	h.reset()
	h.mu.Unlock()

	m := make([]gauge, 0, 4)
	m = append(m,
		gauge{Name: name, Period: period, Count: count, Sum: sum, Min: min, Max: max, SumSq: sumsq},
	)

	for _, perc := range percs {
		m = append(m, gauge{Name: perc.n, Period: period, Count: 1, Sum: perc.v, Min: perc.v, Max: perc.v, SumSq: perc.v * perc.v})
	}
	return m
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
	h.h.Add(value)
}

// With returns a new LibratoHistogram with the same name / buckets but it is
// otherwise a noop
func (h *Histogram) With(lv ...string) kmetrics.Histogram {
	n := Histogram{name: h.name, buckets: h.buckets}
	n.reset()
	return &n
}

func (h *Histogram) reset() {
	// Not happy with this, but the existing histogram doesn't have a Reset.
	h.h = gohistogram.NewHistogram(h.buckets)
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
