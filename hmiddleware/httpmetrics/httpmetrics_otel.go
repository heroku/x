package httpmetrics

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/heroku/x/go-kit/metrics"
	"github.com/heroku/x/go-kit/metricsregistry"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const (
	// metric names
	requestDuration = "http.server.duration" // duration in milliseconds

	// metric attribute keys
	routeKey         = "http.route"
	methodKey        = "http.request.method"
	statusKey        = "http.response.status_code"
	serverAddressKey = "server.address"
	urlSchemeKey     = "url.scheme"
)

// NewOTEL returns an HTTP middleware which captures HTTP request counts and latency
// annotated with attributes for method, route, status.
//
// See https://opentelemetry.io/docs/specs/otel/metrics/semantic_conventions/http-metrics/
func NewOTEL(p metrics.Provider) func(http.Handler) http.Handler {
	reg := metricsregistry.New(p)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			start := time.Now()
			next.ServeHTTP(ww, r)
			dur := time.Since(start)

			labels := []string{
				methodKey, r.Method,
			}

			if status := ww.Status(); status != 0 {
				kv := []string{statusKey, strconv.Itoa(status)}
				labels = append(labels, kv...)
			}

			ctx := r.Context()
			if ctx.Value(chi.RouteCtxKey) != nil {
				rtCtx := chi.RouteContext(ctx)
				if len(rtCtx.RoutePatterns) > 0 {
					route := getRouteAsString(rtCtx.RoutePatterns)
					kv := []string{routeKey, route}
					labels = append(labels, kv...)
				}
			}

			if r.URL != nil {
				kv := []string{
					urlSchemeKey, r.URL.Scheme,
					serverAddressKey, r.URL.Host,
				}
				labels = append(labels, kv...)
			}

			reg.GetOrRegisterExplicitHistogram(requestDuration, metrics.ThirtySecondDistribution).With(labels...).Observe(ms(dur))
		})
	}
}

func getRouteAsString(patterns []string) string {
	var result string
	for _, pattern := range patterns {
		result += pattern
	}
	return strings.ReplaceAll(result, "/*/", "/")
}
