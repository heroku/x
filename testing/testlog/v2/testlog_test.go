package testlog

import (
	"log/slog"
	"testing"
)

func TestHook(t *testing.T) {
	logger, hook := New()

	logger.Info("test message", "key", "value")

	entries := hook.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	if entries[0].Message != "test message" {
		t.Errorf("expected message 'test message', got %q", entries[0].Message)
	}

	last := hook.LastEntry()
	if last == nil {
		t.Fatal("expected last entry, got nil")
	}
	if last.Message != "test message" {
		t.Errorf("expected last message 'test message', got %q", last.Message)
	}

	hook.Reset()
	if len(hook.Entries()) != 0 {
		t.Errorf("expected 0 entries after reset, got %d", len(hook.Entries()))
	}
}

func TestCheckMethods(t *testing.T) {
	logger, hook := New()

	logger.Info("hello world", "foo", "bar")
	logger.Error("error occurred", slog.Int("code", 42))

	hook.CheckContained(t, "hello")
	hook.CheckContained(t, "error occurred")
	hook.CheckAllContained(t, "hello", "foo=bar")
	hook.CheckNotContained(t, "notfound")
}
