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
		reg.GetOrRegisterCounter("all.requests").Add(1)

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

		reg.GetOrRegisterHistogram("all.request-duration.ms", 50).Observe(ms(dur))
		reg.GetOrRegisterCounter("all.response-statuses." + sts).Add(1)

		ctx := r.Context()
		if ctx.Value(chi.RouteCtxKey) == nil {
			return
		}
		rtCtx := chi.RouteContext(ctx)

		// GET /apps/:foo/bars/:baz_id -> get.apps.foo.bars.baz-id
		met := strings.ToLower(r.Method) + "." + joinRoutePatterns(rtCtx.RoutePatterns)
		reg.GetOrRegisterCounter(met + ".requests").Add(1)
		reg.GetOrRegisterHistogram(met+".request-duration.ms", 50).Observe(ms(dur))
		reg.GetOrRegisterCounter(met + ".response-statuses." + sts).Add(1)
	})
}

func ms(d time.Duration) float64 {
	return float64(d) / float64(time.Millisecond)
}

// turn these into dashes
var dashRe = regexp.MustCompile(`[_]+`)

func joinRoutePatterns(patterns []string) string {
	result := make([]string, len(patterns))
	for idx, pattern := range patterns {
		result[idx] = dashRe.ReplaceAllString(strings.TrimPrefix(strings.TrimSuffix(pattern, "/*"), ":"), "-")
	}
	return strings.Join(result, ".")
}
