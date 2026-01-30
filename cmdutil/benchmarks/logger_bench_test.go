package benchmarks

import (
	"io"
	"testing"

	"github.com/sirupsen/logrus"
)

// BenchmarkLogrusStructured benchmarks logrus with structured fields
func BenchmarkLogrusStructured(b *testing.B) {
	logger := logrus.New()
	logger.Out = io.Discard
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		logger.WithFields(logrus.Fields{
			"method": "GET",
			"path":   "/api/users",
			"status": 200,
			"bytes":  1234,
		}).Info("request completed")
	}
}

// BenchmarkLogrusSimple benchmarks simple logrus logging
func BenchmarkLogrusSimple(b *testing.B) {
	logger := logrus.New()
	logger.Out = io.Discard
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		logger.Info("simple message")
	}
}

// BenchmarkLogrusWithContext benchmarks logrus with pre-configured fields
func BenchmarkLogrusWithContext(b *testing.B) {
	logger := logrus.New()
	logger.Out = io.Discard
	contextLogger := logger.WithFields(logrus.Fields{
		"app":    "myapp",
		"deploy": "production",
	})
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		contextLogger.WithField("status", 200).Info("request")
	}
}

// BenchmarkLogrusConcurrent benchmarks concurrent logrus logging
func BenchmarkLogrusConcurrent(b *testing.B) {
	logger := logrus.New()
	logger.Out = io.Discard
	
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.WithField("status", 200).Info("request")
		}
	})
}
