/* Copyright (c) 2018 Salesforce
 * All rights reserved.
 * Licensed under the BSD 3-Clause license.
 * For full license text, see LICENSE.txt file in the repo root  or https://opensource.org/licenses/BSD-3-Clause
 */

package librato

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	kmetrics "github.com/go-kit/kit/metrics"
)

var (
	doesntmatter = time.Hour
)

func ExampleNew() {
	start := time.Now()
	u, err := url.Parse(DefaultURL)
	if err != nil {
		log.Fatal(err)
	}
	u.User = url.UserPassword("libratoUser", "libratoPassword/Token")

	errHandler := func(err error) {
		log.Println(err)
	}
	p := New(u, 20*time.Second, WithErrorHandler(errHandler))
	c := p.NewCounter("i.am.a.counter")
	h := p.NewHistogram("i.am.a.histogram", DefaultBucketCount)
	g := p.NewGauge("i.am.a.gauge")
	uc := p.NewCardinalityCounter("i.am.a.cardinality.estimate.counter")

	// Pretend applicaion logic....
	c.Add(1)
	h.Observe(time.Since(start).Seconds()) // how long did it take the program to get here.
	g.Set(1000)
	uc.Insert([]byte("count this as 1"))
	// /Pretend

	// block until we report one final time
	p.Stop()
}

func TestLibratoReportRequestDebugging(t *testing.T) {
	for _, debug := range []bool{true, false} {
		t.Run(fmt.Sprintf("%t", debug), func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			}))
			defer srv.Close()
			u, err := url.Parse(srv.URL)
			if err != nil {
				t.Fatal(err)
			}
			p := New(u, doesntmatter, func(p *Provider) { p.requestDebugging = debug }).(*Provider)
			p.Stop()
			p.NewCounter("foo").Add(1) // need at least one metric in order to report
			reqs, err := p.Batch(u, doesntmatter)
			if err != nil {
				t.Fatal("unexpected error", err)
			}
			if len(reqs) != 1 {
				t.Errorf("expected 1 request, got %d", len(reqs))
			}
			err = p.report(reqs[0])
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			e, ok := err.(Error)
			if !ok {
				t.Fatalf("expected an Error, got %T: %q", err, err.Error())
			}

			req := e.Request()
			if debug {
				if req == "" {
					t.Error("unexpected empty request")
				}
			} else {
				if req != "" {
					t.Errorf("expected no request, got %#v", req)
				}
			}

		})
	}
}

type temporary interface {
	Temporary() bool
}

func TestLibratoRetriesWithErrors(t *testing.T) {
	var retried int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		retried++
		b, err := io.Copy(ioutil.Discard, r.Body)
		if err != nil {
			t.Fatal("Unable to read all of the request body:", err)
		}
		if b == 0 {
			t.Fatal("expected to copy more than 0 bytes")
		}
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	var totalErrors, temporaryErrors, finalErrors int
	expectedRetries := 3
	errHandler := func(err error) {
		totalErrors++
		if terr, ok := err.(temporary); ok {
			if terr.Temporary() {
				temporaryErrors++
			} else {
				finalErrors++
			}
			t.Log(err)
		}
	}
	p := New(u, doesntmatter, WithErrorHandler(errHandler), WithRetries(expectedRetries), WithRequestDebugging()).(*Provider)
	p.Stop()
	p.NewCounter("foo").Add(1) // need at least one metric in order to report
	p.reportWithRetry(u, doesntmatter)

	if totalErrors != expectedRetries*2 {
		t.Errorf("expected %d total errors, got %d", expectedRetries*2, totalErrors)
	}

	expectedTemporaryErrors := expectedRetries - 1
	if temporaryErrors != expectedTemporaryErrors*2 {
		t.Errorf("expected %d temporary errors, got %d", expectedTemporaryErrors*2, temporaryErrors)
	}

	expectedFinalErrors := 1
	if finalErrors != expectedFinalErrors*2 {
		t.Errorf("expected %d final errors, got %d", expectedFinalErrors*2, finalErrors)
	}

	if retried != expectedRetries*2 {
		t.Errorf("expected %d retries, got %d", expectedRetries*2, retried)
	}
}

func TestLibratoRetriesWithErrorsNoDebugging(t *testing.T) {
	var retried int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		retried++
		b, err := io.Copy(ioutil.Discard, r.Body)
		if err != nil {
			t.Fatal("Unable to read all of the request body:", err)
		}
		if b == 0 {
			t.Fatal("expected more than 0 bytes in the body")
		}
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	var totalErrors, temporaryErrors, finalErrors int
	expectedRetries := 3
	errHandler := func(err error) {
		totalErrors++
		if terr, ok := err.(temporary); ok {
			if terr.Temporary() {
				temporaryErrors++
			} else {
				finalErrors++
			}
			t.Log(err)
		}
	}
	p := New(u, doesntmatter, WithErrorHandler(errHandler), WithRetries(expectedRetries)).(*Provider)
	p.Stop()
	p.NewCounter("foo").Add(1) // need at least one metric in order to report
	p.reportWithRetry(u, doesntmatter)

	if totalErrors != expectedRetries*2 {
		t.Errorf("expected %d total errors, got %d", expectedRetries*2, totalErrors)
	}

	expectedTemporaryErrors := expectedRetries - 1
	if temporaryErrors != expectedTemporaryErrors*2 {
		t.Errorf("expected %d temporary errors, got %d", expectedTemporaryErrors*2, temporaryErrors)
	}

	expectedFinalErrors := 1
	if finalErrors != expectedFinalErrors*2 {
		t.Errorf("expected %d final errors, got %d", expectedFinalErrors*2, finalErrors)
	}

	if retried != expectedRetries*2 {
		t.Errorf("expected %d retries, got %d", expectedRetries*2, retried)
	}
}

