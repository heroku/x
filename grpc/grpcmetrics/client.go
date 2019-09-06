package grpcmetrics

import (
	"context"
	"net"
	"time"

	"google.golang.org/grpc"

	"github.com/heroku/x/go-kit/metricsregistry"
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

// InstrumentedDialer returns an instrumented dialer for use with grpc.WithDialer,
// reporting dialing metrics using the given id, metricsNamespace, and registry.
func InstrumentedDialer(id, metricsNamespace string, r metricsregistry.Registry) func(string, time.Duration) (net.Conn, error) {
	r = metricsregistry.NewPrefixed(r, "grpc.client-dialer."+metricsNamespace+"."+id)

	return func(addr string, timeout time.Duration) (net.Conn, error) {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		d := &net.Dialer{}

		r.GetOrRegisterCounter("dials").Add(1)

		t0 := time.Now()
		conn, err := d.DialContext(ctx, "tcp", addr)
		r.GetOrRegisterHistogram("dial-duration.ms", 50).Observe(float64(time.Since(t0)) / float64(time.Millisecond))

		if err != nil {
			r.GetOrRegisterCounter("dial-errors").Add(1)
		}

		return conn, err
	}
}
