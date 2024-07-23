//go:build integration
// +build integration

package service

import (
	"os"
	"testing"

	"github.com/heroku/x/cmdutil"
	"github.com/heroku/x/go-kit/metrics/l2met"
)

func TestPanicReporting(t *testing.T) {

	os.Setenv("APP_NAME", "test-app")
	os.Setenv("DEPLOY", "test")

	t.Cleanup(func() {
		os.Unsetenv("APP_NAME")
		os.Unsetenv("DEPLOY")
	})

	var cfg struct {
		Val string `env:"TEST_VAL,default=test"`
	}

	s := New(&cfg)

	l2met := l2met.New(s.Logger)
	s.MetricsProvider = l2met
	s.Add(cmdutil.NewContextServer(l2met.Run))

	f := func() error {
		panic("test panic")
		return nil
	}

	s.Add(cmdutil.ServerFunc(f))
	s.Run()

}