func TestLibratoBatchingReport(t *testing.T) {
	user := os.Getenv("LIBRATO_TEST_USER")
	pwd := os.Getenv("LIBRATO_TEST_PWD")
	if user == "" || pwd == "" {
		t.Skip("LIBRATO_TEST_USER || LIBRATO_TEST_PWD unset")
	}
	rand.Seed(time.Now().UnixNano())
	u, err := url.Parse(DefaultURL)
	if err != nil {
		t.Fatalf("expected nil, got %q", err)
	}
	u.User = url.UserPassword(user, pwd)

	errs := func(err error) {
		t.Error("unexpected error reporting metrics", err)
	}

	p := New(u, time.Second, WithSource("test.source"), WithErrorHandler(errs))
	h := make([]kmetrics.Histogram, 0, DefaultBatchSize)
	for i := 0; i < DefaultBatchSize; i++ { // each histogram creates multiple gauges
		h = append(h, p.NewHistogram(fmt.Sprintf("test.histogram.%d", i), DefaultBucketCount))
	}

	done := make(chan struct{})
	go func() {
		for i := 0; i < 30; i++ {
			for i := range h {
				h[i].Observe(rand.Float64() * 100)
				h[i].Observe(rand.Float64() * 200)
				h[i].Observe(rand.Float64() * 300)
			}
			time.Sleep(100 * time.Millisecond)
		}
		p.Stop()
		close(done)
	}()

	<-done
	p.Stop() // do a final report
}

func TestLibratoSingleReport(t *testing.T) {
	user := os.Getenv("LIBRATO_TEST_USER")
	pwd := os.Getenv("LIBRATO_TEST_PWD")
	if user == "" || pwd == "" {
		t.Skip("LIBRATO_TEST_USER || LIBRATO_TEST_PWD unset")
	}
	rand.Seed(time.Now().UnixNano())
	u, err := url.Parse(DefaultURL)
	if err != nil {
		t.Fatalf("expected nil, got %q", err)
	}
	u.User = url.UserPassword(user, pwd)

	errs := func(err error) {
		t.Fatal("unexpected error reporting metrics", err)
	}

	p := New(u, doesntmatter, WithSource("test.source"), WithErrorHandler(errs))
	c := p.NewCounter("test.counter")
	g := p.NewGauge("test.gauge")
	h := p.NewHistogram("test.histogram", DefaultBucketCount)
	uc := p.NewCardinalityCounter("test.uniquecounter")
	c.Add(float64(time.Now().Unix())) // increasing value
	g.Set(rand.Float64())
	h.Observe(10)
	h.Observe(100)
	h.Observe(150)
	uc.Insert([]byte("foo.bar"))
	p.Stop() // does a final report
}

func TestLibratoSingleReportWithLabelValuesOnSourceBasedAccount(t *testing.T) {
	user := os.Getenv("LIBRATO_TEST_USER")
	pwd := os.Getenv("LIBRATO_TEST_PWD")
	if user == "" || pwd == "" {
		t.Skip("LIBRATO_TEST_USER || LIBRATO_TEST_PWD unset")
	}
	rand.Seed(time.Now().UnixNano())
	u, err := url.Parse(DefaultURL)
	if err != nil {
		t.Fatalf("expected nil, got %q", err)
	}
	u.User = url.UserPassword(user, pwd)

	errs := func(err error) {
		t.Fatal("unexpected error reporting metrics", err)
	}

	p := New(u, doesntmatter, WithSource("test.source"), WithErrorHandler(errs))
	c := p.NewCounter("test.counter")
	g := p.NewGauge("test.gauge")
	h := p.NewHistogram("test.histogram", DefaultBucketCount)
	c.With("region", "us").With("space", "myspace").Add(float64(time.Now().Unix())) // increasing value
	g.With("region", "us").With("space", "myspace").Set(rand.Float64())
	h.With("region", "us").With("space", "myspace").Observe(10)
	h.With("region", "us").With("space", "myspace").Observe(100)
	h.With("region", "us").With("space", "myspace").Observe(150)
	p.Stop() // does a final report
}

