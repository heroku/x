package librato

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/url"
	"os"
	"sort"
	"testing"
	"time"
)

func TestLibratoSingleReport(t *testing.T) {
	user := os.Getenv("LIBRATO_TEST_USER")
	pwd := os.Getenv("LIBRATO_TEST_PWD")
	if user == "" || pwd == "" {
		t.Skip("LIBRATO_TEST_USER || LIBRATO_TEST_PWD unset")
	}
	rand.Seed(time.Now().UnixNano())
	u, err := url.Parse(DefaultLibratoURL)
	if err != nil {
		t.Fatalf("got %q, expected nil", err)
	}
	u.User = url.UserPassword(user, pwd)

	var p Provider
	p.source = "test.source"
	c := p.NewCounter("test.counter")
	g := p.NewGauge("test.gauge")
	h := p.NewHistogram("test.histogram", DefaultBucketCount)
	c.Add(float64(time.Now().Unix())) // increasing value
	g.Set(rand.Float64())
	h.Observe(10)
	h.Observe(100)
	h.Observe(150)

	// Call the reporter explicitly
	if err := p.report(u, 10*time.Second); err != nil {
		t.Fatalf("got %q, expected nil", err)
	}
}

func TestLibratoReport(t *testing.T) {
	user := os.Getenv("LIBRATO_TEST_USER")
	pwd := os.Getenv("LIBRATO_TEST_PWD")
	if user == "" || pwd == "" {
		t.Skip("LIBRATO_TEST_USER || LIBRATO_TEST_PWD unset")
	}
	rand.Seed(time.Now().UnixNano())
	u, err := url.Parse(DefaultLibratoURL)
	if err != nil {
		t.Fatalf("got %q, expected nil", err)
	}
	u.User = url.UserPassword(user, pwd)

	errHandler := func(err error) {
		t.Errorf("got %q but didn't expect any errors", err)
	}

	p := New(u, time.Second, WithSource("test.source"), WithErrorHandler(errHandler))
	c := p.NewCounter("test.counter")
	g := p.NewGauge("test.gauge")
	h := p.NewHistogram("test.histogram", DefaultBucketCount)

	done := make(chan struct{})

	go func() {
		for i := 0; i < 30; i++ {
			c.Add(float64(time.Now().Unix())) // increasing value
			g.Set(rand.Float64())
			h.Observe(rand.Float64() * 100)
			h.Observe(rand.Float64() * 100)
			h.Observe(rand.Float64() * 100)
			time.Sleep(100 * time.Millisecond)
		}
		p.Stop()
		close(done)
	}()

	<-done
}

func gaugeExpectations(t *testing.T, gJSON []byte, eJSON, eName string, eCount int64, ePeriod, eSum, eMin, eMax, eSumSq float64) {
	if string(gJSON) != eJSON {
		t.Errorf("got %q\nexpected %q", gJSON, eJSON)
	}

	var tg gauge
	err := json.Unmarshal(gJSON, &tg)
	if err != nil {
		t.Fatalf("got %q, expected nil", err)
	}

	if tg.Name != eName {
		t.Errorf("got %q, expected %q", tg.Name, eName)
	}
	if tg.Count != eCount {
		t.Errorf("got %d, expected %d", tg.Count, eCount)
	}
	if tg.Period != ePeriod {
		t.Errorf("got %f, expected %f", tg.Period, ePeriod)
	}
	if tg.Sum != eSum {
		t.Errorf("got %f, expected %f", tg.Sum, eSum)
	}
	if tg.Min != eMin {
		t.Errorf("got %f, expected %f", tg.Min, eMin)
	}
	if tg.Max != eMax {
		t.Errorf("got %f, expected %f", tg.Max, eMax)
	}
	if tg.SumSq != eSumSq {
		t.Errorf("got %f, expected %f", tg.SumSq, eSumSq)
	}
}

