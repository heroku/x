package basicauth

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/heroku/x/go-kit/metrics/testmetrics"
)

// Authenticate returns a handler that passes a request to the underlying
// handler if basic auth credentials are valid.
func TestAuthenticate(t *testing.T) {
	metricsProvider := testmetrics.NewProvider(t)
	checker := NewChecker([]Credential{
		{Username: "username", Password: "password"},
	})
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	s := httptest.NewServer(checker.Authenticate(metricsProvider)(h))
	uri, _ := url.Parse(s.URL)
	uri.User = url.UserPassword("username", "password")
	client := &http.Client{}
	req, _ := http.NewRequest("GET", s.URL, nil)
	req.SetBasicAuth("username", "password")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got %d but want %d", resp.StatusCode, http.StatusOK)
	}
	metricsProvider.CheckCounter("server.request-auth-failures", 0)
}

// Authenticate returns a handler that returns an HTTP 403 Forbidden and
// doesn't allow the request to passthrough to the underlying handler if
// an unknown username is provided.
func TestAuthenticateWithUnknownUsername(t *testing.T) {
	metricsProvider := testmetrics.NewProvider(t)
	checker := NewChecker([]Credential{
		{Username: "username", Password: "password"},
	})
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	s := httptest.NewServer(checker.Authenticate(metricsProvider)(h))
	uri, _ := url.Parse(s.URL)
	uri.User = url.UserPassword("username", "password")
	client := &http.Client{}
	req, _ := http.NewRequest("GET", s.URL, nil)
	req.SetBasicAuth("invalid", "password")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("got %d but want %d", resp.StatusCode, http.StatusForbidden)
	}
	metricsProvider.CheckCounter("server.request-auth-failures", 1)
}

// Authenticate returns a handler that returns an HTTP 403 Forbidden and
// doesn't allow the request to passthrough to the underlying handler if an
// invalid password is provided.
func TestAuthenticateWithInvalidPassword(t *testing.T) {
	metricsProvider := testmetrics.NewProvider(t)
	checker := NewChecker([]Credential{
		{Username: "username", Password: "password"},
	})
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	s := httptest.NewServer(checker.Authenticate(metricsProvider)(h))
	uri, _ := url.Parse(s.URL)
	uri.User = url.UserPassword("username", "password")
	client := &http.Client{}
	req, _ := http.NewRequest("GET", s.URL, nil)
	req.SetBasicAuth("username", "invalid")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("got %d but want %d", resp.StatusCode, http.StatusForbidden)
	}
	metricsProvider.CheckCounter("server.request-auth-failures", 1)
}
