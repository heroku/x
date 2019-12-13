package svclog

import (
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/heroku/x/testing/testlog"
)

func TestLoggerEmitsAppAndDeployData(t *testing.T) {
	cfg := Config{
		AppName: "sushi",
		Deploy:  "production",
	}
	logger := NewLogger(cfg)
	hook := test.NewLocal(logger.(*log.Entry).Logger)
	logger.Info("message")
	entry := hook.LastEntry()

	if got := entry.Data["app"]; got != "sushi" {
		t.Fatalf("want sushi, got: %s", got)
	}
	if got := entry.Data["deploy"]; got != "production" {
		t.Fatalf("want production, got: %s", got)
	}
	if got, ok := entry.Data["dyno"]; ok {
		t.Fatalf("want nothing, got dyno=%s", got)
	}
}

func TestLoggerEmitsDynoData(t *testing.T) {
	cfg := Config{
		AppName: "sushi",
		Deploy:  "production",
		Dyno:    "web.1",
	}
	logger := NewLogger(cfg)
	hook := test.NewLocal(logger.(*log.Entry).Logger)
	logger.Info("message")
	entry := hook.LastEntry()

	if got := entry.Data["dyno"]; got != "web.1" {
		t.Fatalf("want web.1, got: %s", got)
	}
}

func TestLoggerEmitsSaramaComponentData(t *testing.T) {
	cfg := Config{
		AppName: "sushi",
		Deploy:  "production",
	}
	logger := NewLogger(cfg)
	saramaLogger := SaramaLogger(logger)
	hook := test.NewLocal(logger.(*log.Entry).Logger)
	saramaLogger.Info("message")
	entry := hook.LastEntry()

	if got, ok := entry.Data["component"]; !ok {
		t.Fatalf("want component=sarama, got nothing")
	} else if got != "sarama" {
		t.Fatalf("want sarama, got: %s", got)
	}
}

func TestLoggerTrimsNewLineFromSaramaLoggerMsg(t *testing.T) {
	cfg := Config{
		AppName: "sushi",
		Deploy:  "production",
	}
	logger := NewLogger(cfg)
	saramaLogger := SaramaLogger(logger)

	hook := test.NewLocal(logger.(*log.Entry).Logger)
	msg := "message\n"
	newMsg := "message"

	saramaLogger.Printf(msg)
	entry := hook.LastEntry()

	if entry.Message != newMsg {
		t.Fatalf("wanted message with new line char removed, got %q", entry.Message)
	}
}

func TestLossyLogger(t *testing.T) {
	expectedLimit := 10
	burstWindow := time.Millisecond * 50

	logger, hook := testlog.NewNullLogger()
	sampler := NewSampleLogger(logger, expectedLimit, burstWindow)
	timer := time.NewTimer(burstWindow)

	go func() {
		for i := 0; i < 1000; i++ {
			sampler.Printf("message")
		}
	}()

	<-timer.C
	if len(hook.Entries()) != expectedLimit {
		t.Fatalf("want %d, got %d", expectedLimit, len(hook.Entries()))
	}
}
