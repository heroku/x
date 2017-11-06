package grpcmetrics

import (
	"context"
	"time"

	"github.com/heroku/cedar/lib/kit/metricsregistry"
	"google.golang.org/grpc"
)

// NewUnaryClientInterceptor returns an interceptor for unary client calls
// which will report metrics using the given registry.
func NewUnaryClientInterceptor(r metricsregistry.Registry) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) (err error) {
		r1 := metricsregistry.NewPrefixed(r, metricPrefix("client", method))

		defer func(begin time.Time) {
			instrumentMethod(r1, time.Since(begin), err)
		}(time.Now())

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// NewStreamClientInterceptor returns an interceptor for stream client calls
// which will report metrics using the given registry.
func NewStreamClientInterceptor(r metricsregistry.Registry) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (cs grpc.ClientStream, err error) {
		r1 := metricsregistry.NewPrefixed(r, metricPrefix("client", method))

		defer func(begin time.Time) {
			instrumentMethod(r1, time.Since(begin), err)
		}(time.Now())

		cs, err = streamer(ctx, desc, cc, method, opts...)
		return &clientStream{r1, cs}, err
	}
}

// clientStream provides a light wrapper over grpc.ClientStream
// to instrument SendMsg and RecvMsg.
type clientStream struct {
	reg metricsregistry.Registry
	grpc.ClientStream
}

// SendMsg implements the grpc.ClientStream interface.
func (cs *clientStream) SendMsg(m interface{}) (err error) {
	defer func(begin time.Time) {
		instrumentStreamSend(cs.reg, time.Since(begin), err)
	}(time.Now())

	return cs.ClientStream.SendMsg(m)
}

// RecvMsg implements the grpc.ClientStream interface.
func (cs *clientStream) RecvMsg(m interface{}) (err error) {
	defer func(begin time.Time) {
		instrumentStreamRecv(cs.reg, time.Since(begin), err)
	}(time.Now())

	return cs.ClientStream.RecvMsg(m)
}
