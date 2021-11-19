package lambda

import (
	"os"
	"testing"
)

func TestNew_Function(t *testing.T) {
	setupFuncConfig(t)

	f := New(nil)

	if f.Logger == nil {
		t.Fatal("logger not configured")
	}

	if f.MetricsProvider == nil {
		t.Fatal("metrics provider not configured")
	}
}

func setupFuncConfig(t *testing.T) {
	os.Setenv("APP_NAME", "test-app")
	os.Setenv("DEPLOY", "test")

	t.Cleanup(func() {
		os.Unsetenv("APP_NAME")
		os.Unsetenv("DEPLOY")
	})
}
