package middleware_test

import (
	"io"
	"log"
	"net/http"

	"github.com/heroku/x/dynoid/middleware"
)

const AUDIENCE = "testing"

func Example() {
	authorized := middleware.AuthorizeSameSpace(AUDIENCE)
	secureHandler := authorized(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "Hello from a secure endpoint!\n")
	}))

	http.Handle("/secure", secureHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
