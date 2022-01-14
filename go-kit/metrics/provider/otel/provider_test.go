package otel

import (
	"context"
	"testing"
)

func TestNewCardinalityCounter_InsertDoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code did panic")
		}
	}()

	p, _ := New(context.Background(), "my-service")
	c := p.NewCardinalityCounter("my-counter")

	c.Insert([]byte("hey"))
}
