package slogrus

import (
	"io"
	"log/slog"
	"testing"

	"github.com/sirupsen/logrus"
)

func BenchmarkDirectInfo(b *testing.B) {
	l := slog.New(slog.NewTextHandler(io.Discard, nil))
	br := New(l)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		br.Info("msg")
	}
}

func BenchmarkWithFieldInfo(b *testing.B) {
	l := slog.New(slog.NewTextHandler(io.Discard, nil))
	br := New(l)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		br.WithField("key", "val").Info("msg")
	}
}

func BenchmarkWithFieldsInfo(b *testing.B) {
	l := slog.New(slog.NewTextHandler(io.Discard, nil))
	br := New(l)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		br.WithFields(logrus.Fields{"a": "1", "b": "2"}).Info("msg")
	}
}

func BenchmarkNativeSlog(b *testing.B) {
	l := slog.New(slog.NewTextHandler(io.Discard, nil))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		l.Info("msg")
	}
}
