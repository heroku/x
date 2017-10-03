package grpcmetrics

import (
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/net/context"
	"google.golang.org/grpc/status"
)

// The helpers here exist to make friendly metric names for
// metric providers that don't support labeled metrics.

func metricPrefix(rpcType, fullMethod string) string {
	service, method := methodInfo(fullMethod)
	return fmt.Sprintf("grpc.%s.%s.%s", rpcType, service, method)
}

// methodInfo splits gRPC FullMethod names into service and method
// strings which are suitable for embedding in a metric name.
func methodInfo(fullMethod string) (string, string) {
	parts := strings.Split(fullMethod, "/")
	if len(parts) < 3 {
		return "unknown", "unknown"
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
		return "canceled"
	}

	st, ok := status.FromError(err)
	if !ok {
		return "unknown"
	}

	return dasherize(st.Code().String())
}

var uppers = regexp.MustCompile(`([[:lower:]])([[:upper:]])`)

func dasherize(s string) string {
	return strings.ToLower(uppers.ReplaceAllString(s, "$1-$2"))
}
