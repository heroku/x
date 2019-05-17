package grpcserver

import (
	"context"
	"testing"

	"github.com/pkg/errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestErrorToCode_Unknown(t *testing.T) {
	code := ErrorToCode(errors.New("other"))
	if code != codes.Unknown {
		t.Fatalf("code = %v, want %v", code, codes.Unknown)
	}
}

func TestErrorToCode_GRPC(t *testing.T) {
	err := status.Errorf(codes.NotFound, "not found")
	code := ErrorToCode(err)
	if code != codes.NotFound {
		t.Fatalf("code = %v, want %v", code, codes.NotFound)
	}
}

func TestErrorToCode_Wrapped(t *testing.T) {
	err := status.Errorf(codes.NotFound, "not found")
	code := ErrorToCode(errors.WithStack(err))
	if code != codes.NotFound {
		t.Fatalf("code = %v, want %v", code, codes.NotFound)
	}
}

func TestErrorToCode_Canceled(t *testing.T) {
	code := ErrorToCode(context.Canceled)
	if code != codes.Canceled {
		t.Fatalf("code = %v, want %v", code, codes.Canceled)
	}
}

func TestErrorToCode_DeadlineExceeded(t *testing.T) {
	code := ErrorToCode(context.DeadlineExceeded)
	if code != codes.DeadlineExceeded {
		t.Fatalf("code = %v, want %v", code, codes.DeadlineExceeded)
	}
}
