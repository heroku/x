package panichandler

import (
	"context"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

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
	//TODO:  SA1019: grpc.Errorf is deprecated: use status.Errorf instead.  (staticcheck)
	return grpc.Errorf(codes.Internal, "panic: %v", r) //nolint:staticcheck
}
