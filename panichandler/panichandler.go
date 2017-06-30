package panichandler

import (
	log "github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var _ grpc.UnaryServerInterceptor = UnaryPanicHandler
var _ grpc.StreamServerInterceptor = StreamPanicHandler

// UnaryPanicHandler is an interceptor to catch panics and return err code 13
// with a description of the panic.
func UnaryPanicHandler(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	defer handleCrash(func(r interface{}) {
		err = toPanicError(r)
	})
	return handler(ctx, req)
}

// StreamPanicHandler is an interceptor to catch panics and return err code 13
// with a description of the panic.
func StreamPanicHandler(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
	defer handleCrash(func(r interface{}) {
		err = toPanicError(r)
	})
	return handler(srv, stream)
}

// LoggingUnaryPanicHandler returns a server interceptor which recovers
// panics, logs them as errors with logger, and returns a gRPC internal
// error to clients.
func LoggingUnaryPanicHandler(logger log.FieldLogger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer handleCrash(func(r interface{}) {
			werr := errors.Errorf("grpc unary server panic: %v", r)
			logger.WithError(werr).Error("grpc unary server panic")
			err = toPanicError(werr)
		})
		return handler(ctx, req)
	}
}

// LoggingStreamPanicHandler returns a stream server interceptor which
// recovers panics, logs them as errors with logger, and returns a
// gRPC internal error to clients.
func LoggingStreamPanicHandler(logger log.FieldLogger) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer handleCrash(func(r interface{}) {
			werr := errors.Errorf("grpc stream server panic: %+v", r)
			logger.WithError(werr).Error("grpc stream server panic")
			err = toPanicError(werr)
		})
		return handler(srv, stream)
	}
}

func handleCrash(handler func(interface{})) {
	if r := recover(); r != nil {
		handler(r)
	}
}

func toPanicError(r interface{}) error {
	return grpc.Errorf(codes.Internal, "panic: %v", r)
}
