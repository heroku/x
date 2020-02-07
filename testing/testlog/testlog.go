// Package testlog provides a test logger and helpers to check log output.
package testlog

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"
)

// Hook is a hook designed for dealing with logs in test scenarios.
type Hook struct {
	sync.Mutex
	entries []*logrus.Entry
}

// New sets up a test logger that produces no output. Use the returned hook to
// observe and make assertions about what was logged.
func New() (*logrus.Logger, *Hook) {
	l := logrus.New()
	l.Out = ioutil.Discard

	hook := new(Hook)
	l.Hooks.Add(hook)

	return l, hook
}

// NewNullLogger Creates a discarding logger and installs the test hook.
//
// Deprecated: Use New instead.
func NewNullLogger() (*logrus.Logger, *Hook) {
	return New()
}

// Entries is a thread safe accessor for all entries.
func (t *Hook) Entries() []*logrus.Entry {
	t.Lock()
	defer t.Unlock()

	res := make([]*logrus.Entry, len(t.entries))

	for idx, e := range t.entries {
		res[idx] = &logrus.Entry{
			Logger:  e.Logger,
			Time:    e.Time,
			Data:    e.Data,
			Message: e.Message,
			Level:   e.Level,
		}
	}
	return res
}

// Levels complies to the Hook interface.
func (t *Hook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire complies to the Hook interface.
func (t *Hook) Fire(e *logrus.Entry) error {
	t.Lock()
	defer t.Unlock()

	t.entries = append(t.entries, e)
	return nil
}

// LastEntry returns the last entry that was logged or nil.
func (t *Hook) LastEntry() (l *logrus.Entry) {
	t.Lock()
	defer t.Unlock()

	if i := len(t.entries) - 1; i >= 0 {
		return t.entries[i]
	}
	return nil
}

// String returns the string representation of all the entries cumulatively
// logged in this Hook. If isolation is needed, prefer to make a new hook
// per test case.
func (t *Hook) String() string {
	var res []string
	entries := t.Entries()
	for _, e := range entries {
		if s, err := e.String(); err == nil {
			res = append(res, s)
		}
	}
	return strings.Join(res, " ")
}

// Reset removes all Entries from this test hook.
func (t *Hook) Reset() {
	t.Lock()
	defer t.Unlock()

	t.entries = make([]*logrus.Entry, 0)
}

// CheckContained looks through all the passed strings and verifies that
// at least one of those have been logged.
func (t *Hook) CheckContained(tb testing.TB, strs ...string) {
	tb.Helper()

	if strs == nil {
		return
	}

	found := false
	for _, str := range strs {
		found = found || contains(t.String(), str)
	}

	if !found {
		tb.Fatalf("got entries:\n%v\nexpected to find:\n%v\n", t.String(), strs)
	}
}

// CheckNotContained looks through all the passed strings and verifies that
// none of those fragments have been logged.
func (t *Hook) CheckNotContained(tb testing.TB, strs ...string) {
	tb.Helper()

	for _, str := range strs {
		if contains(t.String(), str) {
			tb.Fatalf("got `%s` expected none in %s", str, t.String())
		}
	}
}

// CheckAllContained looks through all the passed strings and verifies that
// all of them have been logged.
func (t *Hook) CheckAllContained(tb testing.TB, strs ...string) {
	tb.Helper()

	if strs == nil {
		return
	}

	found := 0
	for _, str := range strs {
		if contains(t.String(), str) {
			found++
		}
	}

	if found != len(strs) {
		tb.Fatalf("got entries: `%v` expected to find: `%v`", t.String(), strs)
	}
}

func contains(haystack, needle string) bool {
	needle = canonicalizeQuotes(needle)
	return strings.Contains(haystack, needle)
}

func canonicalizeQuotes(str string) string {
	chunks := strings.Split(str, "=")
	if len(chunks) != 2 {
		return str
	}

	key := chunks[0]
	val, err := strconv.Unquote(chunks[1])
	if err != nil {
		return str
	}

	if needsQuoting(val) {
		return fmt.Sprintf("%s=%q", key, val)
	}

	return fmt.Sprintf("%s=%s", key, val)
}

// Doesn't need quoting: a-z, A-Z, 0-9, '@', '-', '+', '.', '_', '/', & '@'
func needsQuoting(text string) bool {
	for _, ch := range text { // runes are bytes, so these are ascii lookups
		if !((ch >= '@' && ch <= 'Z') || // ASCII 64 .. 90
			(ch >= 'a' && ch <= 'z') || // ASCII 97 .. 122
			(ch >= '.' && ch <= '9') || // ASCII 46 .. 57
			(ch >= '^' && ch <= '_') || // ASCII 94 .. 95
			ch == '+' || ch == '-') { // ASCII 43 & 45
			return true
		}
	}
	return false
}
