package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/heroku/x/dynoid"
)

type ctxKeyDynoID int

const (
	DynoIDKey ctxKeyDynoID = iota
	DynoIDErrKey
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
			if _, err := FromContext(r.Context()); err != nil {
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
	var token *dynoid.Token
	return func(next http.Handler) http.Handler {
		serverError := internalServerError("failed to load dyno-id")(next)
		authorize := Authorize(audience, func(issuer string) error {
			if issuer != token.IDToken.Issuer {
				return &dynoid.UntrustedIssuerError{Issuer: issuer}
			}

			return nil
		})(next)

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var err error
			if token == nil {
				token, err = dynoid.ReadLocalToken(r.Context(), audience)
				if err != nil {
					serverError.ServeHTTP(w, r)
					return
				}
			}

			authorize.ServeHTTP(w, r)
		})
	}

}

// AuthorizeSpace populates the dyno identity and blocks any requests that
// aren't from one of the given spaces.
func AuthorizeSpaces(audience, host string, spaces ...string) func(http.Handler) http.Handler {
	return Authorize(audience, dynoid.AllowHerokuSpace(host, spaces...))
}

// AddToContext adds the Token to the given context
func AddToContext(ctx context.Context, token *dynoid.Token, err error) context.Context {
	ctx = context.WithValue(ctx, DynoIDKey, token)
	ctx = context.WithValue(ctx, DynoIDErrKey, err)
	return ctx
}

// FromContext fetches the Token from the context
func FromContext(ctx context.Context) (*dynoid.Token, error) {
	token, _ := ctx.Value(DynoIDKey).(*dynoid.Token)
	err, _ := ctx.Value(DynoIDErrKey).(error)

	return token, err
}

func populateDynoID(audience string, callback dynoid.IssuerCallback) func(*http.Request) *http.Request {
	verifier := dynoid.NewWithCallback(audience, callback)

	return func(r *http.Request) *http.Request {
		ctx := r.Context()

		rawToken := tokenFromHeader(r)
		if rawToken == "" {
			return r.WithContext(AddToContext(ctx, nil, ErrTokenMissing))
		}

		token, err := verifier.Verify(r.Context(), rawToken)

		return r.WithContext(AddToContext(ctx, token, err))
	}
}

func tokenFromHeader(r *http.Request) string {
	bearer := r.Header.Get("Authorization")
	if len(bearer) > 7 && bearer[:7] == "Bearer " {
		return bearer[7:]
	}
	return ""
}

func internalServerError(error string) func(http.Handler) http.Handler {
	return func(http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, error, http.StatusInternalServerError)
		})
	}
}
