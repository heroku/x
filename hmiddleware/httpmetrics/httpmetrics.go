package httpmetrics

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"

	"github.com/heroku/x/go-kit/metrics"
	"github.com/heroku/x/go-kit/metricsregistry"
)

// New returns an HTTP middleware which captures request metrics and reports
// them to the given provider.
func New(p metrics.Provider) func(http.Handler) http.Handler {
	reg := metricsregistry.New(p)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reg.GetOrRegisterCounter("http.server.all.requests").Add(1)

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			start := time.Now()
			next.ServeHTTP(ww, r)
			dur := time.Since(start)

			st := ww.Status()
			if st == 0 {
				// Assume no Write or WriteHeader means OK.
				st = http.StatusOK
			}
			sts := strconv.Itoa(st)

			reg.GetOrRegisterHistogram("http.server.all.request-duration.ms", 50).Observe(ms(dur))
			reg.GetOrRegisterCounter("http.server.all.response-statuses." + sts).Add(1)

			ctx := r.Context()
			if ctx.Value(chi.RouteCtxKey) == nil {
				return
			}
			rtCtx := chi.RouteContext(ctx)

			if len(rtCtx.RoutePatterns) == 0 {
				// Did not match a route, give up.
				return
			}

			// GET /apps/:foo/bars/:baz_id -> get.apps.foo.bars.baz-id
			met := strings.ToLower(r.Method) + "." + nameRoutePatterns(rtCtx.RoutePatterns)
			reg.GetOrRegisterCounter("http.server." + met + ".requests").Add(1)
			reg.GetOrRegisterHistogram("http.server."+met+".request-duration.ms", 50).Observe(ms(dur))
			reg.GetOrRegisterCounter("http.server." + met + ".response-statuses." + sts).Add(1)
		})
	}
}

// NewServer returns an http.Handler which calls next for each
// request and reports metrics to the given provider.
//
// Deprecated: NewServer is awkward to use since it doesn't follow the normal
// pattern for middleware. Use New instead.
func NewServer(p metrics.Provider, next http.Handler) http.Handler {
	return New(p)(next)
}

func ms(d time.Duration) float64 {
	return float64(d) / float64(time.Millisecond)
}

// turn these into dashes
var dashRe = regexp.MustCompile(`[_]+`)

// nameRoutePatterns transforms route patterns into Librato metric names.
//
// chi.Router's inject patterns into the request context. Each router that
// handles a request adds the pattern it used to match the request to the
// request context. For example, a chi.Router that mounts a sub-router on / to
// handle a set of paths might have a set of patterns like:
//
//    []string{"/*", "/kpi/v1/apps/:id"}
//
// which would be transformed into a metric called "kpi.v1.apps.id".
func nameRoutePatterns(patterns []string) string {
	result := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		pattern = strings.TrimSuffix(pattern, "/*")
		pattern = strings.TrimPrefix(pattern, "/")

		// Certain patterns, e.g. /* will become an empty string and
		// thus should be skipped after the first two transformations.
		if pattern == "" {
			continue
		}

		parts := strings.Split(pattern, "/")
		for pidx, part := range parts {
			part = strings.TrimPrefix(part, "{")
			part = strings.TrimSuffix(part, "}")
			part = dashRe.ReplaceAllString(part, "-")
			parts[pidx] = part
		}

		result = append(result, strings.Join(parts, "."))
	}

	return strings.Join(result, ".")
}