func TestLibratoSingleReportWithLabelValuesOnTagBasedAccount(t *testing.T) {
	user := os.Getenv("LIBRATO_TEST_USER")
	pwd := os.Getenv("LIBRATO_TEST_PWD")
	if user == "" || pwd == "" {
		t.Skip("LIBRATO_TEST_USER || LIBRATO_TEST_PWD unset")
	}
	rand.Seed(time.Now().UnixNano())
	u, err := url.Parse(DefaultURL)
	if err != nil {
		t.Fatalf("expected nil, got %q", err)
	}
	u.User = url.UserPassword(user, pwd)

	errs := func(err error) {
		t.Fatal("unexpected error reporting metrics", err)
	}

	p := New(u, doesntmatter, WithTags("app", "myapp"), WithSource("test.source"), WithErrorHandler(errs))
	c := p.NewCounter("test.counter")
	g := p.NewGauge("test.gauge")
	h := p.NewHistogram("test.histogram", DefaultBucketCount)
	cc := p.NewCardinalityCounter("test.cardinality-counter")
	c.With("region", "us").With("space", "myspace").Add(float64(time.Now().Unix())) // increasing value
	g.With("region", "us").With("space", "myspace").Set(rand.Float64())
	h.With("region", "us").With("space", "myspace").Observe(10)
	h.With("region", "us").With("space", "myspace").Observe(100)
	h.With("region", "us").With("space", "myspace").Observe(150)
	cc.With("region", "us").With("space", "myspace").Insert([]byte("foo.bar"))
	p.Stop() // does a final report
}

func TestLibratoReport(t *testing.T) {
	user := os.Getenv("LIBRATO_TEST_USER")
	pwd := os.Getenv("LIBRATO_TEST_PWD")
	if user == "" || pwd == "" {
		t.Skip("LIBRATO_TEST_USER || LIBRATO_TEST_PWD unset")
	}
	rand.Seed(time.Now().UnixNano())
	u, err := url.Parse(DefaultURL)
	if err != nil {
		t.Fatalf("expected nil, got %q", err)
	}
	u.User = url.UserPassword(user, pwd)

	errs := func(err error) {
		t.Error("unexpected error reporting metrics", err)
	}

	p := New(u, time.Second, WithSource("test.source"), WithErrorHandler(errs))
	c := p.NewCounter("test.counter")
	g := p.NewGauge("test.gauge")
	h := p.NewHistogram("test.histogram", DefaultBucketCount)
	uc := p.NewCardinalityCounter("test.uniquecounter")

	done := make(chan struct{})

	go func() {
		for i := 0; i < 30; i++ {
			c.Add(float64(time.Now().Unix())) // increasing value
			g.Set(rand.Float64())
			h.Observe(rand.Float64() * 100)
			h.Observe(rand.Float64() * 100)
			h.Observe(rand.Float64() * 100)
			uc.Insert([]byte("something"))
			time.Sleep(100 * time.Millisecond)
		}
		p.Stop()
		close(done)
	}()

	<-done
	p.Stop() // does a final report
}

