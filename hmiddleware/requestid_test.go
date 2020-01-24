package hmiddleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi"
)

func TestRequestID(t *testing.T) {
	requestID := "request-id-123"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Set("X-Request-Id", requestID)
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("ok"))

		if err != nil {
			t.Fatal(err)
		}
	})

	r := chi.NewRouter()
	r.Use(RequestID)
	r.Get("/", handler)
	s := httptest.NewServer(handler)
	defer s.Close()
	rsp, err := http.Get(s.URL)
	if err != nil {
		t.Fatal(err)
	}

	defer rsp.Body.Close()
	requestIDHeader := strings.Join(rsp.Header["X-Request-Id"], "")

	if requestIDHeader != requestID {
		t.Errorf("got %v, want %v", requestIDHeader, requestID)
	}
	if err != nil {
		t.Fatal(err)
	}
}
