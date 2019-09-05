package grpcclient

import (
	"crypto/tls"
	"net/url"

	"github.com/heroku/x/tlsconfig"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/heroku/x/tlsconfig"
)

// TLSOption is a function which modifies a TLS configuration.
type TLSOption func(*tls.Config)

// Credentials returns a gRPC DialOption configured for mutual TLS.
func Credentials(serverURL string, caCerts [][]byte, fullCert tls.Certificate, tlsopts ...TLSOption) (grpc.DialOption, error) {
	uri, err := url.Parse(serverURL)
	if err != nil {
		return nil, err
	}

	cfg, err := tlsconfig.NewMutualTLS(caCerts, fullCert)
	if err != nil {
		return nil, err
	}
	cfg.ServerName = uri.Host

	for _, o := range tlsopts {
		o(cfg)
	}

	return grpc.WithTransportCredentials(credentials.NewTLS(cfg)), nil
}

// SkipVerify disables verification of server certificates.
func SkipVerify(cfg *tls.Config) {
	cfg.InsecureSkipVerify = true
}
