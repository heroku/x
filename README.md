# x

[![CircleCI](https://circleci.com/gh/heroku/x.svg?style=svg)](https://circleci.com/gh/heroku/x)&nbsp;
[![GoDoc](http://godoc.org/badge.png)](http://godoc.org/github.com/heroku/x)

A set of packages for reuse within Heroku Go applications.

## Development

The Makefile provides a few targets to help ensure the code is linted and tested.

```console
$ make help
target                         help
------                         ----
lint                           Runs golangci-lint. Override defaults with LINT_RUN_OPTS
test                           Runs go test. Override defaults with GOTEST_OPT
coverage                       Generates a coverage profile and opens a web browser with the results
ci-lint                        Runs the ci based lint job locally.
ci-test                        Runs the ci based test job locally
ci-coverage                    Runs the ci based coverage job locally

'ci-' targets require the CircleCI cli tool: https://circleci.com/docs/2.0/local-cli/
```
