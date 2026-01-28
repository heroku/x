package https

import (
	"net/http"

	"github.com/unrolled/secure"
)

// RedirectHandler takes an http.Handler and returns a secured form of it.
func RedirectHandler(h http.Handler) http.Handler {
	return secure.New(secure.Options{
		SSLRedirect: true,
		SSLProxyHeaders: map[string]string{
			"X-Forwarded-Proto": "https",
		},

		// Set the Strict-Transport-Security HTTP header's max-age to 31536000 (1 year)
		// see https://security.herokai.com/security_reviews/243
		STSSeconds: 31536000,
	}).Handler(h)
}
