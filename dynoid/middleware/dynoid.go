package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/heroku/x/dynoid"
)

var (
	ErrTokenMissing = errors.New("token not found")
)

// Populate attempts to validate and parse a Token from the request for the
// given audience but doesn't enforce any restrictions.
func Populate(audience string, callback dynoid.IssuerCallback) func(http.Handler) http.Handler {
	populate := populateDynoID(audience, callback)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, populate(r))
		})
	}
}

// Authorize populates the dyno identity blocks requests where the callback fails.
func Authorize(audience string, callback dynoid.IssuerCallback) func(http.Handler) http.Handler {
	populate := Populate(audience, callback)

	return func(next http.Handler) http.Handler {
		return populate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, err := dynoid.FromContext(r.Context()); err != nil {
				w.WriteHeader(http.StatusForbidden)
				fmt.Fprint(w, http.StatusText(http.StatusForbidden))
				return
			}

			next.ServeHTTP(w, r)
		}))
	}
}

// AuthorizeSameSpace restricts access to tokens from the same space/issuer for
// the given audience.
func AuthorizeSameSpace(audience string) func(http.Handler) http.Handler {
	return callbackHandler(audience, func(token *dynoid.Token) dynoid.IssuerCallback {
		return func(issuer string) error {
			if issuer != token.IDToken.Issuer {
				return &dynoid.UntrustedIssuerError{Issuer: issuer}
			}

			return nil
		}
	})
}

// AuthorizeSpaces populates the dyno identity and blocks any requests that
// aren't from one of the given spaces.
func AuthorizeSpaces(audience string, spaces ...string) func(http.Handler) http.Handler {
	return callbackHandler(audience, func(token *dynoid.Token) dynoid.IssuerCallback {
		u, err := url.Parse(token.IDToken.Issuer)
		if err != nil {
			panic(fmt.Sprintf("failed to parse issuer (%v)", err))
		}

		return dynoid.AllowHerokuSpace(strings.TrimPrefix(u.Hostname(), "oidc."), spaces...)
	})
}

// AuthorizeSpacesWithIssuer populates the dyno identity and blocks any
// requests that aren't from one of the given spaces and issuer.
func AuthorizeSpacesWithIssuer(audience, issuer string, spaces ...string) func(http.Handler) http.Handler {
	return Authorize(audience, dynoid.AllowHerokuSpace(issuer, spaces...))
}

func populateDynoID(audience string, callback dynoid.IssuerCallback) func(*http.Request) *http.Request {
	verifier := dynoid.NewWithCallback(audience, callback)

	return func(r *http.Request) *http.Request {
		ctx := r.Context()

		rawToken := tokenFromHeader(r)
		if rawToken == "" {
			return r.WithContext(dynoid.ContextWithError(ctx, ErrTokenMissing))
		}

		token, err := verifier.Verify(r.Context(), rawToken)
		if err != nil {
			return r.WithContext(dynoid.ContextWithError(ctx, err))
		}

		return r.WithContext(dynoid.ContextWithToken(ctx, token))
	}
}

func tokenFromHeader(r *http.Request) string {
	bearer := r.Header.Get("Authorization")
	if len(bearer) > 7 && bearer[:7] == "Bearer " {
		return bearer[7:]
	}
	return ""
}

func callbackHandler(audience string, fn func(*dynoid.Token) dynoid.IssuerCallback) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		serverError := internalServerError("failed to load dyno-id")(next)

		var authedNext http.Handler
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if authedNext != nil {
				authedNext.ServeHTTP(w, r)
				return
			}

			token, err := dynoid.ReadLocalToken(r.Context(), audience)
			if err != nil {
				serverError.ServeHTTP(w, r)
				return
			}

			authedNext = Authorize(audience, fn(token))(next)

			authedNext.ServeHTTP(w, r)
		})
	}
}

func internalServerError(error string) func(http.Handler) http.Handler {
	return func(http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, error, http.StatusInternalServerError)
		})
	}
}
