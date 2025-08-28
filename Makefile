# Go Code Similarity Detection Tool - Makefile

# Variables
GO_VERSION := 1.25.0
BINARY_NAME := similarity-go
MAIN_PATH := ./cmd/similarity-go
COVERAGE_OUT := coverage.out
COVERAGE_HTML := coverage.html
BIN_DIR := bin

# Default target
.PHONY: all
all: deps test build

# Install dependencies
.PHONY: deps
deps:
	go mod download
	go mod tidy

# Run tests
.PHONY: test
test:
	go test -v -race ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	go test -race -coverprofile=$(COVERAGE_OUT) ./...
	go tool cover -func=$(COVERAGE_OUT)

# Generate HTML coverage report
.PHONY: coverage-html
coverage-html: test-coverage
	go tool cover -html=$(COVERAGE_OUT) -o $(COVERAGE_HTML)
	@echo "Coverage report generated: $(COVERAGE_HTML)"

# Display coverage report
.PHONY: coverage-report
coverage-report: test-coverage
	@echo "Coverage report:"
	@go tool cover -func=$(COVERAGE_OUT) | grep "total:" | awk '{print "Total coverage: " $$3}'

# Run benchmarks
.PHONY: bench
bench:
	go test -bench=. -benchmem ./...

# Build the application
.PHONY: build
build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY_NAME) $(MAIN_PATH)

# Build for multiple platforms
.PHONY: build-all
build-all: build-linux build-darwin build-windows

.PHONY: build-linux
build-linux:
	mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(BIN_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	GOOS=linux GOARCH=arm64 go build -o $(BIN_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)

.PHONY: build-darwin
build-darwin:
	mkdir -p $(BIN_DIR)
	GOOS=darwin GOARCH=amd64 go build -o $(BIN_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 go build -o $(BIN_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)

.PHONY: build-windows
build-windows:
	mkdir -p $(BIN_DIR)
	GOOS=windows GOARCH=amd64 go build -o $(BIN_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)
	GOOS=windows GOARCH=arm64 go build -o $(BIN_DIR)/$(BINARY_NAME)-windows-arm64.exe $(MAIN_PATH)

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf $(BIN_DIR)
	rm -f $(BINARY_NAME)
	rm -f $(COVERAGE_OUT)
	rm -f $(COVERAGE_HTML)

# Format code
.PHONY: fmt
fmt:
	go fmt ./...

# Vet code
.PHONY: vet
vet:
	go vet ./...

# Run linting (requires golangci-lint)
.PHONY: lint
lint:
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not installed. Install v2.x from https://github.com/golangci/golangci-lint/releases/tag/v2.4.0"; exit 1; }
	golangci-lint run --config .golangci.yml

# Full quality check
.PHONY: quality
quality: fmt vet lint test-coverage

# Development target - quick feedback
.PHONY: dev
dev: fmt test

# CI target - comprehensive check
.PHONY: ci
ci: quality build

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all           - Run deps, test, and build"
	@echo "  deps          - Install dependencies"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage"
	@echo "  coverage-html - Generate HTML coverage report"
	@echo "  coverage-report - Display coverage report"
	@echo "  bench         - Run benchmarks"
	@echo "  build         - Build the application (to $(BIN_DIR)/)"
	@echo "  build-all     - Build for all platforms"
	@echo "  build-linux   - Build for Linux (amd64, arm64)"
	@echo "  build-darwin  - Build for macOS (amd64, arm64)"  
	@echo "  build-windows - Build for Windows (amd64, arm64)"
	@echo "  clean         - Clean build artifacts"
	@echo "  fmt           - Format code"
	@echo "  vet           - Vet code"
	@echo "  lint          - Run linting (requires golangci-lint)"
	@echo "  quality       - Run full quality check (fmt, vet, lint, test-coverage, coverage-check)"
	@echo "  dev           - Development target (fmt, test)"
	@echo "  ci            - CI target (quality, build)"
	@echo "  help          - Show this help"