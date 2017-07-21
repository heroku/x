package grpcmetrics

import (
	"testing"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func TestIgnorableError(t *testing.T) {
	err := grpc.Errorf(codes.NotFound, "release not found")
	ignore := Ignore(err)

	if cerr := errors.Cause(ignore); cerr != err {
		t.Fatalf("Cause(err) failed: got %#v, wanted %#v", cerr, err)
	}

	if !Ignorable(ignore) {
		t.Error("Ignorable(err) failed wanted true")
	}
}

func TestNonIgnorableError(t *testing.T) {
	err := grpc.Errorf(codes.NotFound, "release not found")

	if Ignorable(err) {
		t.Error("Ignorable(err) failed wanted false")
	}
}
