package hmiddleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi"

	"github.com/heroku/x/hcontext"
)

func TestRequestID(t *testing.T) {
	gotRequestID := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, gotRequestID = hcontext.RequestIDFromContext(r.Context())
	})

	r := chi.NewRouter()
	r.Use(RequestID)
	r.Get("/", handler)
	s := httptest.NewServer(r)
	defer s.Close()

	rsp, err := http.Get(s.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer rsp.Body.Close()

	if !gotRequestID {
		t.Errorf("expected RequestID in context")
	}
}
