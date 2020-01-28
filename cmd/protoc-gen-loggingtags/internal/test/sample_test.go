package test

import (
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes"
)

func TestUnsafeField(t *testing.T) {
	s := &Sample{Unsafe: "secret!"}
	if _, ok := s.LoggingTags()["unsafe"]; ok {
		t.Fatal("'unsafe' field should not be present in logging tags output")
	}
}

func TestSafeField(t *testing.T) {
	want := "safe"
	s := &Sample{Safe: want}
	if got, ok := s.LoggingTags()["safe"]; !ok || got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestTimestampField(t *testing.T) {
	want := time.Now().UTC()
	ts, err := ptypes.TimestampProto(want)
	if err != nil {
		t.Fatal(err)
	}
	s := &Sample{Timestamp: ts}
	if got, ok := s.LoggingTags()["timestamp"]; !ok || got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestDurationField(t *testing.T) {
	want := time.Second * 10
	s := &Sample{Duration: ptypes.DurationProto(want)}
	if got, ok := s.LoggingTags()["duration"]; !ok || got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestNestedField(t *testing.T) {
	type loggable interface {
		LoggingTags() map[string]interface{}
	}

	want := "safe"
	s := &NestedSample{Data: &Sample{Safe: want}}
	res := s.LoggingTags()
	if _, ok := res["data"].(loggable); !ok {
		t.Fatalf("want %#v to implement loggable", res["data"])
	}
}
