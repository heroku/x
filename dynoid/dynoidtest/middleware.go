package dynoidtest

import (
	"context"
	"fmt"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/heroku/x/dynoid"
)

// Generate tokens using a local Issuer for the audiences provided and mount
// them dynoid.DefaultFS. Additionally the oidc client is configured to use the
// local issuer.
func LocalIssuer(audiences []string, opts ...IssuerOpt) func(http.Handler) http.Handler {
	_, iss, err := NewWithContext(context.Background(), opts...)
	if err != nil {
		panic(fmt.Sprintf("error creating test issuer (%v)", err))
	}

	tokens := map[string]string{}
	for _, au := range audiences {
		token, err := iss.GenerateIDToken(au)
		if err != nil {
			panic(fmt.Sprintf("error creating token for %q (%v)", au, err))
		}

		tokens[au] = token
	}

	dynoid.DefaultFS = NewFS(tokens)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(oidc.ClientContext(r.Context(), iss.HTTPClient())))
		})
	}
}
