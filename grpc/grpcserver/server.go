package grpcserver

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"strconv"

	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/lstoll/grpce/h2c"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	healthgrpc "google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/peer"

	"github.com/heroku/x/cmdutil"
	"github.com/heroku/x/grpc/requestid"
)

// New configures a gRPC Server with default options and a health server.
func New(opts ...ServerOption) *grpc.Server {
	var o options
	for _, so := range opts {
		so(&o)
	}

	srv := grpc.NewServer(o.serverOptions()...)

	healthpb.RegisterHealthServer(srv, healthgrpc.NewServer())

	return srv
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
// Deprecated: Use NewStandardServer instead.
func RunStandardServer(logger log.FieldLogger, port int, serverCACerts [][]byte, serverCert, serverKey []byte, server Starter, opts ...ServerOption) error {
	cert, err := tls.X509KeyPair(serverCert, serverKey)
	if err != nil {
		return errors.Wrap(err, "creating X509 key pair")
	}

	return NewStandardServer(logger, port, serverCACerts, cert, server, opts...).Run()
}

// NewStandardServer configures a GRPC server with a standard setup including metrics
// (if provider passed), panic handling, a health check service, TLS termination
// with client authentication, and proxy-protocol wrapping.
func NewStandardServer(logger log.FieldLogger, port int, serverCACerts [][]byte, serverCert tls.Certificate, server Starter, opts ...ServerOption) cmdutil.Server {
	tls, err := TLS(serverCACerts, serverCert)
	if err != nil {
		logger.Fatal(err)
	}

	opts = append(opts, tls, LogEntry(logger.WithField("component", "grpc")))
	grpcsrv := New(opts...)

	if err := server.Start(grpcsrv); err != nil {
		logger.Fatal(err)
	}

	return TCP(logger, grpcsrv, net.JoinHostPort("", strconv.Itoa(port)))
}

// NewStandardH2C create a set of servers suitable for serving gRPC services
// using H2C (aka client upgrades). This is suitable for serving gRPC services
// via both hermes and dogwood-router. HTTP 1.x traffic will be passed to the
// provided handler. This will return a *grpc.Server configured with our
// standard set of services, and a HTTP server that should be what is served on
// a listener.
func NewStandardH2C(http11 http.Handler, opts ...ServerOption) (*grpc.Server, *http.Server) {
	var o options
	for _, so := range opts {
		so(&o)
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
func unaryServerErrorUnwrapper(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ interface{}, err error) {
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
func unaryRequestIDTagger(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ interface{}, err error) {
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

// unaryPeerNameTagger sets a grpc_ctxtags peer name tag for logging if the
// caller provider provides a mutual TLS certificate.
func unaryPeerNameTagger(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ interface{}, err error) {
	peerName := getPeerNameFromContext(ctx)
	if peerName != "" {
		grpc_ctxtags.Extract(ctx).Set("peer.name", peerName)
	}

	return handler(ctx, req)
}

// streamPeerNameTagger sets a grpc_ctxtags peer name tag for logging if the
// caller provider provides a mutual TLS certificate.
func streamPeerNameTagger(req interface{}, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	peerName := getPeerNameFromContext(ss.Context())
	if peerName != "" {
		grpc_ctxtags.Extract(ss.Context()).Set("peer.name", peerName)
	}

	return handler(req, ss)
}

func getPeerNameFromContext(ctx context.Context) string {
	cert, ok := getPeerCertFromContext(ctx)
	if !ok {
		return ""
	}
	return cert.Subject.CommonName
}

func getPeerCertFromContext(ctx context.Context) (*x509.Certificate, bool) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return nil, false
	}

	tlsAuth, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return nil, false
	}

	if len(tlsAuth.State.PeerCertificates) == 0 {
		return nil, false
	}

	return tlsAuth.State.PeerCertificates[0], true
}
