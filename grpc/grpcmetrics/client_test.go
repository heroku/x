package grpcmetrics

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/heroku/x/go-kit/metrics/testmetrics"
	"github.com/heroku/x/go-kit/metricsregistry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestUnaryClientInterceptor(t *testing.T) {
	p := testmetrics.NewProvider(t)
	r := metricsregistry.New(p)
	uci := NewUnaryClientInterceptor(r)
	invoker := func(err error) grpc.UnaryInvoker {
		return func(_ context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			return err
		}
	}

	uci(context.Background(), "/spec.Hello/Ping", nil, nil, nil, invoker(nil))
	uci(context.Background(), "/spec.Hello/Ping", nil, nil, nil, invoker(status.Error(codes.Canceled, "canceled")))
	uci(context.Background(), "/spec.Hello/Ping", nil, nil, nil, invoker(errors.New("test")))

	p.CheckCounter("grpc.client.hello.ping.requests", 3)
	p.CheckCounter("grpc.client.hello.ping.response-codes.ok", 1)
	p.CheckCounter("grpc.client.hello.ping.response-codes.canceled", 1)
	p.CheckCounter("grpc.client.hello.ping.response-codes.unknown", 1)
	p.CheckCounter("grpc.client.hello.ping.errors", 1)
	p.CheckObservationCount("grpc.client.hello.ping.request-duration.ms", 3)
}

func TestStreamClientInterceptor(t *testing.T) {
	p := testmetrics.NewProvider(t)
	r := metricsregistry.New(p)
	sci := NewStreamClientInterceptor(r)
	streamer := func(err, clientErr error) grpc.Streamer {
		return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
			return &testClientStream{
				Error: clientErr,
			}, err
		}
	}

	sci(context.Background(), nil, nil, "/spec.Hello/ClientStream", streamer(errors.New("client request error"), nil))
	sci(context.Background(), nil, nil, "/spec.Hello/ClientStream", streamer(status.Error(codes.Canceled, "canceled"), nil))

	cs, _ := sci(context.Background(), nil, nil, "/spec.Hello/ClientStream", streamer(nil, nil))
	cs.RecvMsg("test")
	cs.SendMsg("test")

	cs, _ = sci(context.Background(), nil, nil, "/spec.Hello/ClientStream", streamer(nil, errors.New("client stream error")))
	cs.RecvMsg("test")
	cs.SendMsg("test")

	p.CheckCounter("grpc.client.hello.client-stream.requests", 4)
	p.CheckCounter("grpc.client.hello.client-stream.errors", 1)
	p.CheckObservationCount("grpc.client.hello.client-stream.request-duration.ms", 4)

	p.CheckCounter("grpc.client.hello.client-stream.response-codes.ok", 2)
	p.CheckCounter("grpc.client.hello.client-stream.response-codes.canceled", 1)
	p.CheckCounter("grpc.client.hello.client-stream.response-codes.unknown", 1)

	p.CheckCounter("grpc.client.hello.client-stream.stream.sends", 2)
	p.CheckObservationCount("grpc.client.hello.client-stream.stream.send-duration.ms", 2)
	p.CheckCounter("grpc.client.hello.client-stream.stream.sends.errors", 1)

	p.CheckCounter("grpc.client.hello.client-stream.stream.recvs", 2)
	p.CheckObservationCount("grpc.client.hello.client-stream.stream.recv-duration.ms", 2)
	p.CheckCounter("grpc.client.hello.client-stream.stream.recvs.errors", 1)
}

func TestInstrumentedDialer(t *testing.T) {
	p := testmetrics.NewProvider(t)
	r := metricsregistry.New(p)
	d := InstrumentedDialer("st01", "foo-bars", r)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)

		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			c.Close()
		}
	}()

	c, err := d(ln.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	c.Close()

	p.CheckCounter("grpc.client-dialer.foo-bars.st01.dials", 1)
	p.CheckObservationCount("grpc.client-dialer.foo-bars.st01.dial-duration.ms", 1)

	ln.Close()
	<-done

	_, err = d(ln.Addr().String(), time.Second)
	if err == nil {
		t.Fatal("wanted error dialing to closed listener")
	}

	p.CheckCounter("grpc.client-dialer.foo-bars.st01.dials", 2)
	p.CheckCounter("grpc.client-dialer.foo-bars.st01.dial-errors", 1)
	p.CheckObservationCount("grpc.client-dialer.foo-bars.st01.dial-duration.ms", 2)
}

type testClientStream struct {
	grpc.ClientStream
	Error error
}

func (tcs *testClientStream) SendMsg(m interface{}) error {
	return tcs.Error
}

func (tcs *testClientStream) RecvMsg(m interface{}) error {
	return tcs.Error
}
