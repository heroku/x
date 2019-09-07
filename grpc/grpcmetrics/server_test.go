package grpcmetrics

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc"

	"github.com/heroku/x/go-kit/metrics/testmetrics"
	"github.com/heroku/x/go-kit/metricsregistry"
)

func TestUnaryServerInterceptor(t *testing.T) {
	p := testmetrics.NewProvider(t)
	usi := NewUnaryServerInterceptor(p)
	handler := func(resp interface{}, err error) grpc.UnaryHandler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			return resp, err
		}
	}
	info := &grpc.UnaryServerInfo{
		FullMethod: "/spec.Hello/Ping",
	}

	resp, err := usi(context.Background(), "ping", info, handler("pong", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp != "pong" {
		t.Fatalf("resp = %v, want %q", resp, "pong")
	}

	_, err = usi(context.Background(), "ping", info, handler(nil, errors.New("test")))
	if err == nil {
		t.Fatal("expected an error")
	}

	_, err = usi(context.Background(), "ping", info, handler(nil, context.Canceled))
	if err == nil {
		t.Fatal("expected an error")
	}

	p.CheckCounter("grpc.server.hello.ping.requests", 3)
	p.CheckCounter("grpc.server.hello.ping.response-codes.ok", 1)
	p.CheckCounter("grpc.server.hello.ping.response-codes.canceled", 1)
	p.CheckCounter("grpc.server.hello.ping.response-codes.unknown", 1)
	p.CheckCounter("grpc.server.hello.ping.errors", 1)
	p.CheckObservationCount("grpc.server.hello.ping.request-duration.ms", 3)
}

func TestStreamServerInterceptor(t *testing.T) {
	p := testmetrics.NewProvider(t)
	ssi := NewStreamServerInterceptor(p)
	handler := func(err error) grpc.StreamHandler {
		return func(srv interface{}, stream grpc.ServerStream) error {
			if err == nil {
				if err := stream.SendMsg("ping"); err != nil {
					t.Fatal("unexpected error", err)
				}
				if err := stream.RecvMsg("pong"); err != nil {
					t.Fatal("unexpected error", err)
				}
				if err := stream.SendMsg("ping"); err != nil {
					t.Fatal("unexpected error", err)
				}
			}
			return err
		}
	}
	info := &grpc.StreamServerInfo{
		FullMethod: "/spec.Hello/StreamUpdates",
	}

	err := ssi(nil, &testServerStream{}, info, handler(nil))
	if err != nil {
		t.Fatal(err)
	}

	err = ssi(nil, &testServerStream{}, info, func(srv interface{}, stream grpc.ServerStream) error {
		p.CheckGauge("grpc.server.hello.stream-updates.stream.clients", 1)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	err = ssi(nil, &testServerStream{}, info, handler(errors.New("test")))
	if err == nil {
		t.Fatal("expected an error")
	}

	p.CheckCounter("grpc.server.hello.stream-updates.requests", 3)
	p.CheckCounter("grpc.server.hello.stream-updates.response-codes.ok", 2)
	p.CheckCounter("grpc.server.hello.stream-updates.response-codes.unknown", 1)
	p.CheckCounter("grpc.server.hello.stream-updates.errors", 1)
	p.CheckObservationCount("grpc.server.hello.stream-updates.request-duration.ms", 3)

	p.CheckGauge("grpc.server.hello.stream-updates.stream.clients", 0)

	p.CheckCounter("grpc.server.hello.stream-updates.stream.sends", 2)
	p.CheckObservationCount("grpc.server.hello.stream-updates.stream.send-duration.ms", 2)

	p.CheckCounter("grpc.server.hello.stream-updates.stream.recvs", 1)
	p.CheckObservationCount("grpc.server.hello.stream-updates.stream.recv-duration.ms", 1)
}

func TestInstrumentMethod(t *testing.T) {
	p := testmetrics.NewProvider(t)
	r := metricsregistry.New(p)

	instrumentMethod(r, time.Millisecond, nil)
	instrumentMethod(r, time.Second, nil)
	instrumentMethod(r, 10*time.Second, errors.New(""))

	p.CheckCounter("requests", 3)
	p.CheckCounter("errors", 1)
	p.CheckCounter("response-codes.ok", 2)
	p.CheckCounter("response-codes.unknown", 1)
	p.CheckObservations("request-duration.ms", []float64{1.0, 1000.0, 10000.0})
}

func TestInstrumentStreamSend(t *testing.T) {
	p := testmetrics.NewProvider(t)
	r := metricsregistry.New(p)

	instrumentStreamSend(r, time.Millisecond, nil)
	instrumentStreamSend(r, time.Second, nil)
	instrumentStreamSend(r, 10*time.Second, errors.New("stream send error"))

	p.CheckCounter("stream.sends", 3)
	p.CheckObservations("stream.send-duration.ms", []float64{1.0, 1000.0, 10000.0})
	p.CheckCounter("stream.sends.errors", 1)
}

func TestInstrumentStreamRecv(t *testing.T) {
	p := testmetrics.NewProvider(t)
	r := metricsregistry.New(p)

	instrumentStreamRecv(r, time.Millisecond, nil)
	instrumentStreamRecv(r, time.Second, nil)
	instrumentStreamRecv(r, 10*time.Second, errors.New("stream recv error"))

	p.CheckCounter("stream.recvs", 3)
	p.CheckObservations("stream.recv-duration.ms", []float64{1.0, 1000.0, 10000.0})
	p.CheckCounter("stream.recvs.errors", 1)
}

type testServerStream struct {
	grpc.ServerStream
}

func (*testServerStream) SendMsg(m interface{}) error {
	return nil
}

func (*testServerStream) RecvMsg(m interface{}) error {
	return nil
}
