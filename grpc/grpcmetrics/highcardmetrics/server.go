package highcardmetrics

import (
	"context"
	"regexp"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/heroku/x/go-kit/metrics"
	"github.com/heroku/x/go-kit/metricsregistry"
	"github.com/heroku/x/grpc/grpcmetrics"
)

// NewUnaryServerInterceptor returns an interceptor for unary server calls
// which will report metrics to the given provider.
func NewUnaryServerInterceptor(p metrics.Provider) grpc.UnaryServerInterceptor {
	r0 := metricsregistry.New(p)
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ interface{}, err error) {
		r1 := metricsregistry.NewPrefixed(r0, "grpc.server")

		defer func(begin time.Time) {
			service, method := parseFullMethod(info.FullMethod)
			labels := []string{"service", service, "method", method, "response-status", code(err)}

			instrumentMethod(r1, labels, time.Since(begin))
		}(time.Now())

		return handler(ctx, req)
	}
}

// NewStreamServerInterceptor returns an interceptor for stream server calls
// which will report metrics to the given provider.
func NewStreamServerInterceptor(p metrics.Provider) grpc.StreamServerInterceptor {
	r0 := metricsregistry.New(p)
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		r1 := metricsregistry.NewPrefixed(r0, "grpc.server")

		service, method := parseFullMethod(info.FullMethod)

		labels := []string{"service", service, "method", method}

		clients := r1.GetOrRegisterGauge("stream.clients").With(labels...)
		clients.Add(1)

		defer func(begin time.Time) {
			clients.Add(-1)

			labels = append(labels, "response-status", code(err))

			instrumentMethod(r1, labels, time.Since(begin))
		}(time.Now())

		wrapped := &serverStream{r1, ss, labels}
		return handler(srv, wrapped)
	}
}

// serverStream provides a light wrapper over grpc.ServerStream
// to instrument SendMsg and RecvMsg.
type serverStream struct {
	reg metricsregistry.Registry
	grpc.ServerStream
	labels []string
}

//// RecvMsg implements the grpc.Stream interface.
func (ss *serverStream) SendMsg(m interface{}) (err error) {
	defer func(begin time.Time) {
		instrumentStreamSend(ss.reg, ss.labels, time.Since(begin), err)
	}(time.Now())

	return ss.ServerStream.SendMsg(m)
}

//// RecvMsg implements the grpc.Stream interface.
func (ss *serverStream) RecvMsg(m interface{}) (err error) {
	defer func(begin time.Time) {
		instrumentStreamRecv(ss.reg, ss.labels, time.Since(begin), err)
	}(time.Now())

	return ss.ServerStream.RecvMsg(m)
}

func instrumentMethod(r metricsregistry.Registry, labels []string, duration time.Duration) {
	r.GetOrRegisterHistogram("request-duration.ms", 50).With(labels...).Observe(ms(duration))
	r.GetOrRegisterCounter("requests").With(labels...).Add(1)
}

func instrumentStreamSend(r metricsregistry.Registry, labels []string, duration time.Duration, err error) {
	r.GetOrRegisterHistogram("stream.send-duration.ms", 50).With(labels...).Observe(ms(duration))
	r.GetOrRegisterCounter("stream.sends").With(labels...).Add(1)

	if err != nil && !isCanceled(err) {
		r.GetOrRegisterCounter("stream.sends.errors").Add(1)
	}
}

func instrumentStreamRecv(r metricsregistry.Registry, labels []string, duration time.Duration, err error) {
	r.GetOrRegisterHistogram("stream.recv-duration.ms", 50).With(labels...).Observe(ms(duration))
	r.GetOrRegisterCounter("stream.recvs").With(labels...).Add(1)

	if err != nil && !isCanceled(err) {
		r.GetOrRegisterCounter("stream.recvs.errors").With(labels...).Add(1)
	}
}

func parseFullMethod(fullMethod string) (string, string) {
	parts := strings.Split(fullMethod, "/")
	if len(parts) < 3 {
		return grpcmetrics.Unknown, grpcmetrics.Unknown
	}

	fullService := parts[1]
	method := parts[2]

	sp := strings.Split(fullService, ".")
	service := sp[len(sp)-1]

	return dasherize(service), dasherize(method)
}

// code returns the gRPC error code, handling context and unknown errors.
func code(err error) string {
	if err == context.Canceled {
		return grpcmetrics.Canceled
	}

	st, ok := status.FromError(err)
	if !ok {
		return grpcmetrics.Unknown
	}

	return dasherize(st.Code().String())
}

var uppers = regexp.MustCompile(`([[:lower:]])([[:upper:]])`)

func dasherize(s string) string {
	return strings.ToLower(uppers.ReplaceAllString(s, "$1-$2"))
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

func ms(d time.Duration) float64 {
	return float64(d.Milliseconds())
}
