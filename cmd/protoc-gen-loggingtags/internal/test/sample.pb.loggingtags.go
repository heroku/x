// Code generated by protoc-gen-loggingtags. DO NOT EDIT.

package test

// LoggingTags returns loggable fields as key-value pairs.
func (r *Sample) LoggingTags() map[string]interface{} {
	if r == nil {
		return map[string]interface{}{}
	}
	return map[string]interface{}{
		"safe":      r.Safe,
		"timestamp": r.Timestamp.AsTime(),
		"duration":  r.Duration.AsDuration(),
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
