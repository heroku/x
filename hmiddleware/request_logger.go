package hmiddleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/pkg/errors"

	"github.com/heroku/x/requestid"

	tags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/sirupsen/logrus"
)

// StructuredLogger implements the LogFormatter interface from Chi.
// LogFormatter initiates the beginning of a new LogEntry per request.
// See https://github.com/go-chi/chi/blob/708d187cdc2beff37b6835250dd574f395ebaa03/middleware/logger.go#L54
// For an example of how to use this middleware to combine Logrus logging with the Chi router, see
// https://github.com/go-chi/chi/blob/cca4135d8dddff765463feaf1118047a9e506b4a/_examples/logging/main.go#L2
type StructuredLogger struct {
	Logger logrus.FieldLogger
}

// StructuredLoggerEntry implements the LogEntry interface from Chi.
// LogEntry records the final log when a request completes.
// See https://github.com/go-chi/chi/blob/708d187cdc2beff37b6835250dd574f395ebaa03/middleware/logger.go#L60
// For an example of how to use this middleware to combine Logrus logging with the Chi router, see
// https://github.com/go-chi/chi/blob/cca4135d8dddff765463feaf1118047a9e506b4a/_examples/logging/main.go#L2
type StructuredLoggerEntry struct {
	Logger logrus.FieldLogger
	tags   tags.Tags
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

	requestTags := tags.Extract(r.Context())

	return &StructuredLoggerEntry{Logger: log, tags: requestTags}
}

// Write creates a new LogEntry at the end of a request.
func (l *StructuredLoggerEntry) Write(status, bytes int, elapsed time.Duration) {
	fields := l.tags.Values()
	if fields == nil {
		fields = make(map[string]interface{})
	}
	fields["at"] = "finish"
	fields["status"] = status
	fields["bytes"] = bytes
	fields["service"] = fmt.Sprintf("%dms", elapsed/time.Millisecond)

	l.Logger.WithFields(logrus.Fields(fields)).Info()
}

// Panic is called by Chi's Recoverer middleware.
// See https://github.com/go-chi/chi/blob/baf4ef5b139e284b297573d89daf587457153aa3/middleware/recoverer.go
func (l *StructuredLoggerEntry) Panic(v interface{}, stack []byte) {
	fields := l.tags.Values()
	if fields == nil {
		fields = make(map[string]interface{})
	}
	fields["stack"] = string(stack)
	werr := errors.Errorf("panic: %v", v)
	l.Logger.WithFields(logrus.Fields(fields)).WithError(werr).Error("unhandled panic")
}
