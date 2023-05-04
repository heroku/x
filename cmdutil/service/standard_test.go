package service_test

import (
	"os"
	"testing"
	"time"

	"github.com/heroku/x/cmdutil/service"
)

func TestNewNoConfig(t *testing.T) {
	setupStandardConfig(t)

	s := service.New(nil)

	if s.Logger == nil {
		t.Fatal("standard logger not configured")
	}

	if s.MetricsProvider == nil {
		t.Fatal("standard metrics provider not configured")
	}
}

func TestNewCustomConfig(t *testing.T) {
	setupStandardConfig(t)

	os.Setenv("TEST_VAL", "1m")
	defer os.Unsetenv("TEST_VAL")

	var cfg struct {
		Val time.Duration `env:"TEST_VAL"`
	}
	s := service.New(&cfg)

	if s.Logger == nil {
		t.Fatal("standard logger not configured")
	}

	if s.MetricsProvider == nil {
		t.Fatal("standard metrics provider not configured")
	}

	if cfg.Val != time.Minute {
		t.Fatalf("cfg.Val = %v want %v", cfg.Val, time.Minute)
	}
}

func setupStandardConfig(t *testing.T) {
	os.Setenv("APP_NAME", "test-app")
	os.Setenv("DEPLOY", "test")

	t.Cleanup(func() {
		os.Unsetenv("APP_NAME")
		os.Unsetenv("DEPLOY")
	})
}
