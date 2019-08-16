package requestid

import (
	"context"

	"google.golang.org/grpc/metadata"
)

const (
	metadataKey = "x-request-id"
)

// FromContext returns a request ID from gRPC metadata if available in ctx.
func FromContext(ctx context.Context) (string, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", false
	}

	ids, ok := md[metadataKey]
	if !ok || len(ids) == 0 {
		return "", false
	}

	return ids[0], true
}

// NewMetadata constructs gRPC metadata with the request ID set.
func NewMetadata(id string) metadata.MD {
	return metadata.Pairs(metadataKey, id)
}

// AppendToOutgoingContext returns a context with the request-id added to the gRPC metadata.
func AppendToOutgoingContext(ctx context.Context) context.Context {
	id, ok := FromContext(ctx)
	if !ok {
		return ctx
	}

	return metadata.AppendToOutgoingContext(ctx, metadataKey, id)
}
