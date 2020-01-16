package httpmetrics_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi"

	"github.com/heroku/x/go-kit/metrics/testmetrics"
	"github.com/heroku/x/hmiddleware/httpmetrics"
)

// This example demonstrates how to set up metrics and what will be collected on requests
func Example() {
	// Create a new Metrics Provider
	provider := testmetrics.NewProvider(&testing.T{})
	r := chi.NewRouter()

	// Use the Metrics Provider to create a new HTTP Handler
	r.Use(httpmetrics.New(provider))

	// Metrics will be collected around http OK statuses
	// For all requests and for /foo requests
	r.Get("/foo", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Metrics will be collected around http Bad Request statuses
	// For all requests and for /bar requests
	r.Get("/bar", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))

	server := httptest.NewServer(r)
	defer server.Close()

	// Metrics will be collected for each request
	req, _ := http.NewRequest("GET", server.URL+"/foo", nil)
	if _, err := http.DefaultClient.Do(req); err != nil {
		fmt.Println(err)
	}

	req, _ = http.NewRequest("GET", server.URL+"/bar", nil)
	if _, err := http.DefaultClient.Do(req); err != nil {
		fmt.Println(err)
	}

	provider.PrintCounterValue("http.server.get.bar.response-statuses.400")
	provider.PrintCounterValue("http.server.all.requests")
	provider.PrintCounterValue("http.server.all.response-statuses.200")
	provider.PrintCounterValue("http.server.get.foo.requests")
	provider.PrintCounterValue("http.server.get.foo.response-statuses.200")
	provider.PrintCounterValue("http.server.all.response-statuses.400")
	provider.PrintCounterValue("http.server.get.bar.requests")

	// Output:
	// http.server.get.bar.response-statuses.400: 1
	// http.server.all.requests: 2
	// http.server.all.response-statuses.200: 1
	// http.server.get.foo.requests: 1
	// http.server.get.foo.response-statuses.200: 1
	// http.server.all.response-statuses.400: 1
	// http.server.get.bar.requests: 1
}
