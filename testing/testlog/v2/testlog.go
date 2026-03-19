// Package testlog provides a test logger and helpers to check log output.
package testlog

import (
	"bytes"
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
	return slog.New(&handler{hook: h}), h
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

// String returns a string representation of all log records, rendered through
// slog.TextHandler so the output matches standard slog formatting. Values
// containing spaces or special characters will be quoted by slog (e.g.
// msg="hello world"). Assertions via CheckContained should match the quoted
// form.
func (h *Hook) String() string {
	entries := h.Entries()
	var buf bytes.Buffer
	th := slog.NewTextHandler(&buf, nil)
	for _, r := range entries {
		_ = th.Handle(context.Background(), r)
	}
	return strings.TrimSpace(buf.String())
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

// handler implements slog.Handler, storing records in a shared Hook
// while carrying pre-set attrs and group context.
type handler struct {
	hook  *Hook
	attrs []slog.Attr
	group string
}

func (h *handler) Enabled(context.Context, slog.Level) bool { return true }

func (h *handler) Handle(_ context.Context, r slog.Record) error {
	if len(h.attrs) > 0 {
		r.AddAttrs(h.attrs...)
	}
	h.hook.mu.Lock()
	defer h.hook.mu.Unlock()
	h.hook.records = append(h.hook.records, r.Clone())
	return nil
}

func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if h.group != "" {
		grouped := make([]any, len(attrs))
		for i, a := range attrs {
			grouped[i] = a
		}
		return &handler{
			hook:  h.hook,
			attrs: append(append([]slog.Attr{}, h.attrs...), slog.Group(h.group, grouped...)),
		}
	}
	return &handler{
		hook:  h.hook,
		attrs: append(append([]slog.Attr{}, h.attrs...), attrs...),
	}
}

func (h *handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return &handler{
		hook:  h.hook,
		attrs: append([]slog.Attr{}, h.attrs...),
		group: name,
	}
}
