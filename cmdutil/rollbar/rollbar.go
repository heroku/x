// Package rollbar provides helpers for setting up rollbar error reporting.
package rollbar

import (
	"context"
	"io"
	"net"
	"net/url"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/heroku/rollrus"
	"github.com/sirupsen/logrus"
)

// Config for Rollbar.
type Config struct {
	Token string `env:"ROLLBAR_TOKEN"`
	Env   string `env:"ROLLBAR_ENV"`
}

// Setup installs a Rollbar report handler to the default logrus logger.
//
// Setup is skipped if Token and Env are not present in the config.
func Setup(logger logrus.FieldLogger, cfg Config) {
	if cfg.Token == "" && cfg.Env == "" {
		logger.WithField("at", "skipping-rollbar").Info()
		return
	}

	logger.WithFields(logrus.Fields{
		"at":  "setup-rollbar",
		"env": cfg.Env,
	}).Info()

	hook := rollrus.NewHook(cfg.Token, cfg.Env,
		rollrus.WithIgnoreErrorFunc(shouldIgnore),
		rollrus.WithLevels(logrus.ErrorLevel, logrus.PanicLevel, logrus.FatalLevel),
	)

	logrus.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true})
	logrus.AddHook(hook)
}

func shouldIgnore(err error) bool {
	root := rootError(err)

	for _, fn := range ignoreFuncs {
		if fn(root) {
			return true
		}
	}

	return false
}

// ReportPanic attempts to report the panic to rollbar via the logrus.
func ReportPanic(logger logrus.FieldLogger) {
	if p := recover(); p != nil {
		logger.Panic(p)
	}
}

var ignoreFuncs = []func(error) bool{
	isCanceledOrEOF,
	isTimeout,
	isTemporary,
	isOperationCanceled,
	isClosing,
}

// rootError does unwrapping of stdlib errors. errors.Cause is not necessary
// since rollrus already did this for us.
func rootError(err error) error {
	if e, ok := err.(*url.Error); ok {
		return e.Err
	}
	return err
}

func isCanceledOrEOF(err error) bool {
	if err == context.Canceled || err == io.EOF {
		return true
	}

	if s, ok := status.FromError(err); ok && s.Code() == codes.Canceled {
		return true
	}

	return false
}

func isTimeout(err error) bool {
	type timeout interface {
		Timeout() bool
	}

	e, ok := err.(timeout)
	return ok && e.Timeout()
}

func isTemporary(err error) bool {
	type temporary interface {
		Temporary() bool
	}

	e, ok := err.(temporary)
	return ok && e.Temporary()
}

// isOperationCanceled is a little hacky because errCanceled is private, and
// does not implement any interface, therefore the only way we can check if
// it's the same error is by matching it's string.
// TODO(cyx) bradfitz has a TODO for himself to clean this up eventually, maybe
// one day we can remove this and it will just be a direct context.Canceled
// that's bubbled up from net.OpError.
func isOperationCanceled(err error) bool {
	_, ok := err.(*net.OpError)
	return ok && strings.Contains(err.Error(), "operation was canceled")
}

// isClosing detects gRPC transport errors caused by the unexpected termination
// of long-lived connections to an ELB after a deployment.
func isClosing(err error) bool {
	s, ok := status.FromError(err)
	if !ok {
		return false
	}

	return s.Code() == codes.Unavailable && s.Message() == "transport is closing"
}
