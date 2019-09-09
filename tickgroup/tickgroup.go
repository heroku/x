// Package tickgroup allows a collection of goroutines to call a subtask every set time interval.
package tickgroup

import (
	"context"
	"time"

	"golang.org/x/sync/errgroup"
)

// A Group is a collection of goroutines working on subtask that are spawned on
// a set interval per task.
type Group struct {
	g     *errgroup.Group
	donec <-chan struct{}
}

// New returns a new Group that stops spawning subtasks when ctx is done.
func New(ctx context.Context) *Group {
	return &Group{g: new(errgroup.Group), donec: ctx.Done()}
}

// WithContext creates a child context from the given context, and uses that to
// control context cancelation.
func WithContext(ctx context.Context) (*Group, context.Context) {
	g, ctx := errgroup.WithContext(ctx)
	return &Group{g: g, donec: ctx.Done()}, ctx
}

// Go spawns a subtask f every d. The goroutine terminates when an error is
// encountered. The first call to return a non-nil error cancels the group; its
// error will be returned by Wait.
func (g *Group) Go(d time.Duration, f func() error) {
	g.g.Go(func() error {
		if err := f(); err != nil {
			return err
		}

		t := time.NewTicker(d)
		defer t.Stop()

		for {
			select {
			case <-g.donec:
				return nil

			case <-t.C:
				if err := f(); err != nil {
					return err
				}
			}
		}
	})
}

// Wait blocks until all function calls from the Go method have returned,
// then returns the first non-nil error (if any) from them.
func (g *Group) Wait() error { return g.g.Wait() }
