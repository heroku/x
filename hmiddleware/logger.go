package hmiddleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/middleware"

	"github.com/heroku/x/requestid"

	"github.com/sirupsen/logrus"
)

// PreRequestLogger is a middleware for the github.com/sirupsen/logrus to log requests.
// It logs things similar to heroku logs and adds remote_addr and user_agent.
// If you are using Chi, consider using the RequestLogger middleware instead.
func PreRequestLogger(l logrus.FieldLogger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ww, ok := w.(middleware.WrapResponseWriter)
			if !ok {
				ww = middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			}
			logRequest(l, r, 0, 0, 0, "start")
			next.ServeHTTP(ww, r)
		}
		return http.HandlerFunc(fn)
	}
}

// PostRequestLogger is a middleware for the github.com/sirupsen/logrus to log requests.
// It logs things similar to heroku logs and adds remote_addr and user_agent.
// If you are using Chi, consider using the RequestLogger middleware instead.
func PostRequestLogger(l logrus.FieldLogger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ww, ok := w.(middleware.WrapResponseWriter)
			if !ok {
				ww = middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			}

			t0 := time.Now()
			defer func() {
				logRequest(l, r, ww.Status(), ww.BytesWritten(), time.Since(t0), "finish")
			}()
			next.ServeHTTP(ww, r)
		}
		return http.HandlerFunc(fn)
	}
}

func logRequest(l logrus.FieldLogger, r *http.Request, status int, bytes int, service time.Duration, at string) {
	log := l.WithFields(logrus.Fields{
		"request_id":  requestid.Get(r),
		"method":      r.Method,
		"host":        r.Host,
		"path":        r.URL.RequestURI(),
		"remote_addr": r.RemoteAddr,
		"user_agent":  r.UserAgent(),
		"protocol":    r.URL.Scheme,
		"at":          at,
	})

	if status > 0 {
		log = log.WithField("status", status)
	}

	if bytes > 0 {
		log = log.WithField("bytes", bytes)
	}

	if service > 0 {
		log = log.WithField("service", fmt.Sprintf("%dms", service/time.Millisecond))
	}

	if robot := r.Header.Get("X-Heroku-Robot"); robot != "" {
		log = log.WithField("robot", robot)
	}

	log.Info()
}