func TestLibratoHistogramJSONMarshalers(t *testing.T) {
	p := &Provider{
		histograms: make(map[string]*Histogram),
		now:        func() time.Time { return time.Unix(1529076673, 0).UTC() },
	}
	h := &Histogram{p: p, name: "test.histogram", buckets: DefaultBucketCount, percentilePrefix: ".p"}
	h.reset()
	h2 := h.With("region", "us").(*Histogram)
	h2.Observe(10)
	h2.Observe(100)
	h2.Observe(150)

	pTagsEnabled := &Provider{
		tagsEnabled: true,
		histograms:  make(map[string]*Histogram),
		now:         func() time.Time { return time.Unix(1529076673, 0).UTC() },
	}
	hTags := &Histogram{p: pTagsEnabled, name: "test.histogram", buckets: DefaultBucketCount, percentilePrefix: ".p"}
	hTags.reset()
	h2Tags := hTags.With("region", "us").(*Histogram)
	h2Tags.Observe(10)
	h2Tags.Observe(100)
	h2Tags.Observe(150)

	ePeriod := 60

	d := p.histogramMeasures(h2, ePeriod)
	if len(d) != 4 {
		t.Fatalf("expected length of parts to be 4, got %d", len(d))
	}

	p1, err := json.Marshal(d[0])
	if err != nil {
		t.Fatal("unexpected error unmarshaling", err)
	}
	p99, err := json.Marshal(d[1])
	if err != nil {
		t.Fatal("unexpected error unmarshaling", err)
	}
	p95, err := json.Marshal(d[2])
	if err != nil {
		t.Fatal("unexpected error unmarshaling", err)
	}
	p50, err := json.Marshal(d[3])
	if err != nil {
		t.Fatal("unexpected error unmarshaling", err)
	}

	d2 := p.histogramMeasures(h2Tags, ePeriod)
	if len(d2) != 4 {
		t.Fatalf("expected length of parts to be 4, got %d", len(d2))
	}

	p1Tags, err := json.Marshal(d2[0])
	if err != nil {
		t.Fatal("unexpected error unmarshaling", err)
	}
	p99Tags, err := json.Marshal(d2[1])
	if err != nil {
		t.Fatal("unexpected error unmarshaling", err)
	}
	p95Tags, err := json.Marshal(d2[2])
	if err != nil {
		t.Fatal("unexpected error unmarshaling", err)
	}
	p50Tags, err := json.Marshal(d2[3])
	if err != nil {
		t.Fatal("unexpected error unmarshaling", err)
	}

	cases := []struct {
		eRaw, eName               string
		eCount                    int64
		eMin, eMax, eSum, eStdDev float64
		input                     []byte
	}{
		{
			eRaw:   `{"name":"test.histogram.region:us","time":1529076660,"period":60,"tags":{"region":"us"},"count":3,"sum":260,"min":10,"max":150,"last":150,"stddev":57.92715732327589}`,
			eName:  "test.histogram.region:us",
			eCount: 3, eMin: 10, eMax: 150, eSum: 260, eStdDev: 57.92715732327589,
			input: p1,
		},

		{
			eRaw:   `{"name":"test.histogram.region:us.p99","time":1529076660,"period":60,"tags":{"region":"us"},"count":1,"sum":149,"min":149,"max":149,"last":149,"stddev":0}`,
			eName:  "test.histogram.region:us.p99",
			eCount: 1, eMin: 149, eMax: 149, eSum: 149, eStdDev: 0,
			input: p99,
		},
		{
			eRaw:   `{"name":"test.histogram.region:us.p95","time":1529076660,"period":60,"tags":{"region":"us"},"count":1,"sum":145,"min":145,"max":145,"last":145,"stddev":0}`,
			eName:  "test.histogram.region:us.p95",
			eCount: 1, eMin: 145, eMax: 145, eSum: 145, eStdDev: 0,
			input: p95,
		},
		{
			eRaw:   `{"name":"test.histogram.region:us.p50","time":1529076660,"period":60,"tags":{"region":"us"},"count":1,"sum":100,"min":100,"max":100,"last":100,"stddev":0}`,
			eName:  "test.histogram.region:us.p50",
			eCount: 1, eMin: 100, eMax: 100, eSum: 100, eStdDev: 0,
			input: p50,
		},

		{
			eRaw:   `{"name":"test.histogram","time":1529076660,"period":60,"tags":{"region":"us"},"count":3,"sum":260,"min":10,"max":150,"last":150,"stddev":57.92715732327589}`,
			eName:  "test.histogram",
			eCount: 3, eMin: 10, eMax: 150, eSum: 260, eStdDev: 57.92715732327589,
			input: p1Tags,
		},
		{
			eRaw:   `{"name":"test.histogram.p99","time":1529076660,"period":60,"tags":{"region":"us"},"count":1,"sum":149,"min":149,"max":149,"last":149,"stddev":0}`,
			eName:  "test.histogram.p99",
			eCount: 1, eMin: 149, eMax: 149, eSum: 149, eStdDev: 0,
			input: p99Tags,
		},
		{
			eRaw:   `{"name":"test.histogram.p95","time":1529076660,"period":60,"tags":{"region":"us"},"count":1,"sum":145,"min":145,"max":145,"last":145,"stddev":0}`,
			eName:  "test.histogram.p95",
			eCount: 1, eMin: 145, eMax: 145, eSum: 145, eStdDev: 0,
			input: p95Tags,
		},
		{
			eRaw:   `{"name":"test.histogram.p50","time":1529076660,"period":60,"tags":{"region":"us"},"count":1,"sum":100,"min":100,"max":100,"last":100,"stddev":0}`,
			eName:  "test.histogram.p50",
			eCount: 1, eMin: 100, eMax: 100, eSum: 100, eStdDev: 0,
			input: p50Tags,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.eName, func(t *testing.T) {
			t.Parallel()
			if string(tc.input) != tc.eRaw {
				t.Errorf("expected\n\t\t%q\ngot\n\t\t%q", tc.eRaw, tc.input)
			}

			var tg measurement
			err := json.Unmarshal(tc.input, &tg)
			if err != nil {
				t.Fatal("unexpected error unmarshalling", err)
			}

			if tg.Name != tc.eName {
				t.Errorf("expected %q, got %q", tc.eName, tg.Name)
			}
			if tg.Count != tc.eCount {
				t.Errorf("expected %d, got %d", tc.eCount, tg.Count)
			}
			if tg.Period != ePeriod {
				t.Errorf("expected %d, got %d", ePeriod, tg.Period)
			}
			if math.Float64bits(tg.Sum) != math.Float64bits(tc.eSum) {
				t.Errorf("expected %f, got %f", tc.eSum, tg.Sum)
			}
			if math.Float64bits(tg.Min) != math.Float64bits(tc.eMin) {
				t.Errorf("expected %f, got %f", tc.eMin, tg.Min)
			}
			if math.Float64bits(tg.Max) != math.Float64bits(tc.eMax) {
				t.Errorf("expected %f, got %f", tc.eMin, tg.Max)
			}
			if math.Float64bits(tg.StdDev) != math.Float64bits(tc.eStdDev) {
				t.Errorf("expected %f, got %f", tc.eStdDev, tg.StdDev)
			}
		})
	}
}

