package panichandler

import (
	"context"
	"errors"
	"testing"

	"github.com/heroku/x/testing/testlog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestLoggingUnaryPanicHandler_NoPanic(t *testing.T) {
	l, hook := testlog.NewNullLogger()

	var (
		uhCalled bool
		res      = 1
		testErr  = errors.New("test error")
	)

	uh := func(ctx context.Context, req interface{}) (interface{}, error) {
		uhCalled = true
		return res, testErr
	}

	ph := LoggingUnaryPanicHandler(l)
	gres, gerr := ph(context.Background(), nil, nil, uh)

	if !uhCalled {
		t.Fatal("uh not called")
	}

	if gres != res {
		t.Fatalf("got res %+v, want %+v", gres, res)
	}

	if gerr != testErr {
		t.Fatalf("got err %+v, want %+v", gerr, testErr)
	}

	if got := len(hook.Entries()); got != 0 {
		t.Fatalf("got log entries %+v, wanted nothing logged", got)
	}
}

func TestLoggingUnaryPanicHandler_Panic(t *testing.T) {
	l, hook := testlog.NewNullLogger()

	var (
		uhCalled bool
		res      = 1
		testErr  = errors.New("test error")
	)

	uh := func(ctx context.Context, req interface{}) (interface{}, error) {
		uhCalled = true
		var x *int
		// the common nil pointer deref case
		_ = *x + 1
		return res, testErr
	}

	ph := LoggingUnaryPanicHandler(l)
	_, gerr := ph(context.Background(), nil, nil, uh)

	if !uhCalled {
		t.Fatal("unary handler not called")
	}

	st, ok := status.FromError(gerr)
	if !ok || st.Code() != codes.Internal {
		t.Fatalf("Got %+v want Internal grpc error", gerr)
	}

	hook.CheckAllContained(t, "grpc unary server panic")
}

func TestLoggingStreamPanicHandler_NoPanic(t *testing.T) {
	l, hook := testlog.NewNullLogger()

	var (
		shCalled bool
		testErr  = errors.New("test error")
	)

	sh := func(srv interface{}, stream grpc.ServerStream) error {
		shCalled = true
		return testErr
	}

	ph := LoggingStreamPanicHandler(l)
	gerr := ph(context.Background(), nil, nil, sh)

	if !shCalled {
		t.Fatal("stream handler not called")
	}

	if gerr != testErr {
		t.Fatalf("got err %+v, want %+v", gerr, testErr)
	}

	if got := len(hook.Entries()); got != 0 {
		t.Fatalf("got log entries %+v, wanted nothing logged", got)
	}
}

func TestLoggingStreamPanicHandler_Panic(t *testing.T) {
	l, hook := testlog.NewNullLogger()

	var (
		shCalled bool
		testErr  = errors.New("test error")
	)

	sh := func(srv interface{}, stream grpc.ServerStream) error {
		shCalled = true
		var x *int
		// the common nil pointer deref case
		_ = *x + 1
		return testErr
	}

	ph := LoggingStreamPanicHandler(l)
	gerr := ph(context.Background(), nil, nil, sh)

	if !shCalled {
		t.Fatal("stream handler not called")
	}

	st, ok := status.FromError(gerr)
	if !ok || st.Code() != codes.Internal {
		t.Fatalf("Got %+v want Internal grpc error", gerr)
	}

	hook.CheckAllContained(t, "grpc stream server panic")
}
