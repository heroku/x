package grpcserver

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"

	"github.com/heroku/x/go-kit/metrics"
	"github.com/heroku/x/grpc/grpcmetrics"
	"github.com/heroku/x/grpc/v2/panichandler"
	"github.com/heroku/x/tlsconfig"
)

const (
	defaultReadHeaderTimeout = 60 * time.Second
)

var defaultLogOpts = []logging.Option{
	logging.WithCodes(ErrorToCode),
}

type options struct {
	logCtx                    zerolog.Context
	metricsProvider           metrics.Provider
	authUnaryInterceptor      grpc.UnaryServerInterceptor
	authStreamInterceptor     grpc.StreamServerInterceptor
	highCardUnaryInterceptor  grpc.UnaryServerInterceptor
	highCardStreamInterceptor grpc.StreamServerInterceptor
	readHeaderTimeout         time.Duration

	useValidateInterceptor bool

	grpcOptions []grpc.ServerOption
}

func defaultOptions() options {
	return options{
		readHeaderTimeout: defaultReadHeaderTimeout,
	}
}

// ServerOption sets optional fields on the standard gRPC server
type ServerOption func(*options)

// GRPCOption adds a grpc ServerOption to the server.
func GRPCOption(opt grpc.ServerOption) ServerOption {
	return func(o *options) {
		o.grpcOptions = append(o.grpcOptions, opt)
	}
}

// LogContext provided will be added to the context
func LogContext(ctx zerolog.Context) ServerOption {
	return func(o *options) {
		o.logCtx = ctx
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

// HighCardInterceptors sets interceptors that use
// Attributes/Labels on the instrumentation.
func HighCardInterceptors(unary grpc.UnaryServerInterceptor, stream grpc.StreamServerInterceptor) ServerOption {
	return func(o *options) {
		o.highCardUnaryInterceptor = unary
		o.highCardStreamInterceptor = stream
	}
}

func WithReadHeaderTimeout(d time.Duration) ServerOption {
	return func(o *options) {
		o.readHeaderTimeout = d
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
	logger := o.logCtx.Logger()
	i := []grpc.UnaryServerInterceptor{
		panichandler.LoggingUnaryPanicHandler(logger),
		grpc_ctxtags.UnaryServerInterceptor(),
		UnaryPayloadLoggingTagger,
		unaryRequestIDTagger,
		unaryPeerNameTagger,
	}

	if o.highCardUnaryInterceptor != nil {
		i = append(i, o.highCardUnaryInterceptor)
	} else if o.metricsProvider != nil {
		i = append(i, grpcmetrics.NewUnaryServerInterceptor(o.metricsProvider)) // report metrics on unwrapped errors
	}

	i = append(i,
		unaryServerErrorUnwrapper, // unwrap after we've logged
		logging.UnaryServerInterceptor(InterceptorLogger(logger), defaultLogOpts...),
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
	logger := o.logCtx.Logger()
	i := []grpc.StreamServerInterceptor{
		panichandler.LoggingStreamPanicHandler(logger),
		grpc_ctxtags.StreamServerInterceptor(),
		streamRequestIDTagger,
		streamPeerNameTagger,
	}

	if o.highCardStreamInterceptor != nil {
		i = append(i, o.highCardStreamInterceptor)
	} else if o.metricsProvider != nil {
		i = append(i, grpcmetrics.NewStreamServerInterceptor(o.metricsProvider)) // report metrics on unwrapped errors
	}

	i = append(i,
		streamServerErrorUnwrapper, // unwrap after we've logged
		logging.StreamServerInterceptor(InterceptorLogger(logger), defaultLogOpts...),
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
		// TODO: SA1019: grpc.Errorf is deprecated: use status.Errorf instead.  (staticcheck)
		return grpc.Errorf(codes.Unauthenticated, "unauthenticated") //nolint:staticcheck
	}

	if !f(cert) {
		// TODO: SA1019: grpc.Errorf is deprecated: use status.Errorf instead.  (staticcheck)
		return grpc.Errorf(codes.PermissionDenied, "forbidden") //nolint:staticcheck
	}

	return nil
}

func InterceptorLogger(l zerolog.Logger) logging.Logger {
	return logging.LoggerFunc(func(ctx context.Context, lvl logging.Level, msg string, fields ...any) {
		l := l.With().Fields(fields).Logger()

		switch lvl {
		case logging.LevelDebug:
			l.Debug().Msg(msg)
		case logging.LevelInfo:
			l.Info().Msg(msg)
		case logging.LevelWarn:
			l.Warn().Msg(msg)
		case logging.LevelError:
			l.Error().Msg(msg)
		default:
			panic(fmt.Sprintf("unknown level %v", lvl))
		}
	})
}
