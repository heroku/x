package dynoidtest_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/heroku/x/dynoid/dynoidtest"
	"github.com/heroku/x/dynoid/middleware"
)

type stack struct {
	middleware []func(http.Handler) http.Handler
}

func newStack(middleware ...func(http.Handler) http.Handler) *stack {
	return &stack{
		middleware: middleware,
	}
}

func (s *stack) Use(h func(http.ResponseWriter, *http.Request)) http.Handler {
	var next http.Handler = http.HandlerFunc(h)
	for i := len(s.middleware); i > 0; i-- {
		next = s.middleware[i-1](next)
	}

	return next
}

func TestMiddlewareWithSameSpace(t *testing.T) {
	testAudience := "testing"
	audiences := []string{testAudience}
	cfg, err := dynoidtest.ConfigureLocal(audiences)
	if err != nil {
		t.Fatalf("failed to configure local issuer (%v)", err)
	}

	withStack := newStack(cfg.Middleware(), middleware.AuthorizeSameSpace(testAudience)).Use

	mux := http.NewServeMux()
	mux.Handle("/", withStack(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	mux.Handle("/token", cfg.Handler())

	srv := httptest.NewServer(mux)
	defer srv.Close()

	baseURL, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("error parsing server url (%v)", err)
	}

	resp, err := http.Get(baseURL.String())
	if err != nil {
		t.Fatalf("error fetching index (%v)", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("unexpected status (expected: %d, actual: %d)", http.StatusUnauthorized, resp.StatusCode)
	}

	tokenURL := baseURL.JoinPath("/token")
	tokenURL.RawQuery = url.Values{
		"audience": {testAudience},
	}.Encode()

	t.Logf("audience: %v", tokenURL)

	tokenResp, err := http.Get(tokenURL.String())
	if err != nil {
		t.Fatalf("error fetching token (%v)", err)
	}
	defer tokenResp.Body.Close()

	body := new(strings.Builder)
	if n, err := io.Copy(body, tokenResp.Body); err != nil || n != tokenResp.ContentLength {
		t.Fatalf("failed to read body (%v)", err)
	}

	if tokenResp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status fetching token (%d %q)", tokenResp.StatusCode, body.String())
	}

	req, err := http.NewRequest(http.MethodGet, baseURL.String(), nil)
	if err != nil {
		t.Fatalf("error creating request (%v)", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", body.String()))

	indexResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("error fetching index when authed (%v)", err)
	}

	if indexResp.StatusCode != http.StatusNoContent {
		t.Fatalf("unexpected status (expected: %d, actual: %d)", http.StatusNoContent, indexResp.StatusCode)
	}
}
