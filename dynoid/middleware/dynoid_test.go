package middleware_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/heroku/x/dynoid"
	"github.com/heroku/x/dynoid/dynoidtest"
	"github.com/heroku/x/dynoid/middleware"
)

var noOp = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

func TestAuthorize(t *testing.T) {
	handler := middleware.Authorize(dynoidtest.Audience, dynoid.AllowHerokuHost(dynoidtest.IssuerHost))(noOp)

	ctx, generate := newTokenGenerator(t)

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

			handler.ServeHTTP(w, r)

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

func newTokenGenerator(t *testing.T) (context.Context, func(clientID string) (header string)) {
	ctx, issuer, err := dynoidtest.NewWithContext(context.Background())
	if err != nil {
		t.Fatalf("failed to get new issuer (%v)", err)
	}

	return ctx, func(clientID string) string {
		token, err := issuer.GenerateIDToken(clientID)
		if err != nil {
			t.Fatalf("failed to generate token (%v)", err)
		}

		return fmt.Sprintf("Bearer %s", token)
	}
}
