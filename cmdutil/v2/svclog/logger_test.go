package svclog

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestNewLoggerIncludesAppAndDeploy(t *testing.T) {
	var buf bytes.Buffer

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	defer func() {
		os.Stderr = oldStderr
	}()

	cfg := Config{
		AppName: "sushi",
		Deploy:  "production",
	}

	logger := NewLogger(cfg)
	logger.Info("message")

	w.Close()
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "app=sushi") {
		t.Fatalf("want app=sushi in output, got: %s", output)
	}
	if !strings.Contains(output, "deploy=production") {
		t.Fatalf("want deploy=production in output, got: %s", output)
	}
}

func TestNewLoggerIncludesSpaceAndDyno(t *testing.T) {
	var buf bytes.Buffer

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	defer func() {
		os.Stderr = oldStderr
	}()

	cfg := Config{
		AppName: "sushi",
		Deploy:  "production",
		SpaceID: "space-123",
		Dyno:    "web.1",
	}

	logger := NewLogger(cfg)
	logger.Info("message")

	w.Close()
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "space=space-123") {
		t.Fatalf("want space=space-123 in output, got: %s", output)
	}
	if !strings.Contains(output, "dyno=web.1") {
		t.Fatalf("want dyno=web.1 in output, got: %s", output)
	}
}

func TestLogLevel(t *testing.T) {
	tests := []struct {
		level    string
		expected slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"error", slog.LevelError},
		{"invalid", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			got := parseLevel(tt.level)
			if got != tt.expected {
				t.Errorf("parseLevel(%q) = %v, want %v", tt.level, got, tt.expected)
			}
		})
	}
}

func TestNullLoggerDiscardsOutput(t *testing.T) {
	var buf bytes.Buffer
	logger := NewNullLogger()

	// Capture stderr to verify nothing is written
	oldStderr := slog.Default()
	defer slog.SetDefault(oldStderr)

	handler := slog.NewTextHandler(&buf, nil)
	slog.SetDefault(slog.New(handler))

	logger.Info("testing...")

	if buf.Len() > 0 {
		t.Fatalf("Expected no output but got: %s", buf.String())
	}
}
