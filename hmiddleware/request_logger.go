package hmiddleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/pkg/errors"

	"github.com/heroku/x/requestid"

	"github.com/sirupsen/logrus"
)

// StructuredLogger implements the LogFormatter interface from Chi.
// LogFormatter initiates the beginning of a new LogEntry per request.
// See https://github.com/go-chi/chi/blob/708d187cdc2beff37b6835250dd574f395ebaa03/middleware/logger.go#L54
type StructuredLogger struct {
	Logger logrus.FieldLogger
}

// StructuredLoggerEntry implements the LogEntry interface from Chi.
// LogEntry records the final log when a request completes.
// See https://github.com/go-chi/chi/blob/708d187cdc2beff37b6835250dd574f395ebaa03/middleware/logger.go#L60
type StructuredLoggerEntry struct {
	Logger logrus.FieldLogger
}

// NewLogEntry creates a new LogEntry at the start of a request.
func (l *StructuredLogger) NewLogEntry(r *http.Request) middleware.LogEntry {
	logFields := logrus.Fields{
		"request_id":  requestid.Get(r),
		"method":      r.Method,
		"host":        r.Host,
		"path":        r.URL.RequestURI(),
		"remote_addr": r.RemoteAddr,
		"user_agent":  r.UserAgent(),
		"protocol":    r.URL.Scheme,
		"at":          "start",
	}

	if robot := r.Header.Get("X-Heroku-Robot"); robot != "" {
		logFields["robot"] = robot
	}

	log := l.Logger.WithFields(logFields)
	log.Info()

	return &StructuredLoggerEntry{Logger: log}
}

// Write creates a new LogEntry at the end of a request.
func (l *StructuredLoggerEntry) Write(status, bytes int, elapsed time.Duration) {
	l.Logger.WithFields(logrus.Fields{
		"at":      "finish",
		"status":  status,
		"bytes":   bytes,
		"service": fmt.Sprintf("%dms", elapsed/time.Millisecond),
	}).Info()
}

// Panic is called by Chi's Recoverer middleware.
// See https://github.com/go-chi/chi/blob/baf4ef5b139e284b297573d89daf587457153aa3/middleware/recoverer.go
func (l *StructuredLoggerEntry) Panic(v interface{}, stack []byte) {
	werr := errors.Errorf("panic: %v", v)
	l.Logger.WithFields(logrus.Fields{
		"stack": string(stack),
	}).WithError(werr).Error("unhandled panic")
}
