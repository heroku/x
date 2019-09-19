package grpcserver

import (
	"context"
	"sync"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	RequestLimit = stats.Int64(
		"github.com/heroku/x/grpc/grpcserver/concurrency_limit",
		"Current limit of concurrent requests",
		stats.UnitDimensionless,
	)
	InflightRequests = stats.Int64(
		"github.com/heroku/x/grpc/grpcserver/inflight_requests",
		"Number of in-flight requests",
		stats.UnitDimensionless,
	)
	RejectedRequests = stats.Int64(
		"github.com/heroku/x/grpc/grpcserver/rejected_requests",
		"Number of requests rejected because of concurrency limits",
		stats.UnitDimensionless,
	)
)

var (
	RequestLimitLastValue = &view.View{
		Name:        "github.com/heroku/x/grpc/grpcserver/concurrency_limit",
		Measure:     RequestLimit,
		Aggregation: view.LastValue(),
		Description: "Concurrency limit",
	}
	RejectedRequestsCount = &view.View{
		Name:        "github.com/heroku/x/grpc/grpcserver/rejected_requests",
		Measure:     RejectedRequests,
		Aggregation: view.Sum(),
		Description: "Rejected requests",
	}
	InflightRequestsLastValue = &view.View{
		Name:        "github.com/heroku/x/grpc/grpcserver/inflight_requests",
		Measure:     InflightRequests,
		Aggregation: view.LastValue(),
		Description: "Inflight requests",
	}
)

var (
	DefaultLimiterViews = []*view.View{
		RequestLimitLastValue,
		RejectedRequestsCount,
		InflightRequestsLastValue,
	}
)

// Limiter implements an adaptive concurrency limiter for unary gRPC server
// handlers. The actual limit will be discovered over time via AIMD (Additive
// Increase, Multiplicative Decrease).
func Limiter(initial int64, backoffRatio float64) grpc.UnaryServerInterceptor {
	l := &limiter{
		backoffRatio: backoffRatio,
		limit:        initial,
	}

	return func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ interface{}, err error) {
		tx, ok := l.start()
		if !ok {
			stats.Record(ctx, RejectedRequests.M(1))
			return nil, status.Error(codes.Unavailable, "server concurrency limit reached")
		}

		stats.Record(ctx, RequestLimit.M(tx.limit), InflightRequests.M(tx.inflight))

		defer l.finish(ctx, tx, err)

		return handler(ctx, req)
	}
}

type limiter struct {
	backoffRatio float64

	sync.Mutex
	inflight int64
	limit    int64
}

func (l *limiter) start() (*transaction, bool) {
	l.Lock()
	defer l.Unlock()

	if l.inflight > l.limit {
		return nil, false
	}

	l.inflight++

	return &transaction{inflight: l.inflight, limit: l.limit}, true
}

func (l *limiter) finish(ctx context.Context, tx *transaction, err error) {
	l.Lock()
	defer l.Unlock()

	switch status.Code(err) {
	case codes.Canceled, codes.DeadlineExceeded, codes.Unavailable:
		// Multiplicative decrease, unless the limit has already been decreased
		// since the transaction started. But is that right?
		if l.limit >= tx.limit {
			lim := int64(float64(l.limit) * l.backoffRatio)
			if lim < 1 {
				lim = 1
			}
			l.limit = lim
		}

	default:
		// TODO: allow limit increases if the limit has gone down since the tx
		// started?
		if l.inflight > l.limit/2 { // or should it be >= ?, and why is it / 2?
			l.limit++
		}
	}

	l.inflight--

	stats.Record(ctx, RequestLimit.M(tx.limit), InflightRequests.M(tx.inflight))
}

type transaction struct {
	inflight int64
	limit    int64
}
