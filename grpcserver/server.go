package grpcserver

import (
	"net/http"

	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/heroku/cedar/lib/grpc/grpcclient"
	"github.com/heroku/cedar/lib/grpc/requestid"
	"github.com/heroku/cedar/lib/grpc/testserver"
	"github.com/lstoll/grpce/h2c"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	xcontext "golang.org/x/net/context"
	"google.golang.org/grpc"
	healthgrpc "google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// New configures a gRPC Server with default options and a health server.
func New(opts ...ServerOption) *grpc.Server {
	o := &options{}
	for _, so := range opts {
		so(o)
	}

	srv := grpc.NewServer(o.serverOptions()...)

	healthpb.RegisterHealthServer(srv, healthgrpc.NewServer())

	return srv
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

// RunStandardServer runs a GRPC server with a standard setup including metrics
// (if provider passed), panic handling, a health check service, TLS termination
// with client authentication, and proxy-protocol wrapping.
//
// Deprecated: RunStandardServer is now a wrapper for New and TCP with TLS
// options and a logger.
func RunStandardServer(logger log.FieldLogger, port int, serverCACerts [][]byte, serverCert, serverKey []byte, server Starter, opts ...ServerOption) error {
	tls, err := TLS(serverCACerts, serverCert, serverKey)
	if err != nil {
		return err
	}

	opts = append(opts, tls)
	opts = append(opts, LogEntry(logger.WithField("component", "grpc")))

	grpcsrv := New(opts...)

	if err := server.Start(grpcsrv); err != nil {
		return err
	}

	tcp := TCP(logger, grpcsrv, port)
	return tcp.Run()
}

// NewStandardInProcess starts a new in-proces gRPC server with the standard
// middleware and returns the server and a valid connection.
func NewStandardInProcess(opts ...ServerOption) (*grpc.Server, *grpc.ClientConn, error) {
	o := &options{}
	for _, so := range opts {
		so(o)
	}
	srv, err := NewInProcess("local", o.serverOptions()...)

	if err != nil {
		return nil, nil, err
	}

	return srv.Server, grpcclient.Conn("local"), nil
}

// NewStandardH2C create a set of servers suitible for serving gRPC services
// using H2C (aka client upgrades). This is suitible for serving gRPC services
// via both hermes and dogwood-router. HTTP 1.x traffic will be passed to the
// provided handler. This will return a *grpc.Server configured with our
// standard set of services, and a HTTP server that should be what is served on
// a listener.
func NewStandardH2C(http11 http.Handler, opts ...ServerOption) (*grpc.Server, *http.Server) {
	o := &options{}
	for _, so := range opts {
		so(o)
	}

	gSrv := grpc.NewServer(o.serverOptions()...)

	healthpb.RegisterHealthServer(gSrv, healthgrpc.NewServer())

	h2cSrv := &h2c.Server{
		HTTP2Handler:      gSrv,
		NonUpgradeHandler: http11,
	}

	hSrv := &http.Server{Handler: h2cSrv}

	return gSrv, hSrv
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
