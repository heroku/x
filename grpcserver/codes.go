package grpcserver

import (
	"context"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// ErrorToCode determines the gRPC error code for an error, accounting for
// context errors and errors wrapped with pkg/errors.
//
// ErrorToCode implements grpc_logging.ErrorToCode.
func ErrorToCode(err error) codes.Code {
	err = errors.Cause(err)

	switch err {
	case context.Canceled:
		return codes.Canceled
	case context.DeadlineExceeded:
		return codes.DeadlineExceeded
	default:
		return grpc.Code(err)
	}
}
