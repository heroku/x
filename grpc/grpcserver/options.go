package grpcserver

import (
	"context"
	"crypto/tls"
	"crypto/x509"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ocgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"

	"github.com/heroku/x/go-kit/metrics"
	"github.com/heroku/x/grpc/grpcmetrics"
	"github.com/heroku/x/grpc/panichandler"
	"github.com/heroku/x/tlsconfig"
)

var defaultLogOpts = []grpc_logrus.Option{
	grpc_logrus.WithCodes(ErrorToCode),
}

type options struct {
	logEntry               *logrus.Entry
	metricsProvider        metrics.Provider
	authUnaryInterceptor   grpc.UnaryServerInterceptor
	authStreamInterceptor  grpc.StreamServerInterceptor
	useValidateInterceptor bool

	grpcOptions []grpc.ServerOption
}

// ServerOption sets optional fields on the standard gRPC server
type ServerOption func(*options)

// GRPCOption adds a grpc ServerOption to the server.
func GRPCOption(opt grpc.ServerOption) ServerOption {
	return func(o *options) {
		o.grpcOptions = append(o.grpcOptions, opt)
	}
}

// LogEntry provided will be added to the context
func LogEntry(entry *logrus.Entry) ServerOption {
	return func(o *options) {
		o.logEntry = entry
	}
}

// MetricsProvider will have metrics reported to it
func MetricsProvider(provider metrics.Provider) ServerOption {
	return func(o *options) {
		o.metricsProvider = provider
	}
}

// AuthInterceptors sets interceptors that are intended for
// authentication/authorization in the correct locations in the chain
func AuthInterceptors(unary grpc.UnaryServerInterceptor, stream grpc.StreamServerInterceptor) ServerOption {
	return func(o *options) {
		o.authUnaryInterceptor = unary
		o.authStreamInterceptor = stream
	}
}

// WithOCGRPCServerHandler sets the grpc server up with provided ServerHandler
// as its StatsHandler
func WithOCGRPCServerHandler(h *ocgrpc.ServerHandler) ServerOption {
	return func(o *options) {
		o.grpcOptions = append(o.grpcOptions, grpc.StatsHandler(h))
	}
}

// ValidateInterceptor sets interceptors that will validate every
// message that has a receiver of the form `Validate() error`
//
// See github.com/mwitkow/go-proto-validators for details.
func ValidateInterceptor() ServerOption {
	return func(o *options) {
		o.useValidateInterceptor = true
	}
}

func (o *options) unaryInterceptors() []grpc.UnaryServerInterceptor {
	l := o.logEntry
	if l == nil {
		l = logrus.NewEntry(logrus.New())
	}

	i := []grpc.UnaryServerInterceptor{
		panichandler.LoggingUnaryPanicHandler(l),
		grpc_ctxtags.UnaryServerInterceptor(),
		UnaryPayloadLoggingTagger,
		unaryRequestIDTagger,
		unaryPeerNameTagger,
	}
	if o.metricsProvider != nil {
		i = append(i, grpcmetrics.NewUnaryServerInterceptor(o.metricsProvider)) // report metrics on unwrapped errors
	}
	i = append(i,
		unaryServerErrorUnwrapper, // unwrap after we've logged
		grpc_logrus.UnaryServerInterceptor(l, defaultLogOpts...),
	)
	if o.authUnaryInterceptor != nil {
		i = append(i, o.authUnaryInterceptor)
	}
	if o.useValidateInterceptor {
		i = append(i, grpc_validator.UnaryServerInterceptor())
	}

	return i
}

func (o *options) streamInterceptors() []grpc.StreamServerInterceptor {
	l := o.logEntry
	if l == nil {
		l = logrus.NewEntry(logrus.New())
	}

	i := []grpc.StreamServerInterceptor{
		panichandler.LoggingStreamPanicHandler(l),
		grpc_ctxtags.StreamServerInterceptor(),
		streamRequestIDTagger,
		streamPeerNameTagger,
	}
	if o.metricsProvider != nil {
		i = append(i, grpcmetrics.NewStreamServerInterceptor(o.metricsProvider)) // report metrics on unwrapped errors
	}
	i = append(i,
		streamServerErrorUnwrapper, // unwrap after we've logged
		grpc_logrus.StreamServerInterceptor(l, defaultLogOpts...),
	)
	if o.authStreamInterceptor != nil {
		i = append(i, o.authStreamInterceptor)
	}
	if o.useValidateInterceptor {
		i = append(i, grpc_validator.StreamServerInterceptor())
	}

	return i
}

func (o *options) serverOptions() []grpc.ServerOption {
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(o.unaryInterceptors()...)),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(o.streamInterceptors()...)),
	}
	opts = append(opts, o.grpcOptions...)
	return opts
}

// TLS returns a ServerOption which adds mutual-TLS to the gRPC server.
func TLS(caCerts [][]byte, serverCert tls.Certificate) (ServerOption, error) {
	tlsConfig, err := tlsconfig.NewMutualTLS(caCerts, serverCert)
	if err != nil {
		return nil, err
	}

	return GRPCOption(grpc.Creds(credentials.NewTLS(tlsConfig))), nil
}

// WithPeerValidator configures the gRPC server to reject calls from peers
// which do not provide a certificate or for which the provided function
// returns false.
func WithPeerValidator(f func(*x509.Certificate) bool) ServerOption {
	return func(o *options) {
		o.authStreamInterceptor = func(req interface{}, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
			if err := validatePeer(ss.Context(), f); err != nil {
				return err
			}
			return handler(req, ss)
		}
		o.authUnaryInterceptor = func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ interface{}, err error) {
			if err := validatePeer(ctx, f); err != nil {
				return nil, err
			}
			return handler(ctx, req)
		}
	}
}

func validatePeer(ctx context.Context, f func(*x509.Certificate) bool) error {
	cert, ok := getPeerCertFromContext(ctx)
	if !ok {
		//TODO: SA1019: grpc.Errorf is deprecated: use status.Errorf instead.  (staticcheck)
		return grpc.Errorf(codes.Unauthenticated, "unauthenticated") //nolint:staticcheck
	}

	if !f(cert) {
		//TODO: SA1019: grpc.Errorf is deprecated: use status.Errorf instead.  (staticcheck)
		return grpc.Errorf(codes.PermissionDenied, "forbidden") //nolint:staticcheck
	}

	return nil
}
