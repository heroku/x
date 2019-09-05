package rollbar

import (
	"context"
	"errors"
	"io"
	"net"
	"net/url"
	"testing"

	"github.com/sirupsen/logrus/hooks/test"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestShouldIgnore(t *testing.T) {
	cases := []struct {
		name         string
		err          error
		shouldIgnore bool
	}{
		{"url.Error with timeout=true", &url.Error{Err: &timeoutError{true}}, true},
		{"url.Error with temporary", &url.Error{Err: &tempError{true}}, true},
		{"url.Error with context.Canceled", &url.Error{Err: context.Canceled}, true},
		{"url.Error with random err", &url.Error{Err: errors.New("random")}, false},
		{"timeout=true", timeoutError{true}, true},
		{"timeout=false", timeoutError{false}, false},
		{"temporary=true", tempError{true}, true},
		{"temporary=false", tempError{false}, false},
		{"DeadlineExceeded", context.DeadlineExceeded, true},
		{"Canceled", context.Canceled, true},
		{"grpc Canceled", status.Error(codes.Canceled, "context canceled"), true},
		{"net operation canceled", generateOperationCanceled(t), true},
		{"EOF", io.EOF, true},
		{"url.Error EOF", &url.Error{Err: io.EOF}, true},
		{"transport is closing", status.Error(codes.Unavailable, "transport is closing"), true},
		{"grpc internal error", status.Error(codes.Internal, "other problem"), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(tt *testing.T) {
			if got := shouldIgnore(tc.err); got != tc.shouldIgnore {
				tt.Fatalf("got shouldIgnore %v want %v (err=%+v %T)", got, tc.shouldIgnore, tc.err, tc.err)
			}
		})
	}
}

func TestReportPanic(t *testing.T) {
	logger, hook := test.NewNullLogger()

	defer func() {
		if p := recover(); p == nil {
			t.Fatal("expected ReportPanic to repanic")
		}

		if want, got := 1, len(hook.Entries); want != got {
			t.Fatalf("want hook entries to be %d, got %d", want, got)
		}
		if want, got := "test message", hook.Entries[0].Message; want != got {
			t.Errorf("want hook entry message to be %q, got %q", want, got)
		}
	}()

	func() {
		defer ReportPanic(logger)

		panic("test message")
	}()
}

type tempError struct {
	temporary bool
}

func (e tempError) Error() string {
	return "temp error"
}

func (e tempError) Temporary() bool {
	return e.temporary
}

type timeoutError struct {
	timeout bool
}

func (e timeoutError) Error() string {
	return "timeout error"
}

func (e timeoutError) Timeout() bool {
	return e.timeout
}

func generateOperationCanceled(t testing.TB) error {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var d net.Dialer
	_, err := d.DialContext(ctx, "tcp", ":0")
	return err
}
