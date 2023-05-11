package svclog

import (
	"bytes"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
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

func TestReportPanic(t *testing.T) {
	logger, hook := testlog.New()

	defer func() {
		if p := recover(); p == nil {
			t.Fatal("expected ReportPanic to repanic")
		}

		entries := hook.Entries()
		if want, got := 1, len(entries); want != got {
			t.Fatalf("want hook entries to be %d, got %d", want, got)
		}
		if want, got := "test message", entries[0].Message; want != got {
			t.Errorf("want hook entry message to be %q, got %q", want, got)
		}
	}()

	func() {
		defer ReportPanic(logger)

		panic("test message")
	}()
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
	// we give 10% wiggle room in this test because we don't need the sample logger to be perfectly serialized
	if want, got := expectedLimit, len(output.allCalls()); !((got >= want-1) && (got <= want+1)) {
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

func TestNullLoggerSwallowsLogs(t *testing.T) {
	out, err := captureOutput(func() {
		l := NewNullLogger()
		l.Info("testing...")
	})

	if err != nil {
		t.Fatal(err)
	}

	if out != "" {
		t.Fatalf("Expected no output but got %v", out)
	}
}

func TestLoggerOrNull(t *testing.T) {
	ts := map[string]struct {
		logger func() log.FieldLogger
		msg    string
		want   []string
	}{
		"new": {
			logger: func() log.FieldLogger { return log.New() },
			msg:    "testing...",
			want:   []string{"testing..."},
		},
		"nil": {
			logger: func() log.FieldLogger { return nil },
			msg:    "testing...",
			want:   []string{""},
		},
	}

	for name, tc := range ts {
		t.Run(name, func(t *testing.T) {
			out, err := captureOutput(func() {
				log := tc.logger()
				log = LoggerOrNull(log)
				log.Info(tc.msg)
			})

			if err != nil {
				t.Fatal(err)
			}

			if len(tc.want) == 0 && len(out) != 0 {
				t.Fatalf("Didn't expect any output but got `%v`", out)
			}

			for _, w := range tc.want {
				if !strings.Contains(out, w) {
					t.Fatalf("Expected `%v` to contain `%v`", out, w)
				}
			}
		})
	}
}

func captureOutput(f func()) (string, error) {
	r, w, err := os.Pipe()
	if err != nil {
		return "", errors.Wrap(err, "Error capturing output")
	}

	stdout := os.Stdout
	stderr := os.Stderr
	defer func() {
		os.Stdout = stdout
		os.Stderr = stderr
	}()
	os.Stdout = w
	os.Stderr = w

	errs := make(chan error)
	out := make(chan string)
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		var buf bytes.Buffer
		wg.Done()
		if _, err := io.Copy(&buf, r); err != nil {
			errs <- errors.Wrap(err, "Error copying output to buffer")
		}
		out <- buf.String()
	}()

	wg.Wait()
	f()
	w.Close()
	close(errs)

	return <-out, <-errs
}
