package httpmetrics

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi"

	"github.com/heroku/x/go-kit/metrics/testmetrics"
)

// This example shows how http metrics will be collected for requests
func Example(){
	// Create a new Metrics Provider
	provider := testmetrics.NewProvider(&testing.T{})
	r := chi.NewRouter()

	// Use the Metrics Provider to create a new HTTP Handler
	r.Use(func(next http.Handler) http.Handler {
		return NewServer(provider, next)
	})

	// Metrics will be collected around http OK statuses
	// For all requests and for /foo requests
	r.Get("/foo", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
		w.WriteHeader(http.StatusOK)
	}))

	// Metrics will be collected around http Bad Request statuses
	// For all requests and for /bar requests
	r.Get("/bar", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
		w.WriteHeader(http.StatusBadRequest)
	}))

	server := httptest.NewServer(r)
	defer server.Close()

	// Metrics will be collected for each request
	req, _ := http.NewRequest("GET", server.URL + "/foo", nil)
	http.DefaultClient.Do(req)

	req, _ = http.NewRequest("GET", server.URL + "/bar", nil)
	http.DefaultClient.Do(req)

	provider.PrintCounterValue("http.server.get.bar.response-statuses.400")
	provider.PrintCounterValue("http.server.all.requests")
	provider.PrintCounterValue("http.server.all.response-statuses.200")
	provider.PrintCounterValue("http.server.get.foo.requests")
	provider.PrintCounterValue("http.server.get.foo.response-statuses.200")
	provider.PrintCounterValue("http.server.all.response-statuses.400")
	provider.PrintCounterValue("http.server.get.bar.requests")

	// Output:
	// http.server.get.bar.response-statuses.400: 1
	// http.server.all.requests: 2
	// http.server.all.response-statuses.200: 1
	// http.server.get.foo.requests: 1
	// http.server.get.foo.response-statuses.200: 1
	// http.server.all.response-statuses.400: 1
	// http.server.get.bar.requests: 1
}

func TestServer(t *testing.T) {
	p := testmetrics.NewProvider(t)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	})

	r := httptest.NewRequest("GET", "http://example.org/foo/bar", nil)
	w := httptest.NewRecorder()

	hand := NewServer(p, next)
	hand.ServeHTTP(w, r)

	p.CheckCounter("http.server.all.requests", 1)
	p.CheckCounter("http.server.all.response-statuses.200", 1)
	p.CheckObservationCount("http.server.all.request-duration.ms", 1)
}

func TestServer_ResponseStatus(t *testing.T) {
	p := testmetrics.NewProvider(t)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(502)
	})

	r := httptest.NewRequest("GET", "http://example.org/foo/bar", nil)
	w := httptest.NewRecorder()

	hand := NewServer(p, next)
	hand.ServeHTTP(w, r)

	p.CheckCounter("http.server.all.requests", 1)
	p.CheckCounter("http.server.all.response-statuses.502", 1)
	p.CheckObservationCount("http.server.all.request-duration.ms", 1)
}

func TestServer_Chi(t *testing.T) {
	p := testmetrics.NewProvider(t)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	})

	r := httptest.NewRequest("GET", "http://example.org/foo/bar", nil)

	rctx := chi.NewRouteContext()
	rctx.RoutePatterns = []string{"/*", "/apps/{foo_id}/bars/{bar_id}"}
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	hand := NewServer(p, next)
	hand.ServeHTTP(w, r)

	p.CheckCounter("http.server.all.requests", 1)
	p.CheckCounter("http.server.all.response-statuses.200", 1)
	p.CheckObservationCount("http.server.all.request-duration.ms", 1)

	p.CheckCounter("http.server.get.apps.foo-id.bars.bar-id.requests", 1)
	p.CheckCounter("http.server.get.apps.foo-id.bars.bar-id.response-statuses.200", 1)
	p.CheckObservationCount("http.server.get.apps.foo-id.bars.bar-id.request-duration.ms", 1)
}

func TestServer_NestedChiRouters(t *testing.T) {
	p := testmetrics.NewProvider(t)

	inner := chi.NewRouter()
	inner.Get("/hello/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "id")
		if _, err := io.WriteString(w, fmt.Sprintf("Hello %s!", name)); err != nil {
			t.Fatal("unexpected error", err)
		}
	})

	outer := chi.NewRouter()
	outer.Use(func(next http.Handler) http.Handler {
		return NewServer(p, next)
	})
	outer.Mount("/", inner)

	r := httptest.NewRequest("GET", "http://example.org/hello/world", nil)
	w := httptest.NewRecorder()
	outer.ServeHTTP(w, r)

	p.CheckCounter("http.server.all.requests", 1)
	p.CheckCounter("http.server.all.response-statuses.200", 1)
	p.CheckObservationCount("http.server.all.request-duration.ms", 1)

	p.CheckCounter("http.server.get.hello.name.requests", 1)
	p.CheckCounter("http.server.get.hello.name.response-statuses.200", 1)
	p.CheckObservationCount("http.server.get.hello.name.request-duration.ms", 1)
}
