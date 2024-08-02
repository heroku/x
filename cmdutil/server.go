// Package cmdutil provides abstractions for building things which can be
// started and stopped as a part of a executable's process lifecycle.
package cmdutil

import (
	"context"

	"github.com/oklog/run"
)

// Server runs synchronously and returns any errors. The Run method is expected
// to block until finished, returning any error, or until Stop is called.
// used with oklog/run.Group, where the first Server.Run to return will cancel
// the Group (regardless of the error returned). Use NewContextServer to create
// a Server that can block on a Context until Stop is called.
//
// TODO[freeformz]: Document why Stop takes an error and what to do with is.
type Server interface {
	Run() error
	Stop(error)
}

// ensure ServerFunc implements Server
var _ Server = ServerFunc(func() error { return nil })

// ServerFunc adapts a function to the Server interface. This is useful to adapt
// a closure to being a Server.
type ServerFunc func() error

// Run the function, returning any errors.
func (fn ServerFunc) Run() error { return fn() }

// Stop is a noop for gradual compatibility with oklog run.Group.
func (fn ServerFunc) Stop(error) {}

// ensure ServerFuncs implements Server
var _ Server = ServerFuncs{}

// ServerFuncs adapts two functions, one for Run, one for Stop, to the Server
// interface. This is useful for adapting closures so they can be used as a
// Server.
type ServerFuncs struct {
	RunFunc  func() error
	StopFunc func(error)
}

// Run the Server.
func (sf ServerFuncs) Run() error {
	return sf.RunFunc()
}

// Stop the Server.
func (sf ServerFuncs) Stop(err error) {
	if sf.StopFunc != nil {
		sf.StopFunc(err)
	}
}

// NewContextServer that when Run(), calls the given function with a context
// that is canceled when Stop() is called.
func NewContextServer(fn func(context.Context) error) Server {
	ctx, cancel := context.WithCancel(context.Background())

	return &ServerFuncs{
		RunFunc: func() error {
			return fn(ctx)
		},
		StopFunc: func(error) {
			cancel()
		},
	}

}

// MultiServer which, when Run, will run all of the servers until one of them
// returns or is itself Stopped.
func MultiServer(servers ...Server) Server {
	var g run.Group

	s := NewContextServer(func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})

	g.Add(s.Run, s.Stop)

	for _, srv := range servers {
		g.Add(srv.Run, srv.Stop)
	}

	return ServerFuncs{
		RunFunc:  g.Run,
		StopFunc: s.Stop,
	}
}
