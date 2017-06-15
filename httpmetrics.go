package httpmetrics

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/heroku/cedar/lib/kit/metricsregistry"
	"github.com/heroku/x/go-kit/metrics"
	"github.com/pressly/chi"
	"github.com/pressly/chi/middleware"
)

// NewServer returns an http.Handler which calls next for each
// request and reports metrics to the given provider.
func NewServer(p metrics.Provider, next http.Handler) http.Handler {
	reg := metricsregistry.New(p)
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

func ms(d time.Duration) float64 {
	return float64(d) / float64(time.Millisecond)
}

// turn these into dashes
var dashRe = regexp.MustCompile(`[_]+`)

func nameRoutePatterns(patterns []string) string {
	result := make([]string, len(patterns))
	for idx, pattern := range patterns {
		pattern = strings.TrimPrefix(pattern, "/")
		pattern = strings.TrimSuffix(pattern, "/*")
		parts := strings.Split(pattern, "/")
		for pidx, part := range parts {
			part = strings.TrimPrefix(part, ":")
			part = dashRe.ReplaceAllString(part, "-")
			parts[pidx] = part
		}
		result[idx] = strings.Join(parts, ".")
	}
	return strings.Join(result, ".")
}
