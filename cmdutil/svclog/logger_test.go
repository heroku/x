package svclog

import (
	"sync"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
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

type dummyOutput struct {
	calls [][]byte
	mu    sync.Mutex
}

func (d *dummyOutput) Write(p []byte) (n int, err error) {
	d.mu.Lock()
	if len(p) != 0 {
		d.calls = append(d.calls, p)
	}
	d.mu.Unlock()

	return len(p), nil
}

func (d *dummyOutput) allCalls() [][]byte {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.calls
}

func TestLossyLogger(t *testing.T) {
	expectedLimit := 10
	burstWindow := time.Millisecond * 50

	cfg := Config{
		AppName: "sushi",
		Deploy:  "production",
	}

	sampler := NewSampleLogger(NewLogger(cfg), expectedLimit, burstWindow)
	baseLogger := sampler.(*log.Entry).Logger
	hook := test.NewLocal(baseLogger)
	output := &dummyOutput{}
	baseLogger.SetOutput(output)
	timer := time.NewTimer(burstWindow)

	done := make(chan struct{})
	emitAmount := 1000
	go func() {
		defer close(done)

		for i := 0; i < emitAmount; i++ {
			sampler.Printf("message")
		}
	}()

	<-timer.C
	if want, got := expectedLimit, len(output.allCalls()); got != want {
		t.Fatalf("want %v, got %v", want, got)
	}

	<-done
	allEntries := hook.AllEntries()
	if want, got := emitAmount, len(allEntries); got != want {
		t.Fatalf("want %v, got %v", want, got)
	}

	for _, e := range allEntries {
		if want, got := "sushi", e.Data["app"]; got != want {
			t.Fatalf("want %v, got %v", want, got)
		}
		if want, got := "production", e.Data["deploy"]; got != want {
			t.Fatalf("want %v, got %v", want, got)
		}
		if want, got := true, e.Data["sampled"]; got != want {
			t.Fatalf("want %v, got %v", want, got)
		}
	}

}
