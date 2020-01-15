# x

[![CircleCI](https://circleci.com/gh/heroku/x.svg?style=svg)](https://circleci.com/gh/heroku/x)&nbsp;[![GoDoc](https://godoc.org/github.com/heroku/x?status.svg)](http://godoc.org/github.com/heroku/x)

A set of packages for reuse within Heroku Go applications.

## Commands

* [protoc-gen-loggingtags](./cmd/protoc-gen-loggingtags): a protoc plugin to mark which message fields are safe to log
* [s3env](./cmd/s3env): utility to manage ENV vars in an S3 bucket

## Development

The Makefile provides a few targets to help ensure the code is linted and tested.

```console
$ make help
target                         help
------                         ----
lint                           Runs golangci-lint. Override defaults with LINT_RUN_OPTS
test                           Runs go test. Override defaults with GOTEST_OPT
coverage                       Generates a coverage profile and opens a web browser with the results
proto                          Regenerate protobuf files
ci-lint                        Runs the ci based lint job locally.
ci-test                        Runs the ci based test job locally
ci-coverage                    Runs the ci based coverage job locally

'ci-' targets require the CircleCI cli tool: https://circleci.com/docs/2.0/local-cli/
```
