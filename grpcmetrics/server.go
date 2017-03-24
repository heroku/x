package grpcmetrics

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// NewUnaryServerInterceptor returns an interceptor for unary server calls
// which will report metrics to the given provider.
func NewUnaryServerInterceptor(p Provider) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ interface{}, err error) {
		pp := &prefixProvider{metricPrefix("server", info.FullMethod), p}

		defer func(begin time.Time) {
			instrumentMethod(pp, time.Since(begin), err)
		}(time.Now())

		return handler(ctx, req)
	}
}

// NewStreamServerInterceptor returns an interceptor for stream server calls
// which will report metrics to the given provider.
func NewStreamServerInterceptor(p Provider) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		pp := &prefixProvider{metricPrefix("server", info.FullMethod), p}

		defer func(begin time.Time) {
			instrumentMethod(pp, time.Since(begin), err)
		}(time.Now())

		wrapped := &serverStream{pp, ss}
		return handler(srv, wrapped)
	}
}

// serverStream provides a light wrapper over grpc.ServerStream
// to instrument SendMsg and RecvMsg.
type serverStream struct {
	p Provider
	grpc.ServerStream
}

// RecvMsg implements the grpc.Stream interface.
func (ss *serverStream) SendMsg(m interface{}) error {
	defer func(begin time.Time) {
		instrumentStreamSend(ss.p, time.Since(begin))
	}(time.Now())

	return ss.ServerStream.SendMsg(m)
}

// RecvMsg implements the grpc.Stream interface.
func (ss *serverStream) RecvMsg(m interface{}) error {
	defer func(begin time.Time) {
		instrumentStreamRecv(ss.p, time.Since(begin))
	}(time.Now())

	return ss.ServerStream.RecvMsg(m)
}

func instrumentMethod(p Provider, duration time.Duration, err error) {
	p.NewHistogram("request-duration.ms", 50).Observe(ms(duration))
	p.NewCounter("requests").Add(1)
	p.NewCounter(fmt.Sprintf("response-codes.%s", code(err))).Add(1)
	if err != nil {
		if errors.Cause(err) == context.Canceled {
			// Count cancelations differently from other errors to avoid
			// introducing too much noise into the error count.
			p.NewCounter("context-canceled-errors").Add(1)
		} else {
			p.NewCounter("errors").Add(1)
		}
	}
}

func instrumentStreamSend(p Provider, duration time.Duration) {
	p.NewHistogram("stream.send-duration.ms", 50).Observe(ms(duration))
	p.NewCounter("stream.sends").Add(1)
}

func instrumentStreamRecv(p Provider, duration time.Duration) {
	p.NewHistogram("stream.recv-duration.ms", 50).Observe(ms(duration))
	p.NewCounter("stream.recvs").Add(1)
}

func ms(d time.Duration) float64 {
	return float64(d) / float64(time.Millisecond)
}
