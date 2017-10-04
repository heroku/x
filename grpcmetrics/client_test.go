package grpcmetrics

import (
	"errors"
	"testing"

	"github.com/heroku/cedar/lib/kit/metrics/testmetrics"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestUnaryClientInterceptor(t *testing.T) {
	p := testmetrics.NewProvider(t)
	uci := NewUnaryClientInterceptor(p)
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
