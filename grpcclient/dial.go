package grpcclient

import (
	"net/url"

	"github.com/heroku/runtime/lib/tlsconfig"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Credentials returns a gRPC DialOption configured for mutual TLS.
func Credentials(serverURL string, caCerts [][]byte, clientCert, clientKey []byte) (grpc.DialOption, error) {
	uri, err := url.Parse(serverURL)
	if err != nil {
		return nil, err
	}

	cfg, err := tlsconfig.NewMutualTLS(caCerts, clientCert, clientKey)
	if err != nil {
		return nil, err
	}
	cfg.ServerName = uri.Host

	return grpc.WithTransportCredentials(credentials.NewTLS(cfg)), nil
}
