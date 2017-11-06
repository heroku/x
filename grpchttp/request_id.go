package grpchttp

import (
	"context"
	"net/http"

	"github.com/heroku/cedar/lib/grpc/requestid"
	"google.golang.org/grpc/metadata"
)

// RequestIDAnnotator returns gRPC metadata with the Request-Id header if
// present. The request ID can be later retrieved by requestid.FromContext.
func RequestIDAnnotator(ctx context.Context, r *http.Request) metadata.MD {
	if id := r.Header.Get("Request-Id"); id != "" {
		return requestid.NewMetadata(id)
	}

	return nil
}
