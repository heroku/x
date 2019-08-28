package basicauth

import (
	"net/http"

	"github.com/heroku/x/go-kit/metrics"
)

// Authenticate creates a middleware style HTTP handler that enforces valid
// basic auth credentials before allowing a request to continue to handler.
func (c *Checker) Authenticate(metricsProvider metrics.Provider) func(handler http.Handler) http.Handler {
	counter := metricsProvider.NewCounter("server.request-auth-failures")
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			username, password, ok := r.BasicAuth()
			if !ok || !c.Valid(username, password) {
				w.WriteHeader(http.StatusForbidden)
				counter.Add(1)
				return
			}
			h.ServeHTTP(w, r)
		})
	}
}
