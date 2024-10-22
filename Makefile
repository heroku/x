TOP_LEVEL = $(shell git rev-parse --show-toplevel)
GOOS = $(shell go env GOOS)
GOARCH = $(shell go env GOARCH)
GOPATH = $(shell go env GOPATH)
TOOLS_DIR = $(TOP_LEVEL)/.tools
TOOLS_BIN = $(TOOLS_DIR)/bin
# Make sure this is in-sync with the version in the circle ci config
GOLANGCI_LINT_VERSION := v1.55.0
PKG_SPEC := ./...
MOD := -mod=readonly
GOTEST := go test $(MOD)
COVER_PROFILE = coverage.out
GOTEST_COVERAGE_OPT := -coverprofile=$(COVER_PROFILE) -covermode=atomic

# protoc config
ARCH = $(shell uname -m)
PROTOC_VERSION = 3.18.1
PROTOC_OS = $(shell uname -s | sed 's/Darwin/osx/' | sed 's/Linux/linux/')
PROTOC_ASSET = protoc-$(PROTOC_VERSION)-$(PROTOC_OS)-$(ARCH).zip
PROTOC_DOWNLOAD_URL = https://github.com/protocolbuffers/protobuf/releases/download/v$(PROTOC_VERSION)/$(PROTOC_ASSET)

# Additive or overridable variables
override GOTEST_OPT += -timeout 30s
LINT_RUN_OPTS ?= --fix
override GOMARKDOC_OPTS += --header="" --repository.url="https://github.com/heroku/x"

.DEFAULT_GOAL := precommit

$(TOOLS_DIR):
	mkdir -p $(TOOLS_DIR)

$(TOOLS_BIN): | $(TOOLS_DIR)
	mkdir -p $(TOOLS_BIN)

$(TOOLS_BIN)/protoc: | $(TOOLS_DIR)
	curl -fsLJO $(PROTOC_DOWNLOAD_URL)
	unzip -od ${TOOLS_DIR} $(PROTOC_ASSET) -x readme.txt
	rm $(PROTOC_ASSET)

$(TOOLS_BIN)/protoc-gen-go: | $(TOOLS_BIN)
	go build -o $(TOOLS_BIN)/protoc-gen-go github.com/golang/protobuf/protoc-gen-go

# Processes the circle ci config locally
$(CIRCLECI_CONFIG):
$(PROCESSED_CIRCLECI_CONFIG): $(CIRCLECI_CONFIG)
	circleci config process $(CIRCLECI_CONFIG) > $(PROCESSED_CIRCLECI_CONFIG)

.PHONY: precommit
precommit: lint test coverage 

.PHONY: lint
lint: ## Runs golangci-lint.
	docker run --rm -v $$(pwd):/app -w /app golangci/golangci-lint:$(GOLANGCI_LINT_VERSION) golangci-lint run -v

.PHONY: test
test: ## Runs go test. Override defaults with GOTEST_OPT
	$(GOTEST) $(GOTEST_OPT) $(PKG_SPEC)

.PHONY: coverage
coverage: ## Generates a coverage profile and opens a web browser with the results
	$(GOTEST) $(GOTEST_OPT) $(GOTEST_COVERAGE_OPT) $(PKG_SPEC)
	go tool cover -html=$(COVER_PROFILE)

.PHONY: proto
proto: $(TOOLS_BIN)/protoc $(TOOLS_BIN)/protoc-gen-go | $(TOOLS_BIN) ## Regenerate protobuf files
	rm loggingtags/*.pb*.go || true

	$(TOOLS_BIN)/protoc \
		--plugin=$(TOOLS_BIN)/protoc-gen-go \
		--go_out=paths=source_relative:. \
		loggingtags/*.proto

	go build -o $(TOOLS_BIN)/protoc-gen-loggingtags ./cmd/protoc-gen-loggingtags

	rm ./cmd/protoc-gen-loggingtags/internal/test/*.pb*.go || true
	$(TOOLS_BIN)/protoc \
		--plugin=$(TOOLS_BIN)/protoc-gen-go \
		--plugin=$(TOOLS_BIN)/protoc-gen-loggingtags \
		--go_out=paths=source_relative:. \
		--loggingtags_out=. \
		./cmd/protoc-gen-loggingtags/internal/test/*.proto

$(GOPATH)/bin/gomarkdoc:
	go install github.com/princjef/gomarkdoc/cmd/gomarkdoc@latest

.PHONY: docs
docs: $(GOPATH)/bin/gomarkdoc ## Generate docs using gomarkdoc
	$< $(GOMARKDOC_OPTS) -o ./dynoid/README.md -e ./dynoid/...

.PHONY: verify-docs
verify-docs: $(GOPATH)/bin/gomarkdoc
	@cp ./dynoid/README.md ./dynoid/README.md.orig
	@$< $(GOMARKDOC_OPTS) -o ./dynoid/README.md -e ./dynoid/...
	@if ! cmp ./dynoid/README.md ./dynoid/README.md.orig; then printf "docs not generated\n" >&2; false; fi
