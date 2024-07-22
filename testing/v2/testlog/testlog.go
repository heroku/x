package testlog

import (
	"bytes"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

// Hook is used for validating logs.
type Hook struct {
	buf *bytes.Buffer
}

// New returns a new logger and hook suitable for testing.
func New() (*slog.Logger, *Hook) {
	hook := &Hook{
		buf: bytes.NewBuffer([]byte{}),
	}

	hopts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	logger := slog.New(slog.NewTextHandler(hook.buf, hopts)).With(
		slog.String("app", "test-app"),
		slog.String("deploy", "local"),
		slog.String("dyno", "web.1"),
	)

	return logger, hook

}

// ExpectLogLine uses the hook to validate that
// the next log line contains the passed message and set of key-values in the passed map.
func (hook *Hook) ExpectLogLine(t *testing.T, msg string, m map[string]interface{}) {
	ExpectLogLineFromBuffer(t, hook.buf, msg, m)
}

// ExpectLogLineFromBuffer is the same as hook.ExpectLogLine but instead validates lines from a buffer.
func ExpectLogLineFromBuffer(t *testing.T, b *bytes.Buffer, msg string, m map[string]interface{}) {
	line, err := b.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(line, msg) {
		t.Errorf("expected log line to contain message: %s", msg)
	}

	for k, v := range m {
		if !strings.Contains(line, fmt.Sprintf("%s=%s", k, v)) {
			t.Errorf("expected log line to contain %s=%s", k, v)
		}
	}
}
