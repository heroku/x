package grpcserver

import (
	"context"
	"fmt"

	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"google.golang.org/grpc"
)

// UnaryPayloadLoggingTagger annotates ctx with grpc_ctxtags tags for request and
// response payloads.
//
// A loggable request or response implements this interface
//
//		type loggable interface {
//			LoggingTags() map[string]interface{}
//		}
//
// Any request or response implementing this interface will add tags to the
// context for logging in success and error cases.
func UnaryPayloadLoggingTagger(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ interface{}, err error) {
	tag(ctx, "request", req)

	resp, err := handler(ctx, req)
	if err == nil {
		tag(ctx, "response", resp)
	}

	return resp, err
}

type loggable interface {
	LoggingTags() map[string]interface{}
}

func tag(ctx context.Context, scope string, pb interface{}) {
	tags := grpc_ctxtags.Extract(ctx)
	extractTags(tags, scope, pb)
}

func extractTags(tags grpc_ctxtags.Tags, scope string, pb interface{}) {
	if lg, ok := pb.(loggable); ok {
		for k, v := range lg.LoggingTags() {
			name := fmt.Sprintf("%s.%s", scope, k)
			if _, ok := v.(loggable); ok {
				extractTags(tags, name, v)
			} else {
				tags.Set(name, v)
			}
		}
	}
}
