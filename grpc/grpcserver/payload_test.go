package grpcserver

import (
	"reflect"
	"testing"

	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
)

func TestExtractTagsWithoutLoggingTagsCompatibleValue(t *testing.T) {
	tags := newTags()
	extractTags(tags, "scope", "value")
	got := tags.Values()
	want := make(map[string]interface{})
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestExtractTags(t *testing.T) {
	tags := newTags()
	extractTags(tags, "scope", &value{})
	got := tags.Values()
	want := make(map[string]interface{})
	want["scope.value"] = "hello"
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestExtractTagsWithNestedValue(t *testing.T) {
	tags := newTags()
	extractTags(tags, "scope", &nestedValue{})
	got := tags.Values()
	want := make(map[string]interface{})
	want["scope.nested.value"] = "hello"
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

type value struct{}

func (v *value) LoggingTags() map[string]interface{} {
	res := make(map[string]interface{})
	res["value"] = "hello"
	return res
}

type nestedValue struct{}

func (v *nestedValue) LoggingTags() map[string]interface{} {
	res := make(map[string]interface{})
	res["nested"] = &value{}
	return res
}

// testTags mirrors the implementation of grpc_ctxtags.Tags
type testTags struct {
	values map[string]interface{}
}

func (t *testTags) Set(key string, value interface{}) grpc_ctxtags.Tags {
	t.values[key] = value
	return t
}

func (t *testTags) Has(key string) bool {
	_, ok := t.values[key]
	return ok
}

func (t *testTags) Values() map[string]interface{} {
	return t.values
}

func newTags() grpc_ctxtags.Tags {
	return &testTags{values: make(map[string]interface{})}
}
