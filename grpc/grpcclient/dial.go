package grpcclient

import (
	"context"
	"net/url"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/heroku/x/grpc/requestid"
	"github.com/heroku/x/tlsconfig"
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

// AppendOutgoingRequestID reads the incoming Request-ID from the context and appends it to the
// outgoing context. Forwarding Request-IDs from gRPC service to service allows for request
// tracking across any number of services.
func AppendOutgoingRequestID() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx = requestid.AppendToOutgoingContext(ctx)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
