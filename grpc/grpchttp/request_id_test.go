package grpchttp

import (
	"context"
	"net/http"
	"testing"

	"google.golang.org/grpc/metadata"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"

	"github.com/heroku/x/grpc/requestid"
)

func TestRequestIDAnnotator(t *testing.T) {
	// assert RequestIDAnnotator is properly typed
	runtime.WithMetadata(RequestIDAnnotator)

	baseCtx := context.Background()

	req := &http.Request{
		Header: http.Header{},
	}
	ctx := metadata.NewIncomingContext(baseCtx, RequestIDAnnotator(baseCtx, req))
	if _, ok := requestid.FromContext(ctx); ok {
		t.Fatal("unexpected id for missing Request-Id")
	}

	requestID := "request-1"
	req.Header.Set("Request-Id", requestID)
	ctx = metadata.NewIncomingContext(baseCtx, RequestIDAnnotator(baseCtx, req))
	id, ok := requestid.FromContext(ctx)
	if !ok {
		t.Fatal("missing id for Request-Id")
	}
	if id != requestID {
		t.Fatalf("got %v, want %v", id, requestID)
	}
}
