package testlog

import (
	"encoding/json"
	"testing"

	"github.com/pkg/errors"
)

type LogEvent struct {
	Level   string `json:"level"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

func ToLogEvent(t *testing.T, b []byte) LogEvent {
	le := LogEvent{}
	if err := json.Unmarshal(b, &le); err != nil {
		t.Fatal(err)
	}
	return le
}

func (le LogEvent) VerifyLevel(l string) error {
	if le.Level != l {
		return errors.Errorf("expected level %s, got level %s", l, le.Level)
	}
	return nil
}

func (le LogEvent) VerifyError(l string) error {
	if le.Error != l {
		return errors.Errorf("expected level %s, got level %s", l, le.Error)
	}
	return nil
}

func (le LogEvent) VerifyMessage(l string) error {
	if le.Message != l {
		return errors.Errorf("expected level %s, got level %s", l, le.Message)
	}
	return nil
}
