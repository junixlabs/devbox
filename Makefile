BINARY    := devbox
MODULE    := github.com/junixlabs/devbox
BUILD_DIR := dist
VERSION   ?= $(shell git describe --tags 2>/dev/null || echo "0.1.0-dev")
LDFLAGS   := -s -w -X main.version=$(VERSION)

PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64

.PHONY: build release clean test test-integration vet

build:
	@mkdir -p $(BUILD_DIR)
	go build -ldflags '$(LDFLAGS)' -o $(BUILD_DIR)/$(BINARY) ./cmd/devbox/

release:
	@mkdir -p $(BUILD_DIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/} \
		go build -ldflags '$(LDFLAGS)' -o $(BUILD_DIR)/$(BINARY)-$${platform%/*}-$${platform#*/} ./cmd/devbox/ && \
		echo "Built $(BINARY)-$${platform%/*}-$${platform#*/}"; \
	done
	@cd $(BUILD_DIR) && sha256sum $(BINARY)-* > checksums.txt
	@echo "Checksums written to $(BUILD_DIR)/checksums.txt"

clean:
	rm -rf $(BUILD_DIR)

test:
	go test ./...

test-integration:
	go test -tags integration -v -timeout 300s ./tests/integration/

vet:
	go vet ./...
