package signals

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/heroku/x/testing/testlog"
)

func TestWithNotifyCancel(t *testing.T) {
	notified := make(chan os.Signal, 1)
	ctx := notifyContext(context.Background(), notified, syscall.SIGINT)

	notified <- syscall.SIGINT
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatalf("expected ctx to be canceled")
	}
}

func TestNewServer(t *testing.T) {
	logger, _ := testlog.New()

	sv := NewServer(logger, syscall.SIGWINCH)

	var (
		runErr  error
		runDone = make(chan struct{})
		done    = make(chan struct{})
	)
	defer close(done)

	go func() {
		runErr = sv.Run()
		close(runDone)
	}()

	// We're racing with Run starting and calling signal.Notify, so loop
	// it until the test is done.
	go func() {
		for {
			select {
			case <-done:
				return
			default:
			}
			if err := syscall.Kill(syscall.Getpid(), syscall.SIGWINCH); err != nil {
				t.Error(err)
			}
			time.Sleep(time.Millisecond)
		}
	}()

	select {
	case <-runDone:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Run took too long")
	}

	if runErr != nil {
		t.Fatalf("got Run error %+v, want no error", runErr)
	}

	sv.Stop(nil)
}

// Ensure Run returns when Stop is called, even if no signal
// has been received.
func TestNewServerNoSignal(t *testing.T) {
	logger, _ := testlog.New()

	sv := NewServer(logger, syscall.SIGWINCH)

	var runErr error
	done := make(chan struct{})

	go func() {
		runErr = sv.Run()
		close(done)
	}()

	sv.Stop(nil)
	<-done

	if runErr != nil {
		t.Fatalf("got Run error %+v, want no error", runErr)
	}
}
