// Code generated by protoc-gen-loggingtags. DO NOT EDIT.

package test

import (
	"time"

	"github.com/golang/protobuf/ptypes"
	dpb "github.com/golang/protobuf/ptypes/duration"
	tspb "github.com/golang/protobuf/ptypes/timestamp"
)

// LoggingTags returns loggable fields as key-value pairs.
func (r *Sample) LoggingTags() map[string]interface{} {
	if r == nil {
		return map[string]interface{}{}
	}
	return map[string]interface{}{
		"safe":      r.Safe,
		"timestamp": loggingTagsTimestamp(r.Timestamp),
		"duration":  loggingTagsDuration(r.Duration),
		"with_case": r.WithCase,
		"opt_safe":  r.OptSafe,
	}
}

// LoggingTags returns loggable fields as key-value pairs.
func (r *NestedSample) LoggingTags() map[string]interface{} {
	if r == nil {
		return map[string]interface{}{}
	}
	return map[string]interface{}{
		"data": r.Data,
	}
}

func loggingTagsTimestamp(ts *tspb.Timestamp) time.Time {
	t, _ := ptypes.Timestamp(ts)
	return t
}

func loggingTagsDuration(dur *dpb.Duration) time.Duration {
	d, _ := ptypes.Duration(dur)
	return d
}
