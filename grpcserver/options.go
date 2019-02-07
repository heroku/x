package grpcserver

import (
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/heroku/runtime/lib/grpc/grpcmetrics"
	"github.com/heroku/runtime/lib/grpc/panichandler"
	"github.com/heroku/runtime/lib/tlsconfig"
	"github.com/heroku/x/go-kit/metrics"
	"github.com/mwitkow/go-grpc-middleware"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var defaultLogOpts = []grpc_logrus.Option{
	grpc_logrus.WithCodes(ErrorToCode),
}

// grpc library default is 4MB, lets double that
const maxMsgSize = 1024 * 1024 * 8

type options struct {
	logEntry              *logrus.Entry
	metricsProvider       metrics.Provider
	authUnaryInterceptor  grpc.UnaryServerInterceptor
	authStreamInterceptor grpc.StreamServerInterceptor
	grpcOptions           []grpc.ServerOption
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

// AuthInterceptors sets interceptors that are intented for
// authentication/authorization in the correct locations in the chain
func AuthInterceptors(unary grpc.UnaryServerInterceptor, stream grpc.StreamServerInterceptor) ServerOption {
	return func(o *options) {
		o.authUnaryInterceptor = unary
		o.authStreamInterceptor = stream
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

	return i
}

func (o *options) serverOptions() []grpc.ServerOption {
	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(maxMsgSize),
		grpc.MaxSendMsgSize(maxMsgSize),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(o.unaryInterceptors()...)),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(o.streamInterceptors()...)),
	}
	opts = append(opts, o.grpcOptions...)
	return opts
}

// TLS returns a ServerOption which adds mutual-TLS to the gRPC server.
func TLS(caCerts [][]byte, serverCert []byte, serverKey []byte) (ServerOption, error) {
	tlsConfig, err := tlsconfig.NewMutualTLS(caCerts, serverCert, serverKey)
	if err != nil {
		return nil, err
	}

	return GRPCOption(grpc.Creds(credentials.NewTLS(tlsConfig))), nil
}
