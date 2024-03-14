package middleware_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi"

	"github.com/heroku/x/dynoid"
	"github.com/heroku/x/dynoid/dynoidtest"
	"github.com/heroku/x/dynoid/middleware"
)

func TestAuthorize(t *testing.T) {
	router := chi.NewRouter()
	router.Use(middleware.Authorize(
		dynoidtest.Audience,
		dynoid.AllowHerokuHost(dynoidtest.IssuerHost),
	))
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
	})

	ctx, generate := newIssuer(t)

	tests := map[string]struct {
		AuthorizationHeader string
		StatusCode          int
	}{
		"no token":        {"", http.StatusForbidden},
		"audience/other":  {generate("other"), http.StatusForbidden},
		"audience/heroku": {generate("heroku"), http.StatusOK},
	}

	for label, tc := range tests {
		t.Run(label, func(t *testing.T) {
			r := httptest.NewRequest("GET", "http://example.org/", nil).Clone(ctx)
			w := httptest.NewRecorder()

			if tc.AuthorizationHeader != "" {
				r.Header.Add("Authorization", tc.AuthorizationHeader)
			}

			router.ServeHTTP(w, r)

			resp := w.Result()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read body (%v)", err)
			}

			if resp.StatusCode != tc.StatusCode {
				t.Fatalf("expected %d got (%s -> %q)", tc.StatusCode, resp.Status, body)
			}
		})
	}
}

func newIssuer(t *testing.T) (context.Context, func(clientID string) (header string)) {
	issuer, err := dynoidtest.New()
	if err != nil {
		t.Fatalf("failed to get new issuer (%v)", err)
	}

	return issuer.Context(), func(clientID string) string {
		token, err := issuer.GenerateIDToken(clientID)
		if err != nil {
			t.Fatalf("failed to generate token (%v)", err)
		}

		return fmt.Sprintf("Bearer %s", token)
	}
}