func TestScrubbing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.Copy(ioutil.Discard, r.Body)
		if err != nil {
			t.Fatal("Unable to read all of the request body:", err)
		}
		if b == 0 {
			t.Fatal("expected more than 0 bytes in the body")
		}

		w.WriteHeader(http.StatusBadRequest)
	}))
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	errors := make([]error, 0, 100)
	var errCnt int
	errHandler := func(err error) {
		errors = append(errors, err)
		errCnt++
	}
	u.User = url.UserPassword("foo", "bar") // put user info into the URL
	p := New(u, doesntmatter, WithErrorHandler(errHandler), WithRequestDebugging()).(*Provider)
	p.Stop()

	foo := p.NewCounter("foo")
	foo.Add(1)
	p.reportWithRetry(u, doesntmatter)

	for _, err := range errors {
		e, ok := err.(Error)
		if !ok {
			t.Fatalf("expected Error, got %T: %q", err, err.Error())
		}
		request := e.Request()
		if !strings.Contains(request, "Authorization: Basic [SCRUBBED]") {
			t.Errorf("expected Authorization header to be scrubbed, got %q", request)
		}
	}

	// Close the server now so we get an error from the http client
	srv.Close()
	errors = errors[errCnt:]
	p.reportWithRetry(u, doesntmatter)

	for _, err := range errors {
		_, ok := err.(Error)
		if ok {
			t.Errorf("unexpected Error, got %T: %q", err, err.Error())
		}
		if es := err.Error(); strings.Contains(es, "foo") {
			t.Error("expected the error to not contain sensitive data, got", es)
		}
	}

	if errCnt != 3*DefaultNumRetries {
		t.Errorf("expected total error count to be %d, got %d", 3*DefaultNumRetries, errCnt)
	}
}

func TestWithResetCounters(t *testing.T) {
	for _, reset := range []bool{true, false} {
		t.Run(fmt.Sprintf("%t", reset), func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()
			u, err := url.Parse(srv.URL)
			if err != nil {
				t.Fatal(err)
			}
			p := New(u, doesntmatter, func(p *Provider) { p.resetCounters = reset }).(*Provider)
			p.Stop()

			foo := p.NewCounter("foo")
			foo.Add(1)
			reqs, err := p.Batch(u, doesntmatter)
			if err != nil {
				t.Fatal("unexpected error batching", err)
			}
			if len(reqs) != 1 {
				t.Errorf("expected 1 request, got %d", len(reqs))
			}
			p.report(reqs[0])

			var expected float64
			if reset {
				expected = 0
			} else {
				expected = 1
			}
			type valuer interface {
				Value() float64
			}
			if v := foo.(valuer).Value(); v != expected {
				t.Errorf("expected %f, got %f", expected, v)
			}
		})
	}
}

func TestWithResetCountersCardinalityCounters(t *testing.T) {
	for _, reset := range []bool{true, false} {
		t.Run(fmt.Sprintf("%t", reset), func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()
			u, err := url.Parse(srv.URL)
			if err != nil {
				t.Fatal(err)
			}
			p := New(u, doesntmatter, func(p *Provider) { p.resetCounters = reset }).(*Provider)
			p.Stop()

			foo := p.NewCardinalityCounter("foo")
			foo.Insert([]byte("foo"))

			reqs, err := p.Batch(u, doesntmatter)
			if err != nil {
				t.Fatal("unexpected error batching", err)
			}
			if len(reqs) != 1 {
				t.Errorf("expected 1 request, got %d", len(reqs))
			}
			p.report(reqs[0])

			var expected float64
			if reset {
				expected = 0
			} else {
				expected = 1
			}
			type estimater interface {
				Estimate() uint64
			}
			if v := float64(foo.(estimater).Estimate()); v != expected {
				t.Errorf("expected %f, got %f", expected, v)
			}
		})
	}
}

func TestProviderMetricName(t *testing.T) {
	tests := []struct {
		p           Provider
		scenario    string
		name        string
		labelValues []string
		want        string
	}{
		{
			scenario: "tags disabled, no label values",
			name:     "http_requests",
			want:     "http_requests",
		},

		{
			scenario:    "tags disabled, with label values",
			name:        "http_requests",
			labelValues: []string{"region", "us", "app", "myapp"},
			want:        "http_requests.region:us.app:myapp",
		},

		{
			scenario: "tags enabled, no label values",
			p:        Provider{tagsEnabled: true},
			name:     "http_requests",
			want:     "http_requests",
		},

		{
			scenario:    "tags enabled, with label values",
			p:           Provider{tagsEnabled: true},
			name:        "http_requests",
			labelValues: []string{"region", "us", "app", "myapp"},
			want:        "http_requests",
		},
	}

	for _, test := range tests {
		t.Run(test.scenario, func(t *testing.T) {
			if got := test.p.metricName(test.name, test.labelValues...); test.want != got {
				t.Errorf("want: %q, got %q", test.want, got)
			}
		})
	}
}

