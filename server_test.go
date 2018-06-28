package kit

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
