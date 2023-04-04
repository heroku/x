# x

![CI Testing Status](https://github.com/heroku/x/workflows/ci/badge.svg)&nbsp;[![GoDoc](https://godoc.org/github.com/heroku/x?status.svg)](http://godoc.org/github.com/heroku/x)&nbsp;![Security Code Scanning - action](https://github.com/heroku/x/workflows/Code%20scanning%20-%20action/badge.svg)

A set of packages for reuse within Heroku Go applications.

## Commands

* [protoc-gen-loggingtags](./cmd/protoc-gen-loggingtags): a protoc plugin to mark which message fields are safe to log
* [s3env](./cmd/s3env): utility to manage ENV vars in an S3 bucket

## Development

The Makefile provides a few targets to help ensure the code is linted and tested.

```console
lint                           Runs golangci-lint. Override defaults with LINT_RUN_OPTS
test                           Runs go test. Override defaults with GOTEST_OPT
coverage                       Generates a coverage profile and opens a web browser with the results
proto                          Regenerate protobuf files
```
