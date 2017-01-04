package grpcserver

import (
	"fmt"
	"net"

	proxyproto "github.com/armon/go-proxyproto"
	"github.com/heroku/cedar/lib/grpc/grpcclient"
	"github.com/heroku/cedar/lib/grpc/testserver"
	"github.com/heroku/cedar/lib/tlsconfig"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// NewProxyProtocolListener returns a net.Listener listening on port that is
// suitable for use with a grpc.Server.
func NewProxyProtocolListener(port int) (net.Listener, error) {
	addr := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &proxyproto.Listener{Listener: ln}, nil
}

// NewTCP returns a grpc.Server configured to authenticate using mutual TLS.
func NewTCP(serverCACert, serverCert, serverKey []byte) (*grpc.Server, error) {
	tlsConfig, err := tlsconfig.NewMutualTLS(serverCACert, serverCert, serverKey)
	if err != nil {
		return nil, err
	}
	return grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsConfig))), nil
}

// NewInProcess returns a testserver.GRPCTestServer. This should mostly stand
// in for a grpc.Server. It's started and its connection is registered in the
// global list with grpcclient.RegisterConnection(name, s.Conn).
func NewInProcess(name string) (*testserver.GRPCTestServer, error) {
	s, err := testserver.New()
	if err != nil {
		return nil, err
	}
	if err := s.Start(); err != nil {
		return nil, errors.Wrapf(err, "error initializing %s gRPC server", name)
	}
	grpcclient.RegisterConnection(name, s.Conn)
	return s, nil
}
