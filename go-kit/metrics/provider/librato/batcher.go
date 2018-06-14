package librato

import (
	"bytes"
	"encoding/json"
	"math"
	"net/http"
	"net/url"
	"time"
)

const (
	oldBatcherPath = "/v1/metrics"
	tagBatcherPath = "/v1/measurements"
)

type batcher interface {
	Batch(URL *url.URL, interval time.Duration) ([]*http.Request, error)
}

type oldBatcher struct {
	p *Provider
}

// Batch will batch up all the metrics into []*http.Requests using the old style API.
func (b *oldBatcher) Batch(u *url.URL, interval time.Duration) ([]*http.Request, error) {
	// Calculate the sample time.
	st := time.Now().Truncate(interval).Unix()

	// Sample the metrics.
	gauges := b.sample(int(interval.Seconds()))

	if len(gauges) == 0 { // no data to report
		return nil, nil
	}

	// Don't accidentally leak the creds, which can happen if we return the u with a u.User set
	var user *url.Userinfo
	user, u.User = u.User, nil

	u = u.ResolveReference(&url.URL{Path: oldBatcherPath})

	nextEnd := func(e int) int {
		e += b.p.batchSize
		if l := len(gauges); e > l {
			return l
		}
		return e
	}

	requests := make([]*http.Request, 0, len(gauges)/b.p.batchSize+1)
	for batch, e := 0, nextEnd(0); batch < len(gauges); batch, e = e, nextEnd(e) {
		r := struct {
			Source      string                 `json:"source,omitempty"`
			MeasureTime int64                  `json:"measure_time"`
			Gauges      []gauge                `json:"gauges"`
			Attributes  map[string]interface{} `json:"attributes,omitempty"`
		}{
			Source:      b.p.source,
			MeasureTime: st,
			Gauges:      gauges[batch:e],
		}
		if b.p.ssa {
			r.Attributes = map[string]interface{}{"aggregate": true}
		}

		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(r); err != nil {
			return nil, err
		}

		req, err := http.NewRequest(http.MethodPost, u.String(), &buf)
		if err != nil {
			return nil, err
		}
		if user != nil {
			p, _ := user.Password()
			req.SetBasicAuth(user.Username(), p)
		}
		req.Header.Set("Content-Type", "application/json")
		requests = append(requests, req)
	}

	return requests, nil
}

