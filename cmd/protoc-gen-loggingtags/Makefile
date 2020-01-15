TOP_LEVEL = $(shell git rev-parse --show-toplevel)

TOOLS_DIR = $(TOP_LEVEL)/.tools
TOOLS_BIN = $(TOOLS_DIR)/bin

$(TOOLS_BIN)/protoc-gen-go: | $(TOOLS_BIN)
	go build -o $(TOOLS_BIN)/protoc-gen-go github.com/golang/protobuf/protoc-gen-go

.PHONY: generator
generator: $(TOOLS_BIN)/protoc $(TOOLS_BIN)/protoc-gen-go | $(TOOLS_BIN)
	$(TOOLS_BIN)/protoc \
		--plugin=$(TOOLS_BIN)/protoc-gen-go \
		--go_out=paths=source_relative:. \
		heroku/loggingtags/*.proto

	go build -o $(TOOLS_BIN)/protoc-gen-loggingtags .

.PHONY: proto
proto: $(TOOLS_BIN)/protoc $(TOOLS_BIN)/protoc-gen-go generator
	rm internal/test/*.pb*.go || true

	$(TOOLS_BIN)/protoc \
		--plugin=$(TOOLS_BIN)/protoc-gen-go \
		--plugin=$(TOOLS_BIN)/protoc-gen-loggingtags \
		--go_out=:. \
		--loggingtags_out=. \
		internal/test/*.proto
