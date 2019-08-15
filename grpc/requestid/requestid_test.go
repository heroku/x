package requestid

import (
	"context"
	"testing"

	"google.golang.org/grpc/metadata"
)

func TestFromContext(t *testing.T) {
	requestID := "request-1"
	ctx := metadata.NewIncomingContext(context.Background(), NewMetadata(requestID))
	id, ok := FromContext(ctx)
	if !ok {
		t.Fatal("no id for annotated context")
	}
	if id != requestID {
		t.Fatalf("id = %v, want %v", id, requestID)
	}
}

func TestFromContext_NoMetadata(t *testing.T) {
	if _, ok := FromContext(context.Background()); ok {
		t.Fatalf("got id from empty context")
	}
}

func TestFromContext_UnrelatedMetadata(t *testing.T) {
	md := metadata.Pairs("key", "val")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	if _, ok := FromContext(ctx); ok {
		t.Fatalf("got id from empty context")
	}
}

func TestFromContext_InvalidMetadata(t *testing.T) {
	md := metadata.MD{metadataKey: []string{}}
	ctx := metadata.NewIncomingContext(context.Background(), md)

	if _, ok := FromContext(ctx); ok {
		t.Fatalf("got id from invalid context")
	}
}

func TestAppendToOutgoingContext(t *testing.T) {
	requestID := "request-1"
	ctx := metadata.NewIncomingContext(context.Background(), NewMetadata(requestID))
	ctx = AppendToOutgoingContext(ctx)
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		t.Fatal("no id for annotated context")
	}

	id := md[metadataKey][0]
	if id != requestID {
		t.Fatalf("got request-id %q; wanted %q", id, requestID)
	}
}
