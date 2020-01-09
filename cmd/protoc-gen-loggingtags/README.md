# protoc-gen-loggingtags

A `protoc` plugin that generates `LoggingTags() map[string]interface{}`
functions on Go proto message structs based on field options inside the source
`.proto` files. The logging tags returned are safe to log without leaking
information. The logging tag functions are code-generated so it's always in
sync with the source files.

The `grpcserver` package automatically checks if request or response messages
implement the LoggingTags method and includes the returned tags in the context
tags for logging.

## Example

```go
syntax = "proto3";
import "heroku/loggingtags/safe.proto";

package loggingtags.examples;

message Sample {
  string safe   = 1 [(heroku.loggingtags.safe) = true];
  string unsafe = 2;
}
```

When `LoggingTags()` is called on the Sample message, it will only include the
name and value of the `safe` field.

The field values are unmodified, except for protobuf Timestamp and Duration
fields, which are unpacked to return the native Go type.

Assuming `protoc-gen-loggingtags` is installed into your path, code can be
generated for the above proto definition with:

```
protoc \
  --go_out=. \
  --loggingtags_out=. \
  sample.proto
```

## Development

The `internal/test` package contains sample protobuf messages with loggingtag
annotations and a test suite to exercise them.

Run `make proto` to regenerate the loggingtags code for the sample messages
using the current local version of the generator.
