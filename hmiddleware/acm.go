package hmiddleware

import (
	"net/http"
	"net/url"
	"strings"
)

const http01ChallengePath = "/.well-known/acme-challenge/"

// ACMEValidationMiddleware implements the HTTP01 based redirect protocol
// specific to heroku ACM.
func ACMEValidationMiddleware(validationURL *url.URL) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, http01ChallengePath) {
				qs := validationURL.Query()
				qs.Set("token", strings.TrimPrefix(r.URL.Path, http01ChallengePath))
				qs.Set("host", r.Host)

				u := *validationURL
				u.RawQuery = qs.Encode()

				http.Redirect(w, r, u.String(), http.StatusMovedPermanently)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
