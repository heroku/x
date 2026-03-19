package slogrus

import (
	"bytes"
	"fmt"
	"log/slog"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"
)

func newTestSlog() (*slog.Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	return slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})), &buf
}

func TestDirectMethods(t *testing.T) {
	l, buf := newTestSlog()
	b := New(l)

	b.Info("hello")
	b.Error("boom")
	b.Warn("careful")
	b.Debug("trace")

	out := buf.String()
	for _, want := range []string{"hello", "boom", "careful", "trace"} {
		if !bytes.Contains([]byte(out), []byte(want)) {
			t.Errorf("missing %q in output:\n%s", want, out)
		}
	}
}

func TestWithField(t *testing.T) {
	l, buf := newTestSlog()
	b := New(l)

	b.WithField("key", "val").Info("msg")

	out := buf.String()
	for _, want := range []string{"msg", "key=val"} {
		if !bytes.Contains([]byte(out), []byte(want)) {
			t.Errorf("missing %q in output:\n%s", want, out)
		}
	}
}

func TestWithFields(t *testing.T) {
	l, buf := newTestSlog()
	b := New(l)

	b.WithFields(logrus.Fields{"a": "1", "b": "2"}).Info("msg")

	out := buf.String()
	for _, want := range []string{"msg", "a=1", "b=2"} {
		if !bytes.Contains([]byte(out), []byte(want)) {
			t.Errorf("missing %q in output:\n%s", want, out)
		}
	}
}

func TestWithError(t *testing.T) {
	l, buf := newTestSlog()
	b := New(l)

	b.WithError(fmt.Errorf("fail")).Error("oops")

	out := buf.String()
	for _, want := range []string{"oops", "error=fail"} {
		if !bytes.Contains([]byte(out), []byte(want)) {
			t.Errorf("missing %q in output:\n%s", want, out)
		}
	}
}

func TestLevelMapping(t *testing.T) {
	tests := []struct {
		logrus logrus.Level
		slog   slog.Level
	}{
		{logrus.TraceLevel, slog.LevelDebug},
		{logrus.DebugLevel, slog.LevelDebug},
		{logrus.InfoLevel, slog.LevelInfo},
		{logrus.WarnLevel, slog.LevelWarn},
		{logrus.ErrorLevel, slog.LevelError},
		{logrus.FatalLevel, slog.LevelError},
		{logrus.PanicLevel, slog.LevelError},
	}
	for _, tt := range tests {
		if got := mapLevel(tt.logrus); got != tt.slog {
			t.Errorf("mapLevel(%v) = %v, want %v", tt.logrus, got, tt.slog)
		}
	}
}

func TestConcurrency(t *testing.T) {
	l, _ := newTestSlog()
	b := New(l)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			b.WithField("i", n).Info("concurrent")
		}(i)
	}
	wg.Wait()
}
