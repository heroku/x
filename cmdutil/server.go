package cmdutil

import (
	"context"

	"github.com/oklog/run"
)

// A Server can be run synchronously and return an error.
//
// Servers are typically used with oklog/run.Group.
type Server interface {
	Run() error
	Stop(error)
}

// ServerFunc is a function which implements the Server interface.
type ServerFunc func() error

// Run calls fn and returns any errors.
//
// It implements the Server interface.
func (fn ServerFunc) Run() error { return fn() }

// Stop is a noop for gradual compatibility with oklog run.Group.
//
// It implements the Server interface.
func (fn ServerFunc) Stop(error) {}

// ServerFuncs implements the Server interface with provided functions.
type ServerFuncs struct {
	RunFunc  func() error
	StopFunc func(error)
}

// Run calls RunFunc and returns any errors.
//
// It implements the Server interface.
func (sf ServerFuncs) Run() error {
	return sf.RunFunc()
}

// Stop calls StopFunc, if it's non-nil.
//
// It implements the Server interface.
func (sf ServerFuncs) Stop(err error) {
	if sf.StopFunc != nil {
		sf.StopFunc(err)
	}
}

// NewContextServer returns a Server that runs the given
// function with a context that is canceled when the Server
// is stopped.
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

// MultiServer returns a new kit.Server which will run all of the provided
// servers until one of them fails or the server is stopped.
func MultiServer(srvs ...Server) Server {
	var g run.Group

	s := NewContextServer(func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})

	g.Add(s.Run, s.Stop)

	for _, srv := range srvs {
		g.Add(srv.Run, srv.Stop)
	}

	return ServerFuncs{
		RunFunc:  g.Run,
		StopFunc: s.Stop,
	}
}
