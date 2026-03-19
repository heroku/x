// Package slogrus provides a bridge from *slog.Logger to logrus.FieldLogger.
//
// This is a temporary compatibility shim for vendored dependencies that require
// logrus.FieldLogger. Own code should use *slog.Logger directly. The bridge
// should only be used at boundaries where vendored deps require logrus.FieldLogger,
// never on hot paths. WithField/WithFields/WithError allocate logrus entries
// internally.
package slogrus

import (
	"io"
	"log/slog"

	"github.com/sirupsen/logrus"
)

// Bridge satisfies logrus.FieldLogger, forwarding log calls to a *slog.Logger.
type Bridge struct {
	slog *slog.Logger
	lr   *logrus.Logger
}

var _ logrus.FieldLogger = (*Bridge)(nil)

// New returns a Bridge that satisfies logrus.FieldLogger and forwards to l.
func New(l *slog.Logger) *Bridge {
	lr := logrus.New()
	lr.SetOutput(io.Discard)
	lr.SetLevel(logrus.TraceLevel)
	lr.AddHook(&hook{slog: l})
	return &Bridge{slog: l, lr: lr}
}

// Direct logging — fast path through slog.

func (b *Bridge) Debug(args ...interface{})                 { b.slog.Debug(sprint(args)) }
func (b *Bridge) Info(args ...interface{})                  { b.slog.Info(sprint(args)) }
func (b *Bridge) Warn(args ...interface{})                  { b.slog.Warn(sprint(args)) }
func (b *Bridge) Warning(args ...interface{})               { b.slog.Warn(sprint(args)) }
func (b *Bridge) Error(args ...interface{})                 { b.slog.Error(sprint(args)) }
func (b *Bridge) Fatal(args ...interface{})                 { b.slog.Error(sprint(args)) }
func (b *Bridge) Panic(args ...interface{})                 { b.slog.Error(sprint(args)) }
func (b *Bridge) Print(args ...interface{})                 { b.slog.Info(sprint(args)) }
func (b *Bridge) Debugf(format string, args ...interface{}) { b.slog.Debug(sprintf(format, args)) }
func (b *Bridge) Infof(format string, args ...interface{})  { b.slog.Info(sprintf(format, args)) }
func (b *Bridge) Warnf(format string, args ...interface{})  { b.slog.Warn(sprintf(format, args)) }
func (b *Bridge) Warningf(format string, args ...interface{}) {
	b.slog.Warn(sprintf(format, args))
}
func (b *Bridge) Errorf(format string, args ...interface{}) { b.slog.Error(sprintf(format, args)) }
func (b *Bridge) Fatalf(format string, args ...interface{}) { b.slog.Error(sprintf(format, args)) }
func (b *Bridge) Panicf(format string, args ...interface{}) { b.slog.Error(sprintf(format, args)) }
func (b *Bridge) Printf(format string, args ...interface{}) { b.slog.Info(sprintf(format, args)) }
func (b *Bridge) Debugln(args ...interface{})               { b.slog.Debug(sprintln(args)) }
func (b *Bridge) Infoln(args ...interface{})                { b.slog.Info(sprintln(args)) }
func (b *Bridge) Warnln(args ...interface{})                { b.slog.Warn(sprintln(args)) }
func (b *Bridge) Warningln(args ...interface{})             { b.slog.Warn(sprintln(args)) }
func (b *Bridge) Errorln(args ...interface{})               { b.slog.Error(sprintln(args)) }
func (b *Bridge) Fatalln(args ...interface{})               { b.slog.Error(sprintln(args)) }
func (b *Bridge) Panicln(args ...interface{})               { b.slog.Error(sprintln(args)) }
func (b *Bridge) Println(args ...interface{})               { b.slog.Info(sprintln(args)) }

// Field chaining — slow path through logrus entries.

func (b *Bridge) WithField(key string, value interface{}) *logrus.Entry {
	return b.lr.WithField(key, value)
}

func (b *Bridge) WithFields(fields logrus.Fields) *logrus.Entry {
	return b.lr.WithFields(fields)
}

func (b *Bridge) WithError(err error) *logrus.Entry {
	return b.lr.WithError(err)
}

// hook forwards logrus entries to slog.
type hook struct {
	slog *slog.Logger
}

func (h *hook) Levels() []logrus.Level { return logrus.AllLevels }

func (h *hook) Fire(e *logrus.Entry) error {
	attrs := make([]slog.Attr, 0, len(e.Data))
	for k, v := range e.Data {
		attrs = append(attrs, slog.Any(k, v))
	}
	level := mapLevel(e.Level)
	h.slog.LogAttrs(e.Context, level, e.Message, attrs...)
	return nil
}

func mapLevel(l logrus.Level) slog.Level {
	switch l {
	case logrus.TraceLevel, logrus.DebugLevel:
		return slog.LevelDebug
	case logrus.InfoLevel:
		return slog.LevelInfo
	case logrus.WarnLevel:
		return slog.LevelWarn
	default: // Error, Fatal, Panic
		return slog.LevelError
	}
}
