package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	proxyproto "github.com/armon/go-proxyproto"
	"github.com/joeshaw/envdecode"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/acme/autocert"

	"github.com/heroku/x/cmdutil"
	"github.com/heroku/x/cmdutil/health"
	"github.com/heroku/x/cmdutil/https"
	"github.com/heroku/x/go-kit/metrics"
	"github.com/heroku/x/tlsconfig"
)

type httpConfig struct {
	Platform platformConfig
	Bypass   bypassConfig
}

// HTTP returns a standard HTTP server for the provided handler. Port, TLS, and
// router-bypass config are inferred from the environment.
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
		s := &http.Server{
			Handler: h,
			Addr:    fmt.Sprintf(":%d", cfg.Platform.Port),
		}
		o.configureServer(s)
		srvs = append(srvs, standardServer(l, s))
	}

	if cfg.Platform.AdditionalPort != 0 {
		s := &http.Server{
			Handler: h,
			Addr:    fmt.Sprintf(":%d", cfg.Platform.AdditionalPort),
		}
		o.configureServer(s)
		srvs = append(srvs, standardServer(l, s))
	}

	if cfg.Bypass.InsecurePort != 0 {
		s := &http.Server{
			Handler: h,
			Addr:    fmt.Sprintf(":%d", cfg.Bypass.InsecurePort),
		}
		o.configureServer(s)
		srvs = append(srvs, bypassServer(l, s))
	}

	if cfg.Bypass.SecurePort != 0 {
		tlsConfig := o.tlsConfig
		if tlsConfig == nil {
			tlsConfig = newTLSConfig(cfg.Bypass.TLS)
		}

		s := &http.Server{
			Handler:   h,
			Addr:      fmt.Sprintf(":%d", cfg.Bypass.SecurePort),
			TLSConfig: tlsConfig,
		}
		o.configureServer(s)
		srvs = append(srvs, bypassServer(l, s))
	}

	if cfg.Bypass.HealthPort != 0 {
		srvs = append(srvs, health.NewTCPServer(l, m, health.Config{
			Port: cfg.Bypass.HealthPort,
		}))
	}

	return cmdutil.MultiServer(srvs...)
}

type httpOptions struct {
	skipEnforceHTTPS bool
	tlsConfig        *tls.Config
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

// WithTLSConfig allows services to use a specific TLS configuration instead of
// the default one constructed from environment variables.
func WithTLSConfig(tlscfg *tls.Config) func(*httpOptions) {
	return func(o *httpOptions) {
		o.tlsConfig = tlscfg
	}
}

func newTLSConfig(cfg tlsConfig) *tls.Config {
	var (
		serverCert = []byte(cfg.ServerCert)
		serverKey  = []byte(cfg.ServerKey)
	)

	tlsConfig := tlsconfig.New()

	if cfg.UseAutocert {
		am := &autocert.Manager{
			Prompt: autocert.AcceptTOS,
		}
		tlsConfig.GetCertificate = am.GetCertificate
	} else {
		cert, err := tls.X509KeyPair(serverCert, serverKey)
		if err != nil {
			logrus.WithError(err).Fatal("unable to load TLS config")
		}

		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig
}

// listenHook allows tests to intercept the listener created for standard and
// bypass servers, e.g., to get the resolved address when the server's Addr is
// `:0`.
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

// bypassServer adapts an http.Server to a cmdutil.Server. The server is expected
// to be directly behind an ELB and uses proxyprotocol. It terminates TLS if
// TLSConfig is set on srv.
func bypassServer(l logrus.FieldLogger, srv *http.Server) cmdutil.Server {
	return cmdutil.ServerFuncs{
		RunFunc: func() error {
			l.WithFields(logrus.Fields{
				"at":   "binding",
				"addr": srv.Addr,
			}).Info()

			ln, err := net.Listen("tcp", srv.Addr)
			if err != nil {
				return errors.Wrap(err, "listening to tcp addr")
			}
			defer ln.Close()

			if listenHook != nil {
				listenHook <- ln
			}

			ln = &proxyproto.Listener{Listener: ln}

			if srv.TLSConfig != nil {
				return srv.ServeTLS(ln, "", "")
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