func TestHistogramLabelValues(t *testing.T) {
	p := &Provider{histograms: make(map[string]*Histogram)}
	h := p.NewHistogram("test.histogram", DefaultBucketCount).(*Histogram)

	if len(h.labelValues) != 0 {
		t.Fatalf("want no label values, got %#v", h.labelValues)
	}

	h2 := h.With("region", "us").(*Histogram)

	want := []string{"region", "us"}
	if !reflect.DeepEqual(want, h2.labelValues) {
		t.Fatalf("want label values: %#v, got %#v", want, h2.labelValues)
	}

	if h2.metricName() != "test.histogram.region:us" {
		t.Fatalf("want name %q, got %q", "test.histogram.region:us", h2.metricName())
	}

	h3 := h2.With("space", "myspace").(*Histogram)

	want = []string{"region", "us", "space", "myspace"}
	if !reflect.DeepEqual(want, h3.labelValues) {
		t.Fatalf("want label values: %#v, got %#v", want, h3.labelValues)
	}

	if h3.metricName() != "test.histogram.region:us.space:myspace" {
		t.Fatalf("want name %q, got %q", "test.histogram.region:us.space:myspace", h3.metricName())
	}
}

func TestCounterLabelValues(t *testing.T) {
	p := &Provider{counters: make(map[string]*Counter)}
	c := p.NewCounter("test.counter").(*Counter)

	if len(c.LabelValues()) != 0 {
		t.Fatalf("want no label values, got %#v", c.LabelValues())
	}

	c2 := c.With("region", "us").(*Counter)

	want := []string{"region", "us"}
	if !reflect.DeepEqual(want, c2.LabelValues()) {
		t.Fatalf("want label values: %#v, got %#v", want, c2.LabelValues())
	}

	if c2.metricName() != "test.counter.region:us" {
		t.Fatalf("want name %q, got %q", "test.counter.region:us", c2.metricName())
	}

	c3 := c2.With("space", "myspace").(*Counter)

	want = []string{"region", "us", "space", "myspace"}
	if !reflect.DeepEqual(want, c3.LabelValues()) {
		t.Fatalf("want label values: %#v, got %#v", want, c3.LabelValues())
	}

	if c3.metricName() != "test.counter.region:us.space:myspace" {
		t.Fatalf("want name %q, got %q", "test.counter.region:us.space:myspace", c3.metricName())
	}
}

func TestGaugeLabelValues(t *testing.T) {
	p := &Provider{gauges: make(map[string]*Gauge)}
	g := p.NewGauge("test.gauge").(*Gauge)

	if len(g.LabelValues()) != 0 {
		t.Fatalf("want no label values, got %#v", g.LabelValues())
	}

	g2 := g.With("region", "us").(*Gauge)

	want := []string{"region", "us"}
	if !reflect.DeepEqual(want, g2.LabelValues()) {
		t.Fatalf("want label values: %#v, got %#v", want, g2.LabelValues())
	}

	if g2.metricName() != "test.gauge.region:us" {
		t.Fatalf("want name %q, got %q", "test.gauge.region:us", g2.metricName())
	}

	g3 := g2.With("space", "myspace").(*Gauge)

	want = []string{"region", "us", "space", "myspace"}
	if !reflect.DeepEqual(want, g3.LabelValues()) {
		t.Fatalf("want label values: %#v, got %#v", want, g3.LabelValues())
	}

	if g3.metricName() != "test.gauge.region:us.space:myspace" {
		t.Fatalf("want name %q, got %q", "test.gauge.region:us.space:myspace", g3.metricName())
	}
}

