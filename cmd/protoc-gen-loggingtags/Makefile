TOP_LEVEL = $(shell git rev-parse --show-toplevel)
ARCH = $(shell uname -m)
PROTOC_VERSION = 3.11.2
PROTOC_OS = $(shell uname -s | sed 's/Darwin/osx/' | sed 's/Linux/linux/')
PROTOC_ASSET = protoc-$(PROTOC_VERSION)-$(PROTOC_OS)-$(ARCH).zip
PROTOC_DOWNLOAD_URL = https://github.com/protocolbuffers/protobuf/releases/download/v$(PROTOC_VERSION)/$(PROTOC_ASSET)

TOOLS_DIR = $(TOP_LEVEL)/.tools
TOOLS_BIN = $(TOOLS_DIR)/bin

$(TOOLS_DIR):
	mkdir -p $(TOOLS_DIR)

$(TOOLS_BIN): | $(TOOLS_DIR)
	mkdir -p $(TOOLS_BIN)

$(TOOLS_BIN)/protoc-gen-go: | $(TOOLS_BIN)
	go build -o $(TOOLS_BIN)/protoc-gen-go github.com/golang/protobuf/protoc-gen-go

$(TOOLS_BIN)/protoc: | $(TOOLS_DIR)
	curl -fsLJO $(PROTOC_DOWNLOAD_URL)
	unzip -od ${TOOLS_DIR} $(PROTOC_ASSET) -x readme.txt
	rm $(PROTOC_ASSET)

$(TOOLS_BIN)/protoc-gen-loggingtags: | $(TOOLS_BIN)
	$(TOOLS_BIN)/protoc \
		--plugin=$(TOOLS_BIN)/protoc-gen-go \
		--go_out=paths=source_relative:. \
		heroku/loggingtags/*.proto

	go build -o $(TOOLS_BIN)/protoc-gen-loggingtags .

.PHONY: proto
proto: $(TOOLS_BIN)/protoc $(TOOLS_BIN)/protoc-gen-go $(TOOLS_BIN)/protoc-gen-loggingtags
	rm internal/test/*.pb*.go || true

	$(TOOLS_BIN)/protoc \
		--plugin=$(TOOLS_BIN)/protoc-gen-go \
		--plugin=$(TOOLS_BIN)/protoc-gen-loggingtags \
		--go_out=:. \
		--loggingtags_out=. \
		internal/test/*.proto
