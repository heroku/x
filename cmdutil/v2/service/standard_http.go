package service

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/joeshaw/envdecode"
	"github.com/sirupsen/logrus"

	"github.com/heroku/x/cmdutil"
	"github.com/heroku/x/cmdutil/https"
	"github.com/heroku/x/go-kit/metrics"
)

type httpConfig struct {
	Platform platformConfig
	Timeouts timeoutConfig
}

// HTTP returns a standard HTTP server for the provided handler. Port and timeout
// config are inferred from the environment.
func HTTP(l logrus.FieldLogger, m metrics.Provider, h http.Handler, opts ...func(*httpOptions)) cmdutil.Server {
	var cfg httpConfig
	envdecode.MustDecode(&cfg)

	var o httpOptions
	for _, opt := range opts {
		opt(&o)
	}

	if !o.skipEnforceHTTPS {
		h = https.RedirectHandler(h)
	}

	var srvs []cmdutil.Server

	if cfg.Platform.Port != 0 {
		s := httpServerWithTimeouts(cfg.Timeouts)
		s.Handler = h
		s.Addr = fmt.Sprintf(":%d", cfg.Platform.Port)
		o.configureServer(s)
		srvs = append(srvs, standardServer(l, s))
	}

	if cfg.Platform.AdditionalPort != 0 {
		s := httpServerWithTimeouts(cfg.Timeouts)
		s.Handler = h
		s.Addr = fmt.Sprintf(":%d", cfg.Platform.AdditionalPort)
		o.configureServer(s)
		srvs = append(srvs, standardServer(l, s))
	}

	return cmdutil.MultiServer(srvs...)
}

func httpServerWithTimeouts(t timeoutConfig) *http.Server {
	return &http.Server{
		ReadTimeout:       t.Read,
		ReadHeaderTimeout: t.ReadHeader,
		WriteTimeout:      t.Write,
		IdleTimeout:       t.Idle,
	}
}

type httpOptions struct {
	skipEnforceHTTPS bool
	serverHook       func(*http.Server)
}

func (o *httpOptions) configureServer(s *http.Server) {
	if o.serverHook != nil {
		o.serverHook(s)
	}
}

// SkipEnforceHTTPS allows services to opt-out of SSL enforcement required for
// productionization. It should only be used in environments where SSL is not
// available.
func SkipEnforceHTTPS() func(*httpOptions) {
	return func(o *httpOptions) {
		o.skipEnforceHTTPS = true
	}
}

// WithHTTPServerHook allows services to provide a function to
// adjust settings on any HTTP server before after the defaults are
// applied but before the server is started.
func WithHTTPServerHook(fn func(*http.Server)) func(*httpOptions) {
	return func(o *httpOptions) {
		o.serverHook = fn
	}
}

// listenHook allows tests to intercept the listener created for standard
// servers, e.g., to get the resolved address when the server's Addr is `:0`.
var listenHook chan net.Listener

// standardServer adapts an http.Server to a cmdutil.Server. The server is expected
// to be run behind a router and does not terminate TLS.
func standardServer(l logrus.FieldLogger, srv *http.Server) cmdutil.Server {
	return cmdutil.ServerFuncs{
		RunFunc: func() error {
			l.WithFields(logrus.Fields{
				"at":   "binding",
				"addr": srv.Addr,
			}).Info()

			ln, err := net.Listen("tcp", srv.Addr)
			if err != nil {
				return err
			}
			defer ln.Close()

			if listenHook != nil {
				listenHook <- ln
			}

			return srv.Serve(ln)
		},
		StopFunc: func(error) { gracefulShutdown(l, srv) },
	}
}

func gracefulShutdown(l logrus.FieldLogger, s *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	l.WithField("at", "graceful-shutdown").Info()
	if err := s.Shutdown(ctx); err != nil {
		l.WithField("at", "graceful-shutdown").WithError(err).Warn()
		s.Close()
	}
}
