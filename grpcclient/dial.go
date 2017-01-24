package grpcclient

import (
	"net/url"

	"github.com/heroku/cedar/lib/tlsconfig"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Dial initialites a secure gRPC connection to the specified server
// usual mutual TLS authentication.
func Dial(serverURL string, caCert, clientCert, clientKey []byte) (*grpc.ClientConn, error) {
	creds, err := Credentials(serverURL, caCert, clientCert, clientKey)
	if err != nil {
		return nil, err
	}

	return grpc.Dial(serverURL, creds)
}

// Credentials returns a gRPC DialOption configured for mutual TLS.
func Credentials(serverURL string, caCert, clientCert, clientKey []byte) (grpc.DialOption, error) {
	uri, err := url.Parse(serverURL)
	if err != nil {
		return nil, err
	}

	cfg, err := tlsconfig.NewMutualTLS(caCert, clientCert, clientKey)
	if err != nil {
		return nil, err
	}
	cfg.ServerName = uri.Host

	return grpc.WithTransportCredentials(credentials.NewTLS(cfg)), nil
}
