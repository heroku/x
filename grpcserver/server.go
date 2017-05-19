package grpcserver

import (
	"fmt"
	"net"

	log "github.com/Sirupsen/logrus"
	proxyproto "github.com/armon/go-proxyproto"
	"github.com/heroku/cedar/lib/grpc/grpcclient"
	"github.com/heroku/cedar/lib/grpc/grpcmetrics"
	"github.com/heroku/cedar/lib/grpc/panichandler"
	"github.com/heroku/cedar/lib/grpc/testserver"
	"github.com/heroku/cedar/lib/kit/metrics"
	"github.com/heroku/cedar/lib/tlsconfig"
	"github.com/mwitkow/go-grpc-middleware"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	healthgrpc "google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
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

// A Starter registers and starts itself on the provided grpc.Server.
//
// It's expected Start will call the relevant RegisterXXXServer method
// using srv.
type Starter interface {
	Start(srv *grpc.Server) error
}

// RunStandardServer runs a GRPC server with a standard setup including metrics,
// panic handling, a health check service, TLS termination with client authentication,
// and proxy-protocol wrapping.
func RunStandardServer(logger log.FieldLogger, p metrics.Provider, port int, serverCACert, serverCert, serverKey []byte, server Starter) error {
	tlsConfig, err := tlsconfig.NewMutualTLS(serverCACert, serverCert, serverKey)
	if err != nil {
		return err
	}

	srv := grpc.NewServer(
		grpc.Creds(credentials.NewTLS(tlsConfig)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpcmetrics.NewUnaryServerInterceptor(p), panichandler.UnaryPanicHandler)),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpcmetrics.NewStreamServerInterceptor(p), panichandler.StreamPanicHandler)),
	)
	defer srv.Stop()

	healthpb.RegisterHealthServer(srv, healthgrpc.NewServer())

	if err := server.Start(srv); err != nil {
		return err
	}

	proxyprotoLn, err := NewProxyProtocolListener(port)
	if err != nil {
		return err
	}

	logger.WithFields(log.Fields{
		"at":      "binding",
		"service": "grpc-tls",
		"port":    port,
	}).Print()

	return srv.Serve(proxyprotoLn)
}
