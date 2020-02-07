package testlog_test

import (
	"testing"

	"github.com/heroku/x/testing/testlog"
)

func Example() {
	var t *testing.T // use t provided by test function

	logger, hook := testlog.New()

	// Use the test logger
	logger.WithField("a", "b").Info("test")

	hook.CheckAllContained(t, "a=b", "msg=test")
	hook.CheckNotContained(t, "unexpected")
}

func Example_discard() {
	logger, _ := testlog.New()

	// Use the test logger
	logger.WithField("a", "b").Info("test")

	// Output:
}
