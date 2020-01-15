// protoc-gen-loggingtags is a plugin for protoc which allows message fields to
// be annotated as safe to log.
//
// A `LoggingTags() map[string]interface{}` method will be generated for any
// message with a field marked as safe to log. For example, the following proto
// file:
//
//		syntax = "proto3";
//		import "github.com/heroku/x/loggingtags/safe.proto";
//		
//		package loggingtags.examples;
//		
//		message Sample {
//		  string safe   = 1 [(heroku.loggingtags.safe) = true];
//		  string unsafe = 2;
//		}
//
// will generate a LoggingTags() method on Sample which returns only the name
// and value of the `safe` field.
//
// The gRPC utilities here in heroku/x natively support this interface and will
// include safe fields on the request and response in logs and error reports.
package main
