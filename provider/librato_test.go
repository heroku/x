package provider

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

	"github.com/go-kit/kit/metrics/generic"
)

func TestLibratoReport(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	user := os.Getenv("LIBRATO_TEST_USER")
	pwd := os.Getenv("LIBRATO_TEST_PWD")
	if user == "" || pwd == "" {
		t.Skip("LIBRATO_TEST_USER || LIBRATO_TEST_PWD unset")
	}
	u, err := url.Parse(DefaultLibratoURL)
	if err != nil {
		t.Fatalf("got %q, expected nil", err)
	}
	u.User = url.UserPassword(user, pwd)
	source := "test.source"

	var l Librato
	c := l.NewCounter("test.counter")
	g := l.NewGauge("test.gauge")
	h := l.NewHistogram("test.histogram", DefaultBucketCount)
	c.Add(float64(time.Now().Unix())) // increasing value
	g.Set(rand.Float64())
	h.Observe(10)
	h.Observe(100)
	h.Observe(150)

	// Call the reporter explicitly
	if err := l.report(u, 10*time.Second, source); err != nil {
		t.Fatalf("got %q, expected nil", err)
	}
}

func BenchmarkMarshalGeneric(b *testing.B) {
	var sink json.Marshaler
	c := generic.NewCounter("test.counter")
	c.Add(100000)
	for i := 0; i < b.N; i++ {
		j := marshalGeneric(c.Name, c.Value(), 1)
		sink = j
	}
	_ = sink
}

type g struct {
	Name   string  `json:"name"`
	Value  float64 `json:"value"`
	Period float64 `json:"period"`
}

func generalExpectations(t *testing.T, gJSON []byte, eJSON, eName string, eValue float64) {
	if string(gJSON) != eJSON {
		t.Errorf("got %q\nexpected %q", gJSON, eJSON)
	}

	var tg g
	err := json.Unmarshal(gJSON, &tg)
	if err != nil {
		t.Fatalf("got %q, expected nil", err)
	}

	if tg.Name != eName {
		t.Errorf("got %q, expected %q", tg.Name, eName)
	}
	if tg.Value != eValue {
		t.Errorf("got %f, expected %f", tg.Value, eValue)
	}
}

func TestMarshalGeneric(t *testing.T) {
	c := generic.NewCounter("test.counter")
	c.Add(100)
	d := marshalGeneric("test.counter", 100, 1)
	ej := `{"name":"test.counter","value":100.000000,"period":1.000}`
	en := "test.counter"
	ev := 100.00
	generalExpectations(t, []byte(d), ej, en, ev)
}

type lg struct {
	Name       string  `json:"name"`
	Count      int     `json:"count"`
	Sum        float64 `json:"sum"`
	Min        float64 `json:"min"`
	Max        float64 `json:"max"`
	SumSquares float64 `json:"sum_squares"`
}

func TestLibratoHistogramJSONMarshalers(t *testing.T) {
	h := LibratoHistogram{name: "test.histogram", buckets: DefaultBucketCount}
	h.reset()
	h.Observe(10)
	h.Observe(100)
	h.Observe(150)
	d := h.jsonMarshalers(1)
	if len(d) != 4 {
		t.Fatalf("got %d, expected length to be 4", len(d))
	}
	p1 := []byte(d[0].(json.RawMessage))
	ej := `{"name":"test.histogram","count":3,"sum":260.000000,"min":10.000000,"max":150.000000,"sum_squares":32600.000000}`
	if string(p1) != ej {
		t.Errorf("got %q\nexpected %q", p1, ej)
	}

	// Double check our expectations.
	var tlg lg
	err := json.Unmarshal(p1, &tlg)
	if err != nil {
		t.Fatalf("got %q, expected nil", err)
	}
	en := "test.histogram"
	if tlg.Name != en {
		t.Errorf("name: got %q, expected %q", tlg.Name, en)
	}
	ec := 3
	if tlg.Count != ec {
		t.Errorf("count: got %d, expected %d", tlg.Count, ec)
	}
	es := 260.0
	if math.Float64bits(tlg.Sum) != math.Float64bits(es) {
		t.Errorf("sum: got %f, expected %f", tlg.Sum, es)
	}
	ess := 32600.0
	if math.Float64bits(tlg.SumSquares) != math.Float64bits(ess) {
		t.Errorf("sum_squares: got %f, expected %f", tlg.SumSquares, ess)
	}
	em := 10.0
	if math.Float64bits(tlg.Min) != math.Float64bits(em) {
		t.Errorf("min: got %f, expected %f", tlg.Min, em)
	}
	em = 150.0
	if math.Float64bits(tlg.Max) != math.Float64bits(em) {
		t.Errorf("max: got %f, expected %f", tlg.Max, em)
	}

	p99 := []byte(d[1].(json.RawMessage))
	ep99 := `{"name":"test.histogram.p99","value":150.000000,"period":1.000}`
	ep99n := "test.histogram.p99"
	ep99v := 150.0
	generalExpectations(t, p99, ep99, ep99n, ep99v)

	p95 := []byte(d[2].(json.RawMessage))
	ep95 := `{"name":"test.histogram.p95","value":150.000000,"period":1.000}`
	ep95n := "test.histogram.p95"
	ep95v := 150.0
	generalExpectations(t, p95, ep95, ep95n, ep95v)

	p50 := []byte(d[3].(json.RawMessage))
	ep50 := `{"name":"test.histogram.p50","value":100.000000,"period":1.000}`
	ep50n := "test.histogram.p50"
	ep50v := 100.00
	generalExpectations(t, p50, ep50, ep50n, ep50v)
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
	var l Librato
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
			h := l.NewHistogram(tc.name, DefaultBucketCount)
			for _, v := range tc.values {
				h.Observe(v)
			}
			lh, ok := h.(*LibratoHistogram)
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
