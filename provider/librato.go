package provider

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"time"

	"encoding/json"

	"strconv"

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
	mu         sync.Mutex
	counters   []*generic.Counter
	gauges     []*generic.Gauge
	histograms []*LibratoHistogram
}

// Report the metrics to the url every interval with the given source. Cancel
// the context to stop reporting. The returned channel can be used to monitor
// errors, of which there will be a max of 1 every interval. If Librato responds
// with a non 2XX response code a LibratoError is returned. Callers need to
// drain the error channel or it will block reporting. Callers should ensure
// there is only one Report operating at a time.
func (l *Librato) Report(ctx context.Context, url *url.URL, interval time.Duration, source string) <-chan error {
	errors := make(chan error)
	go func() {
		t := time.NewTicker(interval)
		for {
			select {
			case <-t.C:
				err := l.report(url, interval, source)
				if err != nil {
					errors <- err
				}
			case <-ctx.Done():
				t.Stop()
				err := l.report(url, interval, source)
				if err != nil {
					errors <- err
				}
				return
			}
		}
	}()

	return errors
}

// Stop does nothing for this provider, cancel the context passed to the Report
// method to stop the reporter.
func (l *Librato) Stop() {}

// NewCounter for this librato provider.
func (l *Librato) NewCounter(name string) kmetrics.Counter {
	c := generic.NewCounter(name)
	l.mu.Lock()
	defer l.mu.Unlock()
	l.counters = append(l.counters, c)
	return c
}

// NewGauge for this librato provider.
func (l *Librato) NewGauge(name string) kmetrics.Gauge {
	g := generic.NewGauge(name)
	l.mu.Lock()
	defer l.mu.Unlock()
	l.gauges = append(l.gauges, g)
	return g
}

// NewHistogram for this librato provider.
func (l *Librato) NewHistogram(name string, buckets int) kmetrics.Histogram {
	h := LibratoHistogram{name: name, buckets: buckets}
	h.reset()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.histograms = append(l.histograms, &h)
	return &h
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
		Source      string           `json:"source"`
		MeasureTime int64            `json:"measure_time"`
		Counters    []json.Marshaler `json:"counters"`
		Gauges      []json.Marshaler `json:"gauges"`
	}{}
	r.Source = source
	ivSec := int64(interval / time.Second)
	r.MeasureTime = (time.Now().Unix() / ivSec) * ivSec
	period := interval.Seconds()

	for _, c := range l.counters {
		r.Counters = append(r.Counters, marshalGeneric(c.Name, c.Value(), period))
	}
	for _, g := range l.gauges {
		r.Gauges = append(r.Gauges, marshalGeneric(g.Name, g.Value(), period))
	}
	for i := range l.histograms {
		parts := l.histograms[i].jsonMarshalers(period)
		r.Gauges = append(r.Gauges, parts...)
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

// Create a raw json metrics object with the provided name, value and period.
func marshalGeneric(name string, value, period float64) json.RawMessage {
	b := make([]byte, 0, 100)
	b = append(b, `{"name":"`...)
	b = append(b, name...)
	b = append(b, `","value":`...)
	b = strconv.AppendFloat(b, value, 'f', 6, 64)
	b = append(b, `,"period":`...)
	b = strconv.AppendFloat(b, period, 'f', 3, 64)
	b = append(b, '}')
	return json.RawMessage(b)
}

// LibratoHistogram adapts go-kit/Heroku/Librato's ideas of histograms. It
// reports p99, p95 and p50 values as seperate simple gauges as well as a single
// complex gauge.
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
func (h *LibratoHistogram) jsonMarshalers(period float64) []json.Marshaler {
	h.mu.Lock()
	count := h.count
	sum := h.sum
	min := h.min
	max := h.max
	sumsq := h.sumsq
	percs := []struct {
		p string
		v float64
	}{
		{"p99", h.h.Quantile(.99)},
		{"p95", h.h.Quantile(.95)},
		{"p50", h.h.Quantile(.50)},
	}
	h.reset()
	h.mu.Unlock()

	msg := make([]byte, 0, 200)
	msg = append(msg, `{"name":"`...)
	msg = append(msg, h.name...)
	msg = append(msg, `","count":`...)
	msg = strconv.AppendInt(msg, count, 10)
	msg = append(msg, `,"sum":`...)
	msg = strconv.AppendFloat(msg, sum, 'f', 6, 64)
	msg = append(msg, `,"min":`...)
	msg = strconv.AppendFloat(msg, min, 'f', 6, 64)
	msg = append(msg, `,"max":`...)
	msg = strconv.AppendFloat(msg, max, 'f', 6, 64)
	msg = append(msg, `,"sum_squares":`...)
	msg = strconv.AppendFloat(msg, sumsq, 'f', 6, 64)
	msg = append(msg, '}')

	msgs := []json.Marshaler{json.RawMessage(msg)}
	for _, perc := range percs {
		msgs = append(msgs, json.RawMessage(marshalGeneric(h.name+"."+perc.p, perc.v, period)))
	}

	return msgs
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
