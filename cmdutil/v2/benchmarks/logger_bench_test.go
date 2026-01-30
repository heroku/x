package benchmarks

import (
	"io"
	"log/slog"
	"testing"
)

// BenchmarkSlogStructured benchmarks slog with structured fields
func BenchmarkSlogStructured(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		logger.Info("request completed",
			slog.String("method", "GET"),
			slog.String("path", "/api/users"),
			slog.Int("status", 200),
			slog.Int("bytes", 1234),
		)
	}
}

// BenchmarkSlogSimple benchmarks simple slog logging
func BenchmarkSlogSimple(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		logger.Info("simple message")
	}
}

// BenchmarkSlogWithContext benchmarks slog with pre-configured fields
func BenchmarkSlogWithContext(b *testing.B) {
	base := slog.New(slog.NewTextHandler(io.Discard, nil))
	logger := base.With(
		slog.String("app", "myapp"),
		slog.String("deploy", "production"),
	)
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		logger.Info("request", slog.Int("status", 200))
	}
}

// BenchmarkSlogConcurrent benchmarks concurrent slog logging
func BenchmarkSlogConcurrent(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("request", slog.Int("status", 200))
		}
	})
}