func counterExpectations(t *testing.T, gJSON []byte, eJSON, eName string, ePeriod, eValue float64) {
	if string(gJSON) != eJSON {
		t.Errorf("got %q\nexpected %q", gJSON, eJSON)
	}

	var tc counter
	err := json.Unmarshal(gJSON, &tc)
	if err != nil {
		t.Fatalf("got %q, expected nil", err)
	}

	if tc.Name != eName {
		t.Errorf("got %q, expected %q", tc.Name, eName)
	}
	if tc.Period != ePeriod {
		t.Errorf("got %f, expected %f", tc.Period, ePeriod)
	}
	if tc.Value != eValue {
		t.Errorf("got %f, expected %f", tc.Value, eValue)
	}
}

func TestLibratoHistogramJSONMarshalers(t *testing.T) {
	h := Histogram{name: "test.histogram", buckets: DefaultBucketCount}
	h.reset()
	h.Observe(10)
	h.Observe(100)
	h.Observe(150)
	ePeriod := 1.0
	d := h.measures(ePeriod)
	if len(d) != 4 {
		t.Fatalf("got %d, expected length to be 4", len(d))
	}
	p1, err := json.Marshal(d[0])
	if err != nil {
		t.Fatalf("got %q, expected nil", err)
	}
	ej := `{"name":"test.histogram","period":1,"count":3,"sum":260,"min":10,"max":150,"sum_squares":32600}`
	if string(p1) != ej {
		t.Errorf("got %q\nexpected %q", p1, ej)
	}

	// Double check our expectations.
	var tg gauge
	err = json.Unmarshal(p1, &tg)
	if err != nil {
		t.Fatalf("got %q, expected nil", err)
	}
	if tg.Period != ePeriod {
		t.Errorf("period: got %f, expected %f", tg.Period, ePeriod)
	}
	en := "test.histogram"
	if tg.Name != en {
		t.Errorf("name: got %q, expected %q", tg.Name, en)
	}
	ec := int64(3)
	if tg.Count != ec {
		t.Errorf("count: got %d, expected %d", tg.Count, ec)
	}
	es := 260.0
	if math.Float64bits(tg.Sum) != math.Float64bits(es) {
		t.Errorf("sum: got %f, expected %f", tg.Sum, es)
	}
	ess := 32600.0
	if math.Float64bits(tg.SumSq) != math.Float64bits(ess) {
		t.Errorf("sum_squares: got %f, expected %f", tg.SumSq, ess)
	}
	em := 10.0
	if math.Float64bits(tg.Min) != math.Float64bits(em) {
		t.Errorf("min: got %f, expected %f", tg.Min, em)
	}
	em = 150.0
	if math.Float64bits(tg.Max) != math.Float64bits(em) {
		t.Errorf("max: got %f, expected %f", tg.Max, em)
	}

	p99, err := json.Marshal(d[1])
	if err != nil {
		t.Fatalf("got %q, expected nil", err)
	}
	ep99 := `{"name":"test.histogram.p99","period":1,"count":1,"sum":150,"min":150,"max":150,"sum_squares":22500}`
	ep99n := "test.histogram.p99"
	ep99c := int64(1)
	ep99min := 150.0
	ep99max := 150.0
	ep99sum := 150.0
	ep99sumsq := ep99sum * ep99sum
	gaugeExpectations(t, p99, ep99, ep99n, ep99c, ePeriod, ep99sum, ep99min, ep99max, ep99sumsq)

	p95, err := json.Marshal(d[2])
	if err != nil {
		t.Fatalf("got %q, expected nil", err)
	}
	ep95 := `{"name":"test.histogram.p95","period":1,"count":1,"sum":150,"min":150,"max":150,"sum_squares":22500}`
	ep95n := "test.histogram.p95"
	ep95c := int64(1)
	ep95min := 150.0
	ep95max := 150.0
	ep95sum := 150.0
	ep95sumsq := ep95sum * ep95sum
	gaugeExpectations(t, p95, ep95, ep95n, ep95c, ePeriod, ep95sum, ep95min, ep95max, ep95sumsq)

	p50, err := json.Marshal(d[3])
	if err != nil {
		t.Fatalf("got %q, expected nil", err)
	}
	ep50 := `{"name":"test.histogram.p50","period":1,"count":1,"sum":100,"min":100,"max":100,"sum_squares":10000}`
	ep50n := "test.histogram.p50"
	ep50c := int64(1)
	ep50min := 100.00
	ep50max := 100.0
	ep50sum := 100.00
	ep50sumsq := ep50sum * ep50sum
	gaugeExpectations(t, p50, ep50, ep50n, ep50c, ePeriod, ep50sum, ep50min, ep50max, ep50sumsq)
}

