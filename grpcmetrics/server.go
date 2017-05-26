package grpcmetrics

import (
	"fmt"
	"time"

	"github.com/heroku/x/go-kit/metrics"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// NewUnaryServerInterceptor returns an interceptor for unary server calls
// which will report metrics to the given provider.
func NewUnaryServerInterceptor(p metrics.Provider) grpc.UnaryServerInterceptor {
	r0 := newRegistry(p)
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ interface{}, err error) {
		r1 := &prefixedRegistry{r0, metricPrefix("server", info.FullMethod)}

		defer func(begin time.Time) {
			instrumentMethod(r1, time.Since(begin), err)
		}(time.Now())

		return handler(ctx, req)
	}
}

// NewStreamServerInterceptor returns an interceptor for stream server calls
// which will report metrics to the given provider.
func NewStreamServerInterceptor(p metrics.Provider) grpc.StreamServerInterceptor {
	reg := newRegistry(p)
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		reg := &prefixedRegistry{reg, metricPrefix("server", info.FullMethod)}

		clients := reg.GetOrRegisterGauge("stream.clients")
		clients.Add(1)

		defer func(begin time.Time) {
			clients.Add(-1)
			instrumentMethod(reg, time.Since(begin), err)
		}(time.Now())

		wrapped := &serverStream{reg, ss}
		return handler(srv, wrapped)
	}
}

// serverStream provides a light wrapper over grpc.ServerStream
// to instrument SendMsg and RecvMsg.
type serverStream struct {
	reg registry
	grpc.ServerStream
}

// RecvMsg implements the grpc.Stream interface.
func (ss *serverStream) SendMsg(m interface{}) error {
	defer func(begin time.Time) {
		instrumentStreamSend(ss.reg, time.Since(begin))
	}(time.Now())

	return ss.ServerStream.SendMsg(m)
}

// RecvMsg implements the grpc.Stream interface.
func (ss *serverStream) RecvMsg(m interface{}) error {
	defer func(begin time.Time) {
		instrumentStreamRecv(ss.reg, time.Since(begin))
	}(time.Now())

	return ss.ServerStream.RecvMsg(m)
}

func instrumentMethod(r registry, duration time.Duration, err error) {
	r.GetOrRegisterHistogram("request-duration.ms", 50).Observe(ms(duration))
	r.GetOrRegisterCounter("requests").Add(1)
	r.GetOrRegisterCounter(fmt.Sprintf("response-codes.%s", code(err))).Add(1)
	if err != nil {
		if errors.Cause(err) == context.Canceled {
			// Count cancelations differently from other errors to avoid
			// introducing too much noise into the error count.
			r.GetOrRegisterCounter("context-canceled-errors").Add(1)
		} else {
			r.GetOrRegisterCounter("errors").Add(1)
		}
	}
}

func instrumentStreamSend(r registry, duration time.Duration) {
	r.GetOrRegisterHistogram("stream.send-duration.ms", 50).Observe(ms(duration))
	r.GetOrRegisterCounter("stream.sends").Add(1)
}

func instrumentStreamRecv(r registry, duration time.Duration) {
	r.GetOrRegisterHistogram("stream.recv-duration.ms", 50).Observe(ms(duration))
	r.GetOrRegisterCounter("stream.recvs").Add(1)
}

func ms(d time.Duration) float64 {
	return float64(d) / float64(time.Millisecond)
}
