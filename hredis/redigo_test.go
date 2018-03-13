package hredis

import (
	"testing"
	"time"
)

// dontWait for time to pass, we're a TARDIS or something.
func dontWait(t time.Time) error { return nil }

func TestWaitForAvailability(t *testing.T) {
	ok, err := WaitForAvailability("redis://127.0.0.1:6379", time.Millisecond, dontWait)
	if err != nil {
		t.Fatal(err)
	}

	if !ok {
		t.Fatal("expected for redis server to be available")
	}
}

func TestNewRedisPoolFromURL(t *testing.T) {
	p, err := NewRedisPoolFromURL("redis://127.0.0.1:6379")
	if err != nil {
		t.Fatal(err)
	}

	p.Close()
}
