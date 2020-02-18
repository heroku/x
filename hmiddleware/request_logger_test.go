package hmiddleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi"
	tags "github.com/grpc-ecosystem/go-grpc-middleware/tags"

	"github.com/heroku/x/testing/testlog"

	"github.com/go-chi/chi/middleware"
	"github.com/sirupsen/logrus"
)

func TestNewLogEntry(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Heroku-Robot", "I am a robot")
	logger := logrus.New()
	l := &StructuredLogger{
		Logger: logger,
	}

	e := l.NewLogEntry(req)
	if e == nil {
		t.Error("error creating NewLogEntry.")
	}
}

func TestLogEntryWrite(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	logger, hook := testlog.New()
	l := &StructuredLogger{
		Logger: logger,
	}
	e := l.NewLogEntry(req)
	status := 200
	bytes := 0
	elapsed := time.Duration(100)

	e.Write(status, bytes, elapsed)

	hook.CheckAllContained(t,
		`level="info"`,
		`at="finish"`,
		`status="200"`,
	)
}

func TestLogEntryPanic(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	logger, hook := testlog.New()
	l := &StructuredLogger{
		Logger: logger,
	}
	e := l.NewLogEntry(req)

	var i interface{}
	b := []byte{65, 66}
	e.Panic(&i, b)

	hook.CheckAllContained(t,
		`level="error"`,
		`msg="unhandled panic"`,
	)
}

func TestTags(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tags.Extract(r.Context()).Set("foo", "bar")
		w.WriteHeader(http.StatusOK)
	})
	r := chi.NewRouter()
	r.Use(Tags)
	logger, hook := testlog.New()
	l := &StructuredLogger{
		Logger: logger,
	}
	r.Use(middleware.RequestLogger(l))
	r.Get("/", handler)
	s := httptest.NewServer(r)
	defer s.Close()

	rsp, err := http.Get(s.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer rsp.Body.Close()

	hook.CheckAllContained(t,
		`level="info"`,
		`at="finish"`,
		`status="200"`,
		`foo="bar"`,
	)
}

func TestTagsNoMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tags.Extract(r.Context()).Set("foo", "bar")
		w.WriteHeader(http.StatusOK)
	})
	r := chi.NewRouter()
	logger, hook := testlog.New()
	l := &StructuredLogger{
		Logger: logger,
	}
	r.Use(middleware.RequestLogger(l))
	r.Get("/", handler)
	s := httptest.NewServer(r)
	defer s.Close()

	rsp, err := http.Get(s.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer rsp.Body.Close()

	hook.CheckAllContained(t,
		`level="info"`,
		`at="finish"`,
		`status="200"`,
	)
	hook.CheckNotContained(t,
		`foo="bar"`,
	)
}

func TestTagsPanic(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tags.Extract(r.Context()).Set("foo", "bar")
		panic("ohno!")
	})
	r := chi.NewRouter()
	r.Use(Tags)
	logger, hook := testlog.New()
	l := &StructuredLogger{
		Logger: logger,
	}
	r.Use(middleware.RequestLogger(l))
	r.Use(middleware.Recoverer) // recovers from panics

	r.Get("/", handler)
	s := httptest.NewServer(r)
	defer s.Close()

	rsp, err := http.Get(s.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer rsp.Body.Close()

	hook.CheckAllContained(t,
		`level="error"`,
		`msg="unhandled panic"`,
		`foo="bar"`,
	)
}
