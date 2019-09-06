package grpcmetrics

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/heroku/x/go-kit/metrics"
	"github.com/heroku/x/go-kit/metricsregistry"
)

// NewUnaryServerInterceptor returns an interceptor for unary server calls
// which will report metrics to the given provider.
func NewUnaryServerInterceptor(p metrics.Provider) grpc.UnaryServerInterceptor {
	r0 := metricsregistry.New(p)
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ interface{}, err error) {
		r1 := metricsregistry.NewPrefixed(r0, metricPrefix("server", info.FullMethod))

		defer func(begin time.Time) {
			instrumentMethod(r1, time.Since(begin), err)
		}(time.Now())

		return handler(ctx, req)
	}
}

// NewStreamServerInterceptor returns an interceptor for stream server calls
// which will report metrics to the given provider.
func NewStreamServerInterceptor(p metrics.Provider) grpc.StreamServerInterceptor {
	reg := metricsregistry.New(p)
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		reg := metricsregistry.NewPrefixed(reg, metricPrefix("server", info.FullMethod))

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
	reg metricsregistry.Registry
	grpc.ServerStream
}

// RecvMsg implements the grpc.Stream interface.
func (ss *serverStream) SendMsg(m interface{}) (err error) {
	defer func(begin time.Time) {
		instrumentStreamSend(ss.reg, time.Since(begin), err)
	}(time.Now())

	return ss.ServerStream.SendMsg(m)
}

// RecvMsg implements the grpc.Stream interface.
func (ss *serverStream) RecvMsg(m interface{}) (err error) {
	defer func(begin time.Time) {
		instrumentStreamRecv(ss.reg, time.Since(begin), err)
	}(time.Now())

	return ss.ServerStream.RecvMsg(m)
}

func instrumentMethod(r metricsregistry.Registry, duration time.Duration, err error) {
	r.GetOrRegisterHistogram("request-duration.ms", 50).Observe(ms(duration))
	r.GetOrRegisterCounter("requests").Add(1)
	r.GetOrRegisterCounter(fmt.Sprintf("response-codes.%s", code(err))).Add(1)

	if err != nil && !isCanceled(err) {
		r.GetOrRegisterCounter("errors").Add(1)
	}
}

// isCanceled returns true if error is a context or gRPC cancelation error.
func isCanceled(err error) bool {
	if err == context.Canceled {
		return true
	}

	if st, ok := status.FromError(err); ok {
		return st.Code() == codes.Canceled
	}

	return false
}

func instrumentStreamSend(r metricsregistry.Registry, duration time.Duration, err error) {
	r.GetOrRegisterHistogram("stream.send-duration.ms", 50).Observe(ms(duration))
	r.GetOrRegisterCounter("stream.sends").Add(1)

	if err != nil && !isCanceled(err) {
		r.GetOrRegisterCounter("stream.sends.errors").Add(1)
	}
}

func instrumentStreamRecv(r metricsregistry.Registry, duration time.Duration, err error) {
	r.GetOrRegisterHistogram("stream.recv-duration.ms", 50).Observe(ms(duration))
	r.GetOrRegisterCounter("stream.recvs").Add(1)

	if err != nil && !isCanceled(err) {
		r.GetOrRegisterCounter("stream.recvs.errors").Add(1)
	}
}

func ms(d time.Duration) float64 {
	return float64(d) / float64(time.Millisecond)
}
