package panichandler

import (
	"context"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// LoggingUnaryPanicHandler returns a server interceptor which recovers
// panics, logs them as errors with logger, and returns a gRPC internal
// error to clients.
func LoggingUnaryPanicHandler(logger log.FieldLogger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
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
	return func(srv interface{}, stream grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer handleCrash(func(r interface{}) {
			werr := errors.Errorf("grpc stream server panic: %v", r)
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
	return status.Errorf(codes.Internal, "panic: %v", r)
}
