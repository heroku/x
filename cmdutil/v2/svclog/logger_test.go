package svclog

import (
	"bytes"
	"testing"

	"github.com/heroku/x/testing/v2/testlog"
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

	testlog.ExpectLogLineFromReader(t, buf, "", map[string]interface{}{
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

		testlog.ExpectLogLineFromReader(t, buf, "", map[string]interface{}{
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
