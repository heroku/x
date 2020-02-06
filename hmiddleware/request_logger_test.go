package hmiddleware

import (
	"net/http"
	"testing"
	"time"

	"github.com/heroku/x/testing/testlog"

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

	hook.CheckContained(t,
		`level="info"`,
		`at="finished"`,
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

	hook.CheckContained(t,
		`level="error"`,
		`msg="unhandled panic"`,
	)
}
