package grpcserver

import (
	"context"
	"reflect"
	"testing"

	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
)

func TestExtractTagsWithoutLoggingTagsCompatibleValue(t *testing.T) {
	tags := grpc_ctxtags.Extract(context.Background())
	extractTags(tags, "scope", "value")
	got := tags.Values()
	want := make(map[string]interface{})
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestExtractTags(t *testing.T) {
	tags := grpc_ctxtags.Extract(context.Background())
	extractTags(tags, "scope", &value{})
	got := tags.Values()
	want := make(map[string]interface{})
	want["scope.value"] = "hello"
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestExtractTagsWithNestedValue(t *testing.T) {
	tags := grpc_ctxtags.Extract(context.Background())
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
