package svclog

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestLoggerEmitsAppAndDeployData(t *testing.T) {
	buf := bytes.NewBuffer([]byte{})
	cfg := Config{
		AppName:  "sushi",
		Deploy:   "production",
		LogLevel: "INFO",
		Dyno:     "web.1",
		WriteTo:  buf,
	}
	logger := NewLogger(cfg)
	logger.Info("message")

	expectLogLine(t, buf, map[string]string{
		"app":    "sushi",
		"deploy": "production",
		"msg":    "message",
		"dyno":   "web.1",
	})
}

func TestReportPanic(t *testing.T) {
	buf := bytes.NewBuffer([]byte{})
	cfg := Config{
		AppName:  "sushi",
		Deploy:   "production",
		LogLevel: "INFO",
		Dyno:     "web.1",
		WriteTo:  buf,
	}
	logger := NewLogger(cfg)

	defer func() {
		if p := recover(); p == nil {
			t.Fatal("expected ReportPanic to repanic")
		}

		expectLogLine(t, buf, map[string]string{
			"msg":   "\"test message\"",
			"at":    "panic",
			"level": "ERROR",
		})
	}()

	func() {
		defer ReportPanic(logger)

		panic("test message")
	}()
}

func expectLogLine(t *testing.T, buf *bytes.Buffer, m map[string]string) {
	msg, err := buf.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}

	for k, v := range m {
		if !strings.Contains(msg, fmt.Sprintf("%s=%s", k, v)) {
			t.Errorf("expected log line to contain %s=%s", k, v)
		}
	}
}