type testcase struct {
	name        string
	values      []float64
	eCount      int64
	eSum        float64
	eSumSquares float64
	eMin        float64
	eMax        float64
}

func generateHistogramTestData(c, max int) testcase {
	rand.Seed(time.Now().UnixNano())
	var tc testcase
	tc.name = fmt.Sprintf("test %d %d", c, max)
	tc.eCount = int64(c)
	tc.eMin = 0
	values := make([]float64, 0, c)
	for i := 0; i < c; i++ {
		v := float64(rand.Intn(max))
		values = append(values, v)
		tc.values = append(tc.values, v)
		tc.eSum += v
		tc.eSumSquares += v * v
		if v < tc.eMin || tc.eMin == 0 {
			tc.eMin = v
		}
		if v > tc.eMax {
			tc.eMax = v
		}
	}
	sort.Float64s(values)
	return tc
}

func TestHistogram(t *testing.T) {
	var p Provider
	for _, tc := range []testcase{
		generateHistogramTestData(10, 5*int(time.Second/time.Microsecond)),
		generateHistogramTestData(100, 5*int(time.Second/time.Microsecond)),
		generateHistogramTestData(1000, 5*int(time.Second/time.Microsecond)),
		generateHistogramTestData(10000, 5*int(time.Second/time.Microsecond)),
		generateHistogramTestData(100000, 5*int(time.Second/time.Microsecond)),
		generateHistogramTestData(10, 10*int(time.Second/time.Microsecond)),
		generateHistogramTestData(100, 10*int(time.Second/time.Microsecond)),
		generateHistogramTestData(1000, 10*int(time.Second/time.Microsecond)),
		generateHistogramTestData(10000, 10*int(time.Second/time.Microsecond)),
		generateHistogramTestData(100000, 10*int(time.Second/time.Microsecond)),
		generateHistogramTestData(10, int(time.Hour/time.Microsecond)),
		generateHistogramTestData(100, int(time.Hour/time.Microsecond)),
		generateHistogramTestData(1000, int(time.Hour/time.Microsecond)),
		generateHistogramTestData(10000, int(time.Hour/time.Microsecond)),
		generateHistogramTestData(100000, int(time.Hour/time.Microsecond)),
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := p.NewHistogram(tc.name, DefaultBucketCount)
			for _, v := range tc.values {
				h.Observe(v)
			}
			lh, ok := h.(*Histogram)
			if !ok {
				t.Fatal("Could not convert to *Histogram")
			}
			if c := lh.Count(); c != tc.eCount {
				t.Errorf("go Count() (%d), but expected (%d)", c, tc.eCount)
			}
			if s := lh.Sum(); s != tc.eSum {
				t.Errorf("got Sum() (%f), but expected (%f)", s, tc.eSum)
			}
			if ss := lh.SumSq(); ss != tc.eSumSquares {
				t.Errorf("got SumSq() squares (%f), but expected (%f)", ss, tc.eSum)
			}
			if m := lh.Min(); m != tc.eMin {
				t.Errorf("got Min() (%f), but expected (%f)", m, tc.eMin)
			}
			if m := lh.Max(); m != tc.eMax {
				t.Errorf("got Max() (%f), but expecte (%f)", m, tc.eMax)
			}
		})
	}
}
