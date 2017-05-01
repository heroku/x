package provider

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
	"github.com/heroku/metaas/internal/metrics"
)

const (
	// DefaultBucketCount if you don't know what to use for the Histogram's bucket count.
	DefaultBucketCount = 50
	// DefaultLibratoURL for reporting metrics.
	DefaultLibratoURL = "https://metrics-api.librato.com/v1/metrics"
)

var (
	_ metrics.Provider   = &Librato{}
	_ kmetrics.Histogram = &LibratoHistogram{}
)

// LibratoError is used to report information from a non 200 error returned by Librato.
type LibratoError struct {
	code                             int
	body, rateLimitAgg, rateLimitStd string
}

// Code returned by librato
func (e LibratoError) Code() int {
	return e.code
}

// RateLimit info returned by librato in the X-Librato-RateLimit-Agg and
// X-Librato-RateLimit-Std headers
func (e LibratoError) RateLimit() (string, string) {
	return e.rateLimitAgg, e.rateLimitStd
}

// Body returned by librato.
func (e LibratoError) Body() string {
	return e.body
}

// Error interface
func (e LibratoError) Error() string {
	return fmt.Sprintf("code: %d, body: %s, rate-limit-agg: %s, rate-limit-std: %s", e.code, e.body, e.rateLimitAgg, e.rateLimitStd)
}

// Librato go-kit metrics provider. Works with Librato's older source based
// metrics (http://api-docs-archive.librato.com/?shell#create-a-metric, not the
// new tag based metrcis). The generated metric's With methods return new
// metrics, but are otherwise noops as the LabelValues are not applied in any
// meaningful way.
type Librato struct {
	errors chan error
	prefix string

	once sync.Once
	done chan struct{}

	mu         sync.Mutex
	counters   []*generic.Counter
	gauges     []*generic.Gauge
	histograms []*LibratoHistogram
}

// Report the metrics to the url every interval with the given source. Cancel
// the context to stop reporting. The returned channel can be used to monitor
// errors, of which there will be a max of 1 every interval. If Librato responds
// with a non 2XX response code a LibratoError is returned. Callers need to
// drain the error channel or it will block reporting. The error channel is
// closed after a final report is sent. Callers should ensure there is only one
// Report operating at a time.
func NewLibrato(URL *url.URL, interval time.Duration, source, prefix string) *Librato {
	l := &Librato{
		prefix: prefix,
		errors: make(chan error),
		done:   make(chan struct{}),
	}

	go func() {
		t := time.NewTicker(interval)
		for {
			select {
			case <-t.C:
				err := l.report(URL, interval, source)
				if err != nil {
					l.errors <- err
				}
			case <-l.done:
				t.Stop()
				err := l.report(URL, interval, source)
				if err != nil {
					l.errors <- err
				}
				close(l.errors)
				return
			}
		}
	}()

	return l
}

func (l *Librato) Errors() chan error {
	return l.errors
}

// Stop reporting metrics
func (l *Librato) Stop() {
	l.once.Do(func() {
		close(l.done)
	})
}

// NewCounter for this librato provider.
func (l *Librato) NewCounter(name string) kmetrics.Counter {
	c := generic.NewCounter(l.prefix + "." + name)
	l.mu.Lock()
	defer l.mu.Unlock()
	l.counters = append(l.counters, c)
	return c
}

// NewGauge for this librato provider.
func (l *Librato) NewGauge(name string) kmetrics.Gauge {
	g := generic.NewGauge(l.prefix + "." + name)
	l.mu.Lock()
	defer l.mu.Unlock()
	l.gauges = append(l.gauges, g)
	return g
}

// NewHistogram for this librato provider.
func (l *Librato) NewHistogram(name string, buckets int) kmetrics.Histogram {
	h := LibratoHistogram{name: l.prefix + "." + name, buckets: buckets}
	h.reset()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.histograms = append(l.histograms, &h)
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

// report the metrics to the url, every interval, with the provided source
func (l *Librato) report(u *url.URL, interval time.Duration, source string) error {
	l.mu.Lock()
	defer l.mu.Unlock() // should only block New{Histogram,Counter,Gauge}

	if len(l.counters) == 0 || len(l.histograms) == 0 || len(l.gauges) == 0 {
		return nil
	}

	var buf bytes.Buffer
	e := json.NewEncoder(&buf)
	r := struct {
		Source      string    `json:"source"`
		MeasureTime int64     `json:"measure_time"`
		Counters    []counter `json:"counters"`
		Gauges      []gauge   `json:"gauges"`
	}{}
	r.Source = source
	ivSec := int64(interval / time.Second)
	r.MeasureTime = (time.Now().Unix() / ivSec) * ivSec
	period := interval.Seconds()

	for _, c := range l.counters {
		r.Counters = append(r.Counters, counter{Name: c.Name, Period: period, Value: c.Value()})
	}
	for _, g := range l.gauges {
		v := g.Value()
		r.Gauges = append(r.Gauges, gauge{Name: g.Name, Period: period, Count: 1, Sum: v, Min: v, Max: v, SumSq: v * v})
	}
	for _, h := range l.histograms {
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
		return LibratoError{
			resp.StatusCode,
			string(b),
			resp.Header.Get("X-Librato-RateLimit-Agg"),
			resp.Header.Get("X-Librato-RateLimit-Std"),
		}
	}
	return nil
}

// LibratoHistogram adapts go-kit/Heroku/Librato's ideas of histograms. It
// reports p99, p95 and p50 values as gauges in addition to a gauge for the
// histogram itself.
type LibratoHistogram struct {
	buckets int
	name    string

	mu sync.RWMutex
	// I would prefer to use hdrhistogram, but that's incompatible with the
	// go-metrics Histogram interface (int64 vs float64).
	h                    gohistogram.Histogram
	sum, min, max, sumsq float64
	count                int64
}

// the json marshalers for the histograms 4 different gauges
func (h *LibratoHistogram) measures(period float64) []gauge {
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
		{name + ".p99", h.h.Quantile(.99)},
		{name + ".p95", h.h.Quantile(.95)},
		{name + ".p50", h.h.Quantile(.50)},
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
func (h *LibratoHistogram) Observe(value float64) {
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
func (h *LibratoHistogram) With(lv ...string) kmetrics.Histogram {
	n := LibratoHistogram{name: h.name, buckets: h.buckets}
	n.reset()
	return &n
}

func (h *LibratoHistogram) reset() {
	// Not happy with this, but the existing histogram doesn't have a Reset.
	h.h = gohistogram.NewHistogram(h.buckets)
	h.count = 0
	h.sum = 0
	h.min = 0
	h.max = 0
	h.sumsq = 0
}

// Quantile percentage of reported Observations
func (h *LibratoHistogram) Quantile(q float64) float64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.h.Quantile(q)
}

// Count of Observations
func (h *LibratoHistogram) Count() int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.count
}

// Min Observation
func (h *LibratoHistogram) Min() float64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.min
}

// Max Observation
func (h *LibratoHistogram) Max() float64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.max
}

// Sum of the Observations
func (h *LibratoHistogram) Sum() float64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.sum
}

// SumSq of the Observations
func (h *LibratoHistogram) SumSq() float64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.sumsq
}
