package https

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"

	proxyproto "github.com/armon/go-proxyproto"
	"github.com/pkg/errors"
	"golang.org/x/crypto/acme/autocert"

	"github.com/heroku/x/tlsconfig"
)

// Serve starts an HTTP server configured to handle traffic directly from an
// ELB on port and terminate TLS using serverCert and serverKey.
func Serve(handler http.Handler, cfg Config, opts ...func(*http.Server)) error {
	var (
		serverCert = []byte(cfg.ServerCert)
		serverKey  = []byte(cfg.ServerKey)
		srv        = &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.SecurePort),
			Handler: handler,
		}
	)

	for _, opt := range opts {
		opt(srv)
	}

	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return errors.Wrap(err, "listening to tcp addr")
	}
	ln = &proxyproto.Listener{Listener: ln}

	tlsConfig := tlsconfig.New()

	if cfg.UseAutocert {
		am := &autocert.Manager{
			Prompt: autocert.AcceptTOS,
		}
		tlsConfig.GetCertificate = am.GetCertificate
	} else {
		cert, err := tls.X509KeyPair(serverCert, serverKey)
		if err != nil {
			return errors.Wrap(err, "decoding TLS certificate")
		}

		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	ln = tls.NewListener(ln, tlsConfig)
	defer ln.Close()

	err = srv.Serve(ln)
	if err != nil {
		return errors.Wrap(err, "serve")
	}
	return nil
}
