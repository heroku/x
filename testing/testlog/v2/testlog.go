// Package testlog provides a test logger and helpers to check log output.
package testlog

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
)

// Hook captures log records for testing.
type Hook struct {
	mu      sync.Mutex
	records []slog.Record
}

// New creates a slog.Logger that captures log records in a Hook.
func New() (*slog.Logger, *Hook) {
	h := &Hook{}
	return slog.New(h), h
}

// Enabled implements slog.Handler.
func (h *Hook) Enabled(context.Context, slog.Level) bool {
	return true
}

// Handle implements slog.Handler.
func (h *Hook) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, r.Clone())
	return nil
}

// WithAttrs implements slog.Handler.
func (h *Hook) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

// WithGroup implements slog.Handler.
func (h *Hook) WithGroup(name string) slog.Handler {
	return h
}

// Entries returns all captured log records.
func (h *Hook) Entries() []slog.Record {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]slog.Record{}, h.records...)
}

// LastEntry returns the last captured log record or nil.
func (h *Hook) LastEntry() *slog.Record {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.records) == 0 {
		return nil
	}
	r := h.records[len(h.records)-1]
	return &r
}

// Reset clears all captured log records.
func (h *Hook) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = nil
}

// String returns a string representation of all log records.
func (h *Hook) String() string {
	entries := h.Entries()
	var sb strings.Builder
	for i, r := range entries {
		if i > 0 {
			sb.WriteByte(' ')
		}
		sb.WriteString(r.Message)
		r.Attrs(func(a slog.Attr) bool {
			sb.WriteByte(' ')
			sb.WriteString(a.Key)
			sb.WriteByte('=')
			sb.WriteString(a.Value.String())
			return true
		})
	}
	return sb.String()
}

// CheckContained verifies at least one of the strings appears in logs.
func (h *Hook) CheckContained(tb testing.TB, strs ...string) {
	tb.Helper()
	if len(strs) == 0 {
		return
	}
	s := h.String()
	for _, str := range strs {
		if strings.Contains(s, str) {
			return
		}
	}
	tb.Fatalf("got entries:\n%v\nexpected to find one of:\n%v\n", s, strs)
}

// CheckNotContained verifies none of the strings appear in logs.
func (h *Hook) CheckNotContained(tb testing.TB, strs ...string) {
	tb.Helper()
	s := h.String()
	for _, str := range strs {
		if strings.Contains(s, str) {
			tb.Fatalf("got `%s` expected none in %s", str, s)
		}
	}
}

// CheckAllContained verifies all strings appear in logs.
func (h *Hook) CheckAllContained(tb testing.TB, strs ...string) {
	tb.Helper()
	if len(strs) == 0 {
		return
	}
	s := h.String()
	for _, str := range strs {
		if !strings.Contains(s, str) {
			tb.Fatalf("got entries: `%v` expected to find: `%v`", s, strs)
		}
	}
}
