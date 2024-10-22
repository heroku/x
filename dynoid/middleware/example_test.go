package middleware_test

import (
	"io"
	"log"
	"net/http"

	"github.com/heroku/x/dynoid/middleware"
)

const Audience = "testing"

func Example() {
	authorized := middleware.AuthorizeSameSpace(Audience)
	secureHandler := authorized(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := io.WriteString(w, "Hello from a secure endpoint!\n"); err != nil {
			log.Printf("error writing response (%v)", err)
		}
	}))

	http.Handle("/secure", secureHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
