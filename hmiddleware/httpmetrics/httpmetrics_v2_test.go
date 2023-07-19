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

func TestOTELMiddleware(t *testing.T) {
	p := testmetrics.NewProvider(t)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	})

	r := httptest.NewRequest("GET", "http://example.org/foo/bar", nil)
	w := httptest.NewRecorder()

	hand := NewOTEL(p)(next)
	hand.ServeHTTP(w, r)

	p.CheckObservationCount("http.server.duration.http.request.method:GET:url.scheme:http:server.address:example.org", 1)
}

func TestOTELResponseStatus(t *testing.T) {
	p := testmetrics.NewProvider(t)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(502)
	})

	r := httptest.NewRequest("GET", "http://example.org/foo/bar", nil)
	w := httptest.NewRecorder()

	hand := NewOTEL(p)(next)
	hand.ServeHTTP(w, r)

	p.CheckObservationCount("http.server.duration.http.request.method:GET:http.response.status_code:502:url.scheme:http:server.address:example.org", 1)
}

func TestOTELChi(t *testing.T) {
	p := testmetrics.NewProvider(t)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	})

	r := httptest.NewRequest("GET", "http://example.org/foo/bar", nil)

	rctx := chi.NewRouteContext()
	rctx.RoutePatterns = []string{"/*", "/apps/{foo_id}/bars/{bar_id}"}
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	hand := NewOTEL(p)(next)
	hand.ServeHTTP(w, r)

	p.CheckObservationCount("http.server.duration.http.request.method:GET:http.route:/apps/{foo_id}/bars/{bar_id}:url.scheme:http:server.address:example.org", 1)

}

func TestOTELNestedChiRouters(t *testing.T) {
	p := testmetrics.NewProvider(t)

	inner := chi.NewRouter()
	inner.Get("/hello/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "id")
		if _, err := io.WriteString(w, fmt.Sprintf("Hello %s!", name)); err != nil {
			t.Fatal("unexpected error", err)
		}
	})

	outer := chi.NewRouter()
	outer.Use(NewOTEL(p))
	outer.Mount("/", inner)

	r := httptest.NewRequest("GET", "http://example.org/hello/world", nil)
	w := httptest.NewRecorder()
	outer.ServeHTTP(w, r)

	p.CheckObservationCount("http.server.duration.http.request.method:GET:http.response.status_code:200:http.route:/hello/{name}:url.scheme:http:server.address:example.org", 1)

}
