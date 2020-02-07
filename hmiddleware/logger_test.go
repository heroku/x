package hmiddleware

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"

	"github.com/heroku/x/testing/testlog"
)

func TestPreRequestLogger(t *testing.T) {
	logger, hook := testlog.New()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("ok"))

		if err != nil {
			t.Fatal(err)
		}
	})

	preRequestLoggerHandler := PreRequestLogger(logger)(handler)

	r := chi.NewRouter()
	r.Use(PreRequestLogger(logger))
	r.Get("/", handler)

	t.Run("using with stdlib", func(t *testing.T) {
		runPreRequestLoggerTest(t, preRequestLoggerHandler, hook)
	})

	t.Run("using with chi", func(t *testing.T) {
		runPreRequestLoggerTest(t, r, hook)
	})
}

func TestPreRequestLoggerDoesNotDoubleWrapTheResponseWriter(t *testing.T) {
	logger, hook := testlog.New()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww, ok := w.(middleware.WrapResponseWriter)
		if !ok {
			t.Error("wanted the responseWriter to be a WrapResponseWriter")
			return
		}

		if _, ok := ww.Unwrap().(middleware.WrapResponseWriter); ok {
			t.Error("the inner response writer should just be the vanilla response writer")
		}

		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("ok"))

		if err != nil {
			t.Fatal(err)
		}
	})

	preRequestLoggerHandler := PreRequestLogger(logger)(PreRequestLogger(logger)(handler))

	r := chi.NewRouter()
	r.Use(PreRequestLogger(logger))
	r.Get("/", handler)

	t.Run("using with stdlib", func(t *testing.T) {
		runPreRequestLoggerTest(t, preRequestLoggerHandler, hook)
	})

	t.Run("using with chi", func(t *testing.T) {
		runPreRequestLoggerTest(t, r, hook)
	})
}

func TestPostRequestLogger(t *testing.T) {
	logger, hook := testlog.New()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("ok"))

		if err != nil {
			t.Fatal(err)
		}
	})

	postRequestLoggerHandler := PostRequestLogger(logger)(handler)

	r := chi.NewRouter()
	r.Use(PostRequestLogger(logger))
	r.Get("/", handler)

	t.Run("using with stdlib", func(t *testing.T) {
		runPostRequestLoggerTest(t, postRequestLoggerHandler, hook)
	})

	t.Run("using with chi", func(t *testing.T) {
		runPostRequestLoggerTest(t, r, hook)
	})
}

func TestPostRequestLoggerDoesNotDoubleWrapTheResponseWriter(t *testing.T) {
	logger, hook := testlog.New()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww, ok := w.(middleware.WrapResponseWriter)
		if !ok {
			t.Error("wanted the responseWriter to be a WrapResponseWriter")
			return
		}

		if _, ok := ww.Unwrap().(middleware.WrapResponseWriter); ok {
			t.Error("the inner response writer should just be the vanilla response writer")
		}

		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("ok"))

		if err != nil {
			t.Fatal(err)
		}
	})

	postRequestLoggerHandler := PostRequestLogger(logger)(PostRequestLogger(logger)(handler))

	r := chi.NewRouter()
	r.Use(PostRequestLogger(logger))
	r.Get("/", handler)

	t.Run("using with stdlib", func(t *testing.T) {
		runPostRequestLoggerTest(t, postRequestLoggerHandler, hook)
	})

	t.Run("using with chi", func(t *testing.T) {
		runPostRequestLoggerTest(t, r, hook)
	})
}

func runPreRequestLoggerTest(t testing.TB, h http.Handler, hook *testlog.Hook) {
	defer hook.Reset()

	s := httptest.NewServer(h)
	defer s.Close()

	rsp, err := http.Get(s.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer rsp.Body.Close()

	hook.CheckAllContained(t,
		// check only for the existence of these fields
		"host=",
		"request_id=",
		"remote_addr=",
		"user_agent=",

		// check exact values on these fields
		"method=GET",
		"path=\"/\"",
		"protocol=",
		"at=start",
	)

	hook.CheckNotContained(t,
		"status=",
		"bytes=",
		"service=",
		"robot=",
	)
}

func runPostRequestLoggerTest(t testing.TB, h http.Handler, hook *testlog.Hook) {
	defer hook.Reset()

	s := httptest.NewServer(h)
	defer s.Close()

	rsp, err := http.Get(s.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer rsp.Body.Close()

	if rsp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %v, want %v", rsp.StatusCode, http.StatusOK)
	}

	if data, _ := ioutil.ReadAll(rsp.Body); string(data) != "ok" {
		t.Fatalf("Body = %v, want %v", string(data), "ok")
	}

	hook.CheckAllContained(t,
		// check only for the existence of these fields
		"host=",
		"request_id=",
		"remote_addr=",
		"service=",
		"user_agent=",

		// check exact values on these fields
		"method=GET",
		"path=\"/\"",
		"protocol=",
		"at=finish",
		"status=200",
		"bytes=2",
	)
}

func TestRobotAllLogger(t *testing.T) {
	logger, hook := testlog.New()
	defer hook.Reset()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("ok"))

		if err != nil {
			t.Fatal(err)
		}
	})

	r := chi.NewRouter()
	r.Use(PreRequestLogger(logger))
	r.Get("/", handler)

	s := httptest.NewServer(r)
	defer s.Close()

	req, err := http.NewRequest("GET", s.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Add("X-Heroku-Robot", "true")

	_, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	hook.CheckContained(t,
		"method=GET",
		"path=\"/\"",
		"robot=true",
	)
}
