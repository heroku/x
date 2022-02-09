package highcardmetrics

import (
	"context"
	"errors"
	"testing"

	"github.com/heroku/x/go-kit/metrics/testmetrics"
	"google.golang.org/grpc"
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

	p.CheckCounter("grpc.server.requests", 1, "service", "hello", "method", "ping", "response-status", "ok")
	p.CheckCounter("grpc.server.requests", 1, "service", "hello", "method", "ping", "response-status", "canceled")
	p.CheckCounter("grpc.server.requests", 1, "service", "hello", "method", "ping", "response-status", "unknown")

	p.CheckObservationCount("grpc.server.request-duration.ms", 1, "service", "hello", "method", "ping", "response-status", "ok")
	p.CheckObservationCount("grpc.server.request-duration.ms", 1, "service", "hello", "method", "ping", "response-status", "canceled")
	p.CheckObservationCount("grpc.server.request-duration.ms", 1, "service", "hello", "method", "ping", "response-status", "unknown")
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
		p.CheckGauge("grpc.server.stream.clients", 1, "service", "hello", "method", "stream-updates")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	err = ssi(nil, &testServerStream{}, info, handler(errors.New("test")))
	if err == nil {
		t.Fatal("expected an error")
	}

	p.CheckCounter("grpc.server.requests", 2, "service", "hello", "method", "stream-updates", "response-status", "ok")
	p.CheckCounter("grpc.server.requests", 1, "service", "hello", "method", "stream-updates", "response-status", "unknown")
	p.CheckObservationCount("grpc.server.request-duration.ms", 2, "service", "hello", "method", "stream-updates", "response-status", "ok")
	p.CheckObservationCount("grpc.server.request-duration.ms", 1, "service", "hello", "method", "stream-updates", "response-status", "unknown")

	p.CheckGauge("grpc.server.stream.clients", 0, "service", "hello", "method", "stream-updates")

	p.CheckCounter("grpc.server.stream.sends", 2, "service", "hello", "method", "stream-updates")
	p.CheckObservationCount("grpc.server.stream.send-duration.ms", 2, "service", "hello", "method", "stream-updates")

	p.CheckCounter("grpc.server.stream.recvs", 1, "service", "hello", "method", "stream-updates")
	p.CheckObservationCount("grpc.server.stream.recv-duration.ms", 1, "service", "hello", "method", "stream-updates")
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
