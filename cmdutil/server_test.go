package cmdutil

import (
	"context"
	"errors"
	"testing"
)

func TestServerFunc(t *testing.T) {
	err := errors.New("this is an error")
	sf := ServerFunc(func() error { return err })
	if got := sf.Run(); got != err {
		t.Fatalf("got Run err %+v, want %+v", got, err)
	}
}

func TestServerFuncs(t *testing.T) {
	err := errors.New("this is an error")
	var stopErr error

	sf := ServerFuncs{
		RunFunc:  func() error { return err },
		StopFunc: func(e error) { stopErr = e },
	}

	if got := sf.Run(); got != err {
		t.Fatalf("got Run err %+v, want %+v", got, err)
	}

	sf.Stop(err)
	if got := stopErr; got != err {
		t.Fatalf("got Stop err %+v, want %+v", got, err)
	}

	// Should not blow up if StopFunc is nil.
	sf.StopFunc = nil
	sf.Stop(err)
}

func TestNewContextServer(t *testing.T) {
	err := errors.New("this is an error")
	var gotCtx context.Context

	s := NewContextServer(func(ctx context.Context) error {
		gotCtx = ctx
		return err
	})

	if got := s.Run(); got != err {
		t.Fatalf("got Run err %+v, want %+v", got, err)
	}

	if got := gotCtx.Err(); got != nil {
		t.Fatalf("got context Err %+v, wanted none", got)
	}

	s.Stop(err)
	<-gotCtx.Done()

	if got := gotCtx.Err(); got != context.Canceled {
		t.Fatalf("got context Err %+v, wanted context.Canceled", got)
	}
}

// TestMultiServerStop tests that when the MultiServer is stopped it stops the
// servers that were added to it.
func TestMultiServerStop(t *testing.T) {
	s1 := newStoppingServer()
	s2 := newStoppingServer()
	ms := MultiServer(s1, s2)

	done := make(chan struct{})
	go func() {
		ms.Run()
		close(done)
	}()

	ms.Stop(nil)

	// MultiServer should complete after it's stoped
	<-done

	if !s1.stopped {
		t.Error("want s1 to be stopped")
	}

	if !s2.stopped {
		t.Error("want s2 to be stopped")
	}
}

// TestMultiServer_InnerStop tests that when any one of the servers inside the
// MultiServer stops, the entire MultiServer stops.
func TestMultiServer_InnerStop(t *testing.T) {
	s1 := newStoppingServer()
	s2 := newStoppingServer()
	ms := MultiServer(s1, s2)

	done := make(chan struct{})
	go func() {
		ms.Run()
		close(done)
	}()

	s1.Stop(nil)

	// MultiServer should complete after any server is stopped
	<-done

	if !s1.stopped {
		t.Error("want s1 to be stopped")
	}

	if !s2.stopped {
		t.Error("want s2 to be stopped")
	}
}

type stoppingServer struct {
	stopped bool
	Server
}

func newStoppingServer() *stoppingServer {
	s := &stoppingServer{}
	s.Server = NewContextServer(func(ctx context.Context) error {
		<-ctx.Done()
		s.stopped = true
		return ctx.Err()
	})
	return s
}
