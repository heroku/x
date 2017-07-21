package grpcmetrics

import (
	"errors"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func TestCode(t *testing.T) {
	for i, tt := range []struct {
		err  error
		want string
	}{
		{nil, "ok"},
		{errors.New("custom"), "unknown"},
		{grpc.Errorf(codes.InvalidArgument, ""), "invalid-argument"},
		{grpc.Errorf(codes.NotFound, "release not found"), "not-found"},
		{Ignore(grpc.Errorf(codes.NotFound, "release not found")), "not-found"},
	} {
		got := code(tt.err)
		if got != tt.want {
			t.Errorf("%d. code(%+v) = %q, want %q", i, tt.err, got, tt.want)
		}
	}
}

func TestMethodInfo(t *testing.T) {
	for i, tt := range []struct {
		fullMethod string
		service    string
		method     string
	}{
		{"/spec.DomainStreamer/StreamUpdates", "domain-streamer", "stream-updates"},
		{"/Store/Put", "store", "put"},
		{"/x.y.z.Store/Put", "store", "put"},
		{"other", "unknown", "unknown"},
	} {
		service, method := methodInfo(tt.fullMethod)
		if service != tt.service || method != tt.method {
			t.Errorf(
				"%d. methodInfo(%q) = (%q, %q), want (%q, %q)",
				i,
				tt.fullMethod,
				service,
				method,
				tt.service,
				tt.method,
			)
		}
	}
}
