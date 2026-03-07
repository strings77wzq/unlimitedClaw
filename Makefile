# Golem Makefile
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

BINARY  := golem
BUILD_DIR := build
CMD_DIR := ./cmd/golem

LDFLAGS := -s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.date=$(DATE)

GO_BUILD := CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)"

.PHONY: build build-all test lint deps clean fmt vet check

# Default target
build:
	$(GO_BUILD) -o $(BUILD_DIR)/$(BINARY) $(CMD_DIR)

# Cross compilation
build-all:
	GOOS=linux GOARCH=amd64 $(GO_BUILD) -o $(BUILD_DIR)/$(BINARY)-linux-amd64 $(CMD_DIR)
	GOOS=linux GOARCH=arm64 $(GO_BUILD) -o $(BUILD_DIR)/$(BINARY)-linux-arm64 $(CMD_DIR)
	GOOS=darwin GOARCH=amd64 $(GO_BUILD) -o $(BUILD_DIR)/$(BINARY)-darwin-amd64 $(CMD_DIR)
	GOOS=darwin GOARCH=arm64 $(GO_BUILD) -o $(BUILD_DIR)/$(BINARY)-darwin-arm64 $(CMD_DIR)

# Run all tests with race detection
test:
	go test ./... -v -race

# Run linter (if installed)
lint:
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run || echo "golangci-lint not installed, skipping"

# Download and verify dependencies
deps:
	go mod download
	go mod verify

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	go clean

# Format code
fmt:
	gofmt -s -w .

# Run go vet
vet:
	go vet ./...

# Full CI check: deps + fmt check + vet + test
check: deps vet test
	@echo "All checks passed!"
