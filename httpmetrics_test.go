package httpmetrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/heroku/cedar/lib/kit/metrics/testmetrics"
	"github.com/pressly/chi"
)

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
	rctx.RoutePatterns = []string{"apps", ":foo_id", "bars", ":bar_id"}
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