func TestCounterNaming(t *testing.T) {
	tests := []struct {
		name            string
		p               *Provider
		fn              func(p *Provider) kmetrics.Counter
		wantName        string
		wantLabelValues []string
	}{
		{
			name:     "no prefix, no label values",
			p:        &Provider{counters: make(map[string]*Counter)},
			fn:       func(p *Provider) kmetrics.Counter { return p.NewCounter("my-counter") },
			wantName: "my-counter",
		},

		{
			name:     "with prefix, no label values",
			p:        &Provider{prefix: "my-prefix", counters: make(map[string]*Counter)},
			fn:       func(p *Provider) kmetrics.Counter { return p.NewCounter("my-counter") },
			wantName: "my-prefix.my-counter",
		},

		{
			name:            "no prefix, with label values",
			p:               &Provider{counters: make(map[string]*Counter)},
			fn:              func(p *Provider) kmetrics.Counter { return p.NewCounter("my-counter").With("region", "us") },
			wantName:        "my-counter",
			wantLabelValues: []string{"region", "us"},
		},

		{
			name:            "no prefix, with label values, tags enabled",
			p:               &Provider{counters: make(map[string]*Counter), tagsEnabled: true},
			fn:              func(p *Provider) kmetrics.Counter { return p.NewCounter("my-counter").With("region", "us") },
			wantName:        "my-counter",
			wantLabelValues: []string{"region", "us"},
		},

		{
			name:            "with prefix, with label values, tags enabled",
			p:               &Provider{prefix: "my-prefix", counters: make(map[string]*Counter), tagsEnabled: true},
			fn:              func(p *Provider) kmetrics.Counter { return p.NewCounter("my-counter").With("region", "us") },
			wantName:        "my-prefix.my-counter",
			wantLabelValues: []string{"region", "us"},
		},

		{
			name:            "no prefix, with label values, tags enabled, default tags",
			p:               &Provider{counters: make(map[string]*Counter), tagsEnabled: true, defaultTags: []string{"sys", "foo"}},
			fn:              func(p *Provider) kmetrics.Counter { return p.NewCounter("my-counter").With("region", "us") },
			wantName:        "my-counter",
			wantLabelValues: []string{"sys", "foo", "region", "us"},
		},

		{
			name:            "with prefix, with label values, tags enabled, default tags",
			p:               &Provider{prefix: "my-prefix", counters: make(map[string]*Counter), tagsEnabled: true, defaultTags: []string{"sys", "foo"}},
			fn:              func(p *Provider) kmetrics.Counter { return p.NewCounter("my-counter").With("region", "us") },
			wantName:        "my-prefix.my-counter",
			wantLabelValues: []string{"sys", "foo", "region", "us"},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			counter := test.fn(test.p)
			if got := counter.(*Counter).Counter.Name; test.wantName != got {
				t.Fatalf("want name: %q, got %q", test.wantName, got)
			}

			if got := counter.(*Counter).LabelValues(); !reflect.DeepEqual(test.wantLabelValues, got) {
				t.Fatalf("want label values: %q, got %q", test.wantLabelValues, got)
			}
		})
	}
}

func TestGaugeNaming(t *testing.T) {
	tests := []struct {
		name            string
		p               *Provider
		fn              func(p *Provider) kmetrics.Gauge
		wantName        string
		wantLabelValues []string
	}{
		{
			name:     "no prefix, no label values",
			p:        &Provider{gauges: make(map[string]*Gauge)},
			fn:       func(p *Provider) kmetrics.Gauge { return p.NewGauge("my-gauge") },
			wantName: "my-gauge",
		},

		{
			name:     "with prefix, no label values",
			p:        &Provider{prefix: "my-prefix", gauges: make(map[string]*Gauge)},
			fn:       func(p *Provider) kmetrics.Gauge { return p.NewGauge("my-gauge") },
			wantName: "my-prefix.my-gauge",
		},

		{
			name:            "no prefix, with label values",
			p:               &Provider{gauges: make(map[string]*Gauge)},
			fn:              func(p *Provider) kmetrics.Gauge { return p.NewGauge("my-gauge").With("region", "us") },
			wantName:        "my-gauge",
			wantLabelValues: []string{"region", "us"},
		},

		{
			name:            "no prefix, with label values, tags enabled",
			p:               &Provider{gauges: make(map[string]*Gauge), tagsEnabled: true},
			fn:              func(p *Provider) kmetrics.Gauge { return p.NewGauge("my-gauge").With("region", "us") },
			wantName:        "my-gauge",
			wantLabelValues: []string{"region", "us"},
		},

		{
			name:            "with prefix, with label values, tags enabled",
			p:               &Provider{prefix: "my-prefix", gauges: make(map[string]*Gauge), tagsEnabled: true},
			fn:              func(p *Provider) kmetrics.Gauge { return p.NewGauge("my-gauge").With("region", "us") },
			wantName:        "my-prefix.my-gauge",
			wantLabelValues: []string{"region", "us"},
		},

		{
			name:            "no prefix, with label values, tags enabled, default tags",
			p:               &Provider{gauges: make(map[string]*Gauge), tagsEnabled: true, defaultTags: []string{"sys", "foo"}},
			fn:              func(p *Provider) kmetrics.Gauge { return p.NewGauge("my-gauge").With("region", "us") },
			wantName:        "my-gauge",
			wantLabelValues: []string{"sys", "foo", "region", "us"},
		},

		{
			name:            "with prefix, with label values, tags enabled, default tags",
			p:               &Provider{prefix: "my-prefix", gauges: make(map[string]*Gauge), tagsEnabled: true, defaultTags: []string{"sys", "foo"}},
			fn:              func(p *Provider) kmetrics.Gauge { return p.NewGauge("my-gauge").With("region", "us") },
			wantName:        "my-prefix.my-gauge",
			wantLabelValues: []string{"sys", "foo", "region", "us"},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			gauge := test.fn(test.p)
			if got := gauge.(*Gauge).Gauge.Name; test.wantName != got {
				t.Fatalf("want name: %q, got %q", test.wantName, got)
			}

			if got := gauge.(*Gauge).LabelValues(); !reflect.DeepEqual(test.wantLabelValues, got) {
				t.Fatalf("want label values: %q, got %q", test.wantLabelValues, got)
			}
		})
	}
}

