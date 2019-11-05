package cmdutil

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/oklog/run"
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
		if err := ms.Run(); err != nil && err != context.Canceled {
			panic(err)
		}
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
		if err := ms.Run(); err != nil && err != context.Canceled {
			panic(err)
		}
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

func ExampleServerFunc() {
	var a Server = ServerFunc(
		func() error {
			fmt.Println("A")
			return nil
		},
	)
	var g run.Group
	g.Add(a.Run, a.Stop)
	if err := g.Run(); err != nil {
		panic(err)
	}
	// Output: A
}

func ExampleServerFuncs() {
	var a Server = ServerFuncs{
		RunFunc: func() error {
			fmt.Println("A")
			return nil
		},
		StopFunc: func(err error) {

		},
	}
	var g run.Group
	g.Add(a.Run, a.Stop)
	if err := g.Run(); err != nil {
		panic(err)
	}
	// Output: A
}

func ExampleNewContextServer() {
	s := NewContextServer(
		func(ctx context.Context) error {
			// do something that doesn't block ex:
			fmt.Println("A")
			<-ctx.Done() // block waiting for context to be canceled
			// can do any cleanup after this, or just return nil
			// return cleanup()
			return nil
		},
	)
	var exitFast Server = ServerFunc(
		// doesn't do anything, just returns
		func() error {
			return nil
		},
	)
	var g run.Group
	g.Add(s.Run, s.Stop)
	g.Add(exitFast.Run, exitFast.Stop)
	if err := g.Run(); err != nil {
		panic(err)
	}
	// Output: A
}

func ExampleMultiServer() {
	s := MultiServer(
		ServerFunc(
			func() error {
				fmt.Println("A")
				return nil
			}),
	)
	if err := s.Run(); err != nil {
		panic(err)
	}
	// Output: A
}

func ExampleMultiServer_stop() {
	done := make(chan struct{})
	s := MultiServer(
		ServerFuncs{
			RunFunc: func() error {
				fmt.Println("A")
				<-done
				fmt.Println("B")
				return nil
			},
			StopFunc: func(err error) {
				fmt.Println(err)
				close(done)
			},
		},
	)
	go func() {
		time.Sleep(1 * time.Second)
		s.Stop(io.EOF)
	}()
	if err := s.Run(); err != nil && err != context.Canceled {
		panic(err)
	}
	// Output: A
	// context canceled
	// B
}
