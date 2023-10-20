package otel

import (
	"context"
	"runtime/debug"
	"testing"
)

func TestNewCardinalityCounter_InsertDoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code did panic: %#v\n %s", r, string(debug.Stack()))
		}
	}()

	p, err := New(context.Background(), "my-service")
	if err != nil {
		t.Errorf("failed to start metrics provider: %s", err)
	}
	c := p.NewCardinalityCounter("my-counter")

	c.Insert([]byte("hey"))
}
