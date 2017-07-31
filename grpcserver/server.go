package grpcserver

import (
	"fmt"
	"net"

	proxyproto "github.com/armon/go-proxyproto"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/heroku/cedar/lib/grpc/grpcclient"
	"github.com/heroku/cedar/lib/grpc/grpcmetrics"
	"github.com/heroku/cedar/lib/grpc/panichandler"
	"github.com/heroku/cedar/lib/grpc/requestid"
	"github.com/heroku/cedar/lib/grpc/testserver"
	"github.com/heroku/cedar/lib/tlsconfig"
	"github.com/heroku/x/go-kit/metrics"
	"github.com/mwitkow/go-grpc-middleware"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	xcontext "golang.org/x/net/context"
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
func NewTCP(serverCACertList [][]byte, serverCert, serverKey []byte) (*grpc.Server, error) {
	tlsConfig, err := tlsconfig.NewMutualTLS(serverCACertList, serverCert, serverKey)
	if err != nil {
		return nil, err
	}
	return grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsConfig))), nil
}

// NewInProcess returns a testserver.GRPCTestServer. This should mostly stand
// in for a grpc.Server. It's started and its connection is registered in the
// global list with grpcclient.RegisterConnection(name, s.Conn).
func NewInProcess(name string, opts ...grpc.ServerOption) (*testserver.GRPCTestServer, error) {
	s, err := testserver.New(opts...)
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
func RunStandardServer(logger log.FieldLogger, p metrics.Provider, port int, serverCACerts [][]byte, serverCert, serverKey []byte, server Starter) error {
	tlsConfig, err := tlsconfig.NewMutualTLS(serverCACerts, serverCert, serverKey)
	if err != nil {
		return err
	}

	options := []grpc.ServerOption{
		grpc.Creds(credentials.NewTLS(tlsConfig)),
	}

	grpcLogger := logger.WithField("component", "grpc")
	options = append(options, StandardOptions(grpcLogger, p)...)

	srv := grpc.NewServer(options...)
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

// StandardOptions return a list of standard server options to initialize
// servers.
func StandardOptions(l *log.Entry, p metrics.Provider) []grpc.ServerOption {
	logOpts := []grpc_logrus.Option{
		grpc_logrus.WithCodes(ErrorToCode),
	}

	return []grpc.ServerOption{
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			panichandler.LoggingUnaryPanicHandler(l),
			grpc_ctxtags.UnaryServerInterceptor(),
			UnaryPayloadLoggingTagger,
			unaryRequestIDTagger,
			grpcmetrics.NewUnaryServerInterceptor(p), // report metrics on unwrapped errors
			unaryServerErrorUnwrapper,                // unwrap after we've logged
			grpc_logrus.UnaryServerInterceptor(l, logOpts...),
		)),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			panichandler.LoggingStreamPanicHandler(l),
			grpc_ctxtags.StreamServerInterceptor(),
			streamRequestIDTagger,
			grpcmetrics.NewStreamServerInterceptor(p), // report metrics on unwrapped errors
			streamServerErrorUnwrapper,                // unwrap after we've logged
			grpc_logrus.StreamServerInterceptor(l, logOpts...),
		)),
	}
}

// NewStandardInProcess starts a new in-proces gRPC server with the standard
// middleware and returns the server and a valid connection.
func NewStandardInProcess(l *log.Entry, p metrics.Provider) (*grpc.Server, *grpc.ClientConn, error) {
	srv, err := NewInProcess("local", StandardOptions(l, p)...)

	if err != nil {
		return nil, nil, err
	}

	return srv.Server, grpcclient.Conn("local"), nil
}

// unaryServerErrorUnwrapper removes errors.Wrap annotations from errors so
// gRPC status codes are correctly returned to interceptors and clients later
// in the chain.
func unaryServerErrorUnwrapper(ctx xcontext.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ interface{}, err error) {
	res, err := handler(ctx, req)
	return res, errors.Cause(err)
}

// streamServerErrorUnwrapper removes errors.Wrap annotations from errors so
// gRPC status codes are correctly returned to interceptors and clients later
// in the chain.
func streamServerErrorUnwrapper(srv interface{}, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	err := handler(srv, ss)
	return errors.Cause(err)
}

// unaryRequestIDTagger sets a grpc_ctxtags request_id tag for logging if the
// context includes a request ID.
func unaryRequestIDTagger(ctx xcontext.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ interface{}, err error) {
	if id, ok := requestid.FromContext(ctx); ok {
		grpc_ctxtags.Extract(ctx).Set("request_id", id)
	}

	return handler(ctx, req)
}

// streamRequestIDTagger sets a grpc_ctxtags request_id tag for logging if the
// context includes a request ID.
func streamRequestIDTagger(req interface{}, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	if id, ok := requestid.FromContext(ss.Context()); ok {
		grpc_ctxtags.Extract(ss.Context()).Set("request_id", id)
	}

	return handler(req, ss)
}