// extended librato gauge format is used for all metric types in the old batcher.
type gauge struct {
	Name   string  `json:"name"`
	Period int     `json:"period"`
	Count  int64   `json:"count"`
	Sum    float64 `json:"sum"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	SumSq  float64 `json:"sum_squares"`
}

// sample the metrics
func (b *oldBatcher) sample(period int) []gauge {
	b.p.mu.Lock()
	defer b.p.mu.Unlock() // should only block New{Histogram,Counter,Gauge,Cardinalityounter}

	if len(b.p.counters) == 0 && len(b.p.histograms) == 0 && len(b.p.gauges) == 0 && len(b.p.cardinalityCounters) == 0 {
		return nil
	}

	// Assemble all the data we have to send
	var gauges []gauge
	for _, c := range b.p.counters {
		var v float64
		if b.p.resetCounters {
			v = c.ValueReset()
		} else {
			v = c.Value()
		}
		gauges = append(gauges, gauge{Name: c.metricName(), Period: period, Count: 1, Sum: v, Min: v, Max: v, SumSq: v * v})
	}

	for _, g := range b.p.gauges {
		v := g.Value()

		gauges = append(gauges, gauge{Name: g.metricName(), Period: period, Count: 1, Sum: v, Min: v, Max: v, SumSq: v * v})
	}

	for _, h := range b.p.histograms {
		gauges = append(gauges, b.histogramMeasures(h, period)...)
	}

	for _, c := range b.p.cardinalityCounters {
		var v float64
		if b.p.resetCounters {
			v = float64(c.EstimateReset())
		} else {
			v = float64(c.Estimate())
		}
		gauges = append(gauges, gauge{Name: c.Name, Period: period, Count: 1, Sum: v, Min: v, Max: v, SumSq: v * v})
	}

	return gauges
}

// the json marshalers for the histograms 4 different gauges
func (b *oldBatcher) histogramMeasures(h *Histogram, period int) []gauge {
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
	name := h.metricName()
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

type taggedBatcher struct {
	p *Provider
}

func (b *taggedBatcher) Batch(u *url.URL, interval time.Duration) ([]*http.Request, error) {
	// Sample the metrics.
	measurements := b.sample(int(interval.Seconds()))

	if len(measurements) == 0 { // no data to report
		return nil, nil
	}

	// Don't accidentally leak the creds, which can happen if we return the u with a u.User set
	var user *url.Userinfo
	user, u.User = u.User, nil

	u = u.ResolveReference(&url.URL{Path: tagBatcherPath})

	nextEnd := func(e int) int {
		e += b.p.batchSize
		if l := len(measurements); e > l {
			return l
		}
		return e
	}

	requests := make([]*http.Request, 0, len(measurements)/b.p.batchSize+1)
	for batch, e := 0, nextEnd(0); batch < len(measurements); batch, e = e, nextEnd(e) {
		r := struct {
			Measurements []measurement `json:"measurements"`
		}{
			Measurements: measurements[batch:e],
		}

		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(r); err != nil {
			return nil, err
		}

		req, err := http.NewRequest(http.MethodPost, u.String(), &buf)
		if err != nil {
			return nil, err
		}
		if user != nil {
			p, _ := user.Password()
			req.SetBasicAuth(user.Username(), p)
		}
		req.Header.Set("Content-Type", "application/json")
		requests = append(requests, req)
	}

	return requests, nil
}

func (b *taggedBatcher) tagsFor(labelValues ...string) map[string]string {
	if len(labelValues) == 0 {
		return map[string]string{"source": b.p.source}
	}
	return labelValuesToTags(labelValues...)
}

// sample the metrics
func (b *taggedBatcher) sample(period int) []measurement {
	b.p.mu.Lock()
	defer b.p.mu.Unlock() // should only block New{Histogram,Counter,Gauge,Cardinalityounter}

	// TODO: also add cardinality counters.
	if len(b.p.counters) == 0 && len(b.p.histograms) == 0 && len(b.p.gauges) == 0 {
		return nil
	}

	var attrs map[string]interface{}
	if b.p.ssa {
		attrs = map[string]interface{}{"aggregate": true}
	}

	ts := time.Now().Truncate(time.Second * time.Duration(period))

	// Assemble all the data we have to send
	var measurements []measurement
	for _, c := range b.p.counters {
		var v float64
		if b.p.resetCounters {
			v = c.ValueReset()
		} else {
			v = c.Value()
		}

		measurements = append(measurements, measurement{
			Name:       c.Name,
			Time:       ts.Unix(),
			Period:     period,
			Count:      1,
			Sum:        v,
			Min:        v,
			Max:        v,
			Last:       v,
			StdDev:     0,
			Tags:       b.tagsFor(c.LabelValues()...),
			Attributes: attrs,
		})
	}

	for _, g := range b.p.gauges {
		v := g.Value()
		measurements = append(measurements, measurement{
			Name:       g.Name,
			Time:       ts.Unix(),
			Period:     period,
			Count:      1,
			Sum:        v,
			Min:        v,
			Max:        v,
			Last:       v,
			StdDev:     0,
			Tags:       b.tagsFor(g.LabelValues()...),
			Attributes: attrs,
		})
	}

	for _, h := range b.p.histograms {
		measurements = append(measurements, b.histogramMeasures(h, period)...)
	}

	return measurements
}

// the json marshalers for the histograms 4 different gauges
func (b *taggedBatcher) histogramMeasures(h *Histogram, period int) []measurement {
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
	stddev := stddev(sum, sumsq, count)
	last := h.last
	name := h.metricName()
	ts := time.Now().Truncate(time.Second * time.Duration(period))

	var attrs map[string]interface{}
	if b.p.ssa {
		attrs = map[string]interface{}{"aggregate": true}
	}

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

	m := make([]measurement, 0, 4)
	m = append(m, measurement{
		Name:       name,
		Period:     period,
		Time:       ts.Unix(),
		Count:      count,
		Sum:        sum,
		Min:        min,
		Max:        max,
		Last:       last,
		StdDev:     stddev,
		Attributes: attrs,
		Tags:       b.tagsFor(h.labelValues...),
	})

	for _, perc := range percs {
		m = append(m, measurement{
			Name:   perc.n,
			Period: period,
			Time:   ts.Unix(),
			Count:  1,
			Sum:    perc.v,
			Min:    perc.v,
			Max:    perc.v,
			Last:   perc.v,
			StdDev: 0,
		})
	}

	return m
}

type measurement struct {
	Name   string `json:"name"`
	Time   int64  `json:"time"`
	Period int    `json:"period"`

	Attributes map[string]interface{} `json:"attributes,omitempty"`
	Tags       map[string]string      `json:"tags"`

	Sum    float64 `json:"sum"`
	Count  int64   `json:"count"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	Last   float64 `json:"last"`
	StdDev float64 `json:"stddev"`
}

func labelValuesToTags(labelValues ...string) map[string]string {
	res := make(map[string]string)
	l := len(labelValues)
	for i := 0; i < l; i += 2 {
		res[labelValues[i]] = labelValues[i+1]
	}
	return res
}

// The square of the distance from the mean is necessary in calculating
// standard deviation. It's expressed as:
//
//   Σ (x - μ)²
//
// When doing time series datasets, we typically only hold on to the sum,
// sum of squares, and the number of discrete values we've observed.
//
// Luckily, the square of distance from the mean can be expressed using
// these as well:
//
//   Σ (x - μ)² = Σ (x² - 2xμ + μ²) = Σ x² + - Σ 2xμ + Σ μ²
//                                  = sum_squares + -2(sum/n)(sum) + (sum / n)²
//                                  = sum_squares + -2(sum²/n) + n(sum / n)²
//                                  = sum_squares + -2(sum²/n) + n(sum² / n²)
//                                  = sum_squares + -2(sum²/n) + sum²/n
//                                  = sum_squares - sum²/n
//
func squareOfDistanceFromMean(sum, sumSquares, n float64) float64 {
	return sumSquares - math.Pow(sum, 2)/n
}

// Standard deviation can be expressed, simply as:
//
//   √ (Σ (x - μ)² / N)
//
// Since we only have sum, sumSquares, and n in a time series context, we'll
// use a derived formula from those values.
func stddev(sum, sumSquares float64, count int64) float64 {
	return math.Sqrt(squareOfDistanceFromMean(sum, sumSquares, float64(count)) / float64(count))
}