func TestHistogramNaming(t *testing.T) {
	tests := []struct {
		name            string
		p               *Provider
		fn              func(p *Provider) kmetrics.Histogram
		wantName        string
		wantLabelValues []string
	}{
		{
			name:     "no prefix, no label values",
			p:        &Provider{histograms: make(map[string]*Histogram)},
			fn:       func(p *Provider) kmetrics.Histogram { return p.NewHistogram("my-histogram", 50) },
			wantName: "my-histogram",
		},

		{
			name:     "with prefix, no label values",
			p:        &Provider{prefix: "my-prefix", histograms: make(map[string]*Histogram)},
			fn:       func(p *Provider) kmetrics.Histogram { return p.NewHistogram("my-histogram", 50) },
			wantName: "my-prefix.my-histogram",
		},

		{
			name:            "no prefix, with label values",
			p:               &Provider{histograms: make(map[string]*Histogram)},
			fn:              func(p *Provider) kmetrics.Histogram { return p.NewHistogram("my-histogram", 50).With("region", "us") },
			wantName:        "my-histogram",
			wantLabelValues: []string{"region", "us"},
		},

		{
			name:            "no prefix, with label values, tags enabled",
			p:               &Provider{histograms: make(map[string]*Histogram), tagsEnabled: true},
			fn:              func(p *Provider) kmetrics.Histogram { return p.NewHistogram("my-histogram", 50).With("region", "us") },
			wantName:        "my-histogram",
			wantLabelValues: []string{"region", "us"},
		},

		{
			name:            "with prefix, with label values, tags enabled",
			p:               &Provider{prefix: "my-prefix", histograms: make(map[string]*Histogram), tagsEnabled: true},
			fn:              func(p *Provider) kmetrics.Histogram { return p.NewHistogram("my-histogram", 50).With("region", "us") },
			wantName:        "my-prefix.my-histogram",
			wantLabelValues: []string{"region", "us"},
		},

		{
			name:            "no prefix, with label values, tags enabled, default tags",
			p:               &Provider{histograms: make(map[string]*Histogram), tagsEnabled: true, defaultTags: []string{"sys", "foo"}},
			fn:              func(p *Provider) kmetrics.Histogram { return p.NewHistogram("my-histogram", 50).With("region", "us") },
			wantName:        "my-histogram",
			wantLabelValues: []string{"sys", "foo", "region", "us"},
		},

		{
			name:            "with prefix, with label values, tags enabled, default tags",
			p:               &Provider{prefix: "my-prefix", histograms: make(map[string]*Histogram), tagsEnabled: true, defaultTags: []string{"sys", "foo"}},
			fn:              func(p *Provider) kmetrics.Histogram { return p.NewHistogram("my-histogram", 50).With("region", "us") },
			wantName:        "my-prefix.my-histogram",
			wantLabelValues: []string{"sys", "foo", "region", "us"},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			histogram := test.fn(test.p)
			if got := histogram.(*Histogram).name; test.wantName != got {
				t.Fatalf("want name: %q, got %q", test.wantName, got)
			}

			if got := histogram.(*Histogram).labelValues; !reflect.DeepEqual(test.wantLabelValues, got) {
				t.Fatalf("want label values: %q, got %q", test.wantLabelValues, got)
			}
		})
	}
}

func TestInternalMetrics(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Librato-RateLimit-Agg", "remaining=101")
		w.Header().Set("X-Librato-RateLimit-Std", "remaining=102")
	}))
	defer srv.Close()

	u, _ := url.Parse(srv.URL)

	p := New(u, 20*time.Second).(*Provider)

	c := p.NewCounter("my.counter")
	c.Add(1)

	g := p.NewGauge("my.gauge")
	g.Set(1)

	p.reportWithRetry(u, 20*time.Second)

	if got := p.ratelimitAgg.(*Gauge).Value(); got != 101 {
		t.Fatalf("want agg rate limit 101, got %f", got)
	}

	if got := p.ratelimitStd.(*Gauge).Value(); got != 102 {
		t.Fatalf("want std rate limit 101, got %f", got)
	}

	if got := p.measurements.(*Gauge).Value(); got != 5 {
		t.Fatalf("want measurements gauge to be 5, got %f", got)
	}
}

func TestRemainingRateLimit(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{
			name:  "default",
			input: "limit=600000,remaining=599998,reset=1531370400",
			want:  599998,
		},
		{
			name:  "at beginning",
			input: "remaining=599998,reset=1531370400,limit=600000",
			want:  599998,
		},
		{
			name:  "at end",
			input: "reset=1531370400,limit=600000,remaining=599998",
			want:  599998,
		},
		{
			name:  "empty",
			input: "",
			want:  -1,
		},
		{
			name:  "just remaining",
			input: "remaining=599998",
			want:  599998,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := remainingRateLimit(test.input); test.want != got {
				t.Errorf("want %d got %d", test.want, got)
			}
		})
	}
}
