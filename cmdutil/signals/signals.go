// Package signals provides a signal handler which is usable as a cmdutil.Server.
package signals

import (
	"context"
	"os"
	"os/signal"

	"github.com/sirupsen/logrus"

	"github.com/heroku/x/cmdutil"
)

// WithNotifyCancel creates a sub-context from the given context which gets
// canceled upon receiving any of the configured signals.
func WithNotifyCancel(ctx context.Context, signals ...os.Signal) context.Context {
	notified := make(chan os.Signal, 1)
	return notifyContext(ctx, notified, signals...)
}

func notifyContext(ctx context.Context, notified chan os.Signal, signals ...os.Signal) context.Context {
	ctx, cancel := context.WithCancel(ctx)
	signal.Notify(notified, signals...)

	go func() {
		<-notified
		cancel()
	}()

	return ctx
}

// NewServer returns a cmdutil.Server that returns from Run
// when any of the provided signals are received.
// Run always returns a nil error.
func NewServer(logger logrus.FieldLogger, signals ...os.Signal) cmdutil.Server {
	ch := make(chan os.Signal, 1)

	return cmdutil.ServerFuncs{
		RunFunc: func() error {
			signal.Notify(ch, signals...)
			sig := <-ch
			if sig != nil {
				logger.Infoln("received signal", sig)
			}
			return nil
		},
		StopFunc: func(error) {
			signal.Stop(ch)
			select {
			case ch <- nil:
			default:
			}
		},
	}
}
