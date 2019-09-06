package grpchttp

import (
	"context"
	"net/http"

	"google.golang.org/grpc/metadata"

	"github.com/heroku/x/grpc/requestid"
	httprequestid "github.com/heroku/x/requestid"
)

// RequestIDAnnotator returns gRPC metadata with the Request-Id header if
// present. The request ID can be later retrieved by requestid.FromContext.
func RequestIDAnnotator(ctx context.Context, r *http.Request) metadata.MD {
	if id := httprequestid.Get(r); id != "" {
		return requestid.NewMetadata(id)
	}

	return nil
}
