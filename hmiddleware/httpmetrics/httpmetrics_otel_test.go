package httpmetrics

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/heroku/x/go-kit/metrics/testmetrics"
)

func TestOTELMiddleware(t *testing.T) {
	p := testmetrics.NewProvider(t)

	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})

	r := httptest.NewRequest("GET", "http://example.org/foo/bar", nil)
	w := httptest.NewRecorder()

	hand := NewOTEL(p)(next)
	hand.ServeHTTP(w, r)

	p.CheckObservationCount("http.server.duration.http.request.method:GET:url.scheme:http:server.address:example.org", 1)
}

func TestOTELResponseStatus(t *testing.T) {
	p := testmetrics.NewProvider(t)

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})

	r := httptest.NewRequest("GET", "http://example.org/foo/bar", nil)

	rctx := chi.NewRouteContext()
	rctx.RoutePatterns = []string{"/*", "/apps/{foo_id}/bars/{bar_id}"}
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	hand := NewOTEL(p)(next)
	hand.ServeHTTP(w, r)

	p.CheckObservationCount("http.server.duration.http.request.method:GET:http.route:/apps/{foo_id}/bars/{bar_id}:url.scheme:http:server.address:example.org", 1)

}

func TestOTELNestedChiRouters(tt *testing.T) {

	cases := []struct {
		name        string
		url         string
		outerRoute  string
		innerRouter func(t *testing.T) *chi.Mux
		observation string
	}{
		{
			name:       "inner outer",
			url:        "http://example.org/hello/world",
			outerRoute: "/",
			innerRouter: func(t *testing.T) *chi.Mux {
				r := chi.NewRouter()
				r.Get("/hello/{name}", func(w http.ResponseWriter, r *http.Request) {
					name := chi.URLParam(r, "id")
					if _, err := io.WriteString(w, fmt.Sprintf("Hello %s!", name)); err != nil {
						t.Fatal("unexpected error", err)
					}
				})
				return r
			},
			observation: "http.server.duration.http.request.method:GET:http.response.status_code:200:http.route:/hello/{name}:url.scheme:http:server.address:example.org",
		},
		{
			name:       "outer inner",
			url:        "http://example.org/hello/world",
			outerRoute: "/hello/{name}",
			innerRouter: func(t *testing.T) *chi.Mux {
				r := chi.NewRouter()
				r.Get("/", func(w http.ResponseWriter, r *http.Request) {
					name := chi.URLParam(r, "id")
					if _, err := io.WriteString(w, fmt.Sprintf("Hello %s!", name)); err != nil {
						t.Fatal("unexpected error", err)
					}
				})
				return r
			},
			observation: "http.server.duration.http.request.method:GET:http.response.status_code:200:http.route:/hello/{name}/:url.scheme:http:server.address:example.org",
		},
		{
			name:       "inner outer star",
			url:        "http://example.org/hello/world/1",
			outerRoute: "/",
			innerRouter: func(t *testing.T) *chi.Mux {
				r := chi.NewRouter()
				r.Get("/hello/{name}/*", func(w http.ResponseWriter, r *http.Request) {
					name := chi.URLParam(r, "id")
					if _, err := io.WriteString(w, fmt.Sprintf("Hello %s!", name)); err != nil {
						t.Fatal("unexpected error", err)
					}
				})
				return r
			},
			observation: "http.server.duration.http.request.method:GET:http.response.status_code:200:http.route:/hello/{name}/*:url.scheme:http:server.address:example.org",
		},
		{
			name:       "slash slash",
			url:        "http://example.org/",
			outerRoute: "/",
			innerRouter: func(t *testing.T) *chi.Mux {
				r := chi.NewRouter()
				r.Get("/", func(w http.ResponseWriter, r *http.Request) {
					name := chi.URLParam(r, "id")
					if _, err := io.WriteString(w, fmt.Sprintf("Hello %s!", name)); err != nil {
						t.Fatal("unexpected error", err)
					}
				})
				return r
			},
			observation: "http.server.duration.http.request.method:GET:http.response.status_code:200:http.route:/:url.scheme:http:server.address:example.org",
		},
	}

	for _, test := range cases {
		test := test
		tt.Run(test.name, func(t *testing.T) {
			p := testmetrics.NewProvider(t)
			outer := chi.NewRouter()
			outer.Use(NewOTEL(p))
			outer.Mount(test.outerRoute, test.innerRouter(t))

			r := httptest.NewRequest("GET", test.url, nil)
			w := httptest.NewRecorder()
			outer.ServeHTTP(w, r)

			p.CheckObservationCount(test.observation, 1)
		})
	}
}
