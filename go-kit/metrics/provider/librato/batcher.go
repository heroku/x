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
	batchMetricsPath      = "/v1/metrics"
	batchMeasurementsPath = "/v1/measurements"
)

func (p *Provider) Batch(u *url.URL, interval time.Duration) ([]*http.Request, error) {
	if p.tagsEnabled {
		return p.batchMeasurements(u, interval)
	}
	return p.batchMetrics(u, interval)
}

// batchMetrics will batch up all the metrics into []*http.Requests using the old style API.
func (p *Provider) batchMetrics(u *url.URL, interval time.Duration) ([]*http.Request, error) {
	// Calculate the sample time.
	st := time.Now().Truncate(interval).Unix()

	// Sample the metrics.
	measurements := p.sample(int(interval.Seconds()))
	if len(measurements) == 0 {
		// no data to report
		return nil, nil
	}

	gauges := make([]gauge, 0, len(measurements))
	for _, m := range measurements {
		gauges = append(gauges, m.Gauge())
	}

	// Don't accidentally leak the creds, which can happen if we return the u with a u.User set
	var user *url.Userinfo
	user, u.User = u.User, nil

	u = u.ResolveReference(&url.URL{Path: batchMetricsPath})

	nextEnd := func(e int) int {
		e += p.batchSize
		if l := len(gauges); e > l {
			return l
		}
		return e
	}

	requests := make([]*http.Request, 0, len(gauges)/p.batchSize+1)
	for batch, e := 0, nextEnd(0); batch < len(gauges); batch, e = e, nextEnd(e) {
		r := struct {
			Source      string                 `json:"source,omitempty"`
			MeasureTime int64                  `json:"measure_time"`
			Gauges      []gauge                `json:"gauges"`
			Attributes  map[string]interface{} `json:"attributes,omitempty"`
		}{
			Source:      p.source,
			MeasureTime: st,
			Gauges:      gauges[batch:e],
		}
		if p.ssa {
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

func (p *Provider) batchMeasurements(u *url.URL, interval time.Duration) ([]*http.Request, error) {
	// Sample the metrics.
	measurements := p.sample(int(interval.Seconds()))

	if len(measurements) == 0 { // no data to report
		return nil, nil
	}

	// Don't accidentally leak the creds, which can happen if we return the u with a u.User set
	var user *url.Userinfo
	user, u.User = u.User, nil

	u = u.ResolveReference(&url.URL{Path: batchMeasurementsPath})

	nextEnd := func(e int) int {
		e += p.batchSize
		if l := len(measurements); e > l {
			return l
		}
		return e
	}

	requests := make([]*http.Request, 0, len(measurements)/p.batchSize+1)
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

type taggedBatcher struct {
	p *Provider
}

type measurement struct {
	Name   string `json:"name"`
	Time   int64  `json:"time"`
	Period int    `json:"period"`

	Attributes map[string]interface{} `json:"attributes,omitempty"`
	Tags       map[string]string      `json:"tags"`

	Count  int64   `json:"count"`
	Sum    float64 `json:"sum"`
	SumSq  float64 `json:"-"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	Last   float64 `json:"last"`
	StdDev float64 `json:"stddev"`
}

func (m *measurement) Gauge() gauge {
	return gauge{
		Name:   m.Name,
		Period: m.Period,
		Count:  m.Count,
		Sum:    m.Sum,
		Min:    m.Min,
		Max:    m.Max,
		SumSq:  m.SumSq,
	}
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
