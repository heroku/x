package main

import (
	"context"
	"net/http"
	"os"

	"github.com/heroku/x/hmetrics"
)

func main() {
	// Don't care about canceling or errors
	go func() { hmetrics.Report(context.Background(), nil) }()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	http.ListenAndServe(":"+port, nil)
}
