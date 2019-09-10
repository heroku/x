package grpcclient

import (
	"context"
	"crypto/tls"
	"net/url"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/heroku/x/grpc/requestid"
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

// AppendOutgoingRequestID reads the incoming Request-ID from the context and appends it to the
// outgoing context. Forwarding Request-IDs from gRPC service to service allows for request
// tracking across any number of services.
func AppendOutgoingRequestID() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx = requestid.AppendToOutgoingContext(ctx)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
