package grpcmetrics

import (
	"time"

	"github.com/heroku/cedar/lib/kit/metricsregistry"
	"github.com/heroku/x/go-kit/metrics"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// NewUnaryClientInterceptor returns an interceptor for unary client calls
// which will report metrics to the given provider.
func NewUnaryClientInterceptor(p metrics.Provider) grpc.UnaryClientInterceptor {
	r0 := metricsregistry.New(p)

	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) (err error) {
		r1 := metricsregistry.NewPrefixed(r0, metricPrefix("client", method))

		defer func(begin time.Time) {
			instrumentMethod(r1, time.Since(begin), err)
		}(time.Now())

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
