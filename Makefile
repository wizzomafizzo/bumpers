.PHONY: all build test lint lint-fix clean install help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test

# Package parameter for targeting specific directories
PKG ?= ./...

# TDD Guard detection and setup
TDDGUARD_AVAILABLE := $(shell command -v tdd-guard-go 2> /dev/null)
PROJECT_ROOT := $(PWD)

# Race flag setup - race detector requires CGO, so disable for cross-compilation
ifeq ($(GOOS),)
	# Native build - use race detection
	RACE_FLAG = -race
else
	# Cross-compilation - skip race detection (requires CGO)
	RACE_FLAG = 
endif

# Conditional test command - pipes through tdd-guard-go if available
ifdef TDDGUARD_AVAILABLE
	GOTEST_WITH_TDD = $(GOTEST) -json $(PKG) 2>&1 | tdd-guard-go -project-root $(PROJECT_ROOT)
else
	GOTEST_WITH_TDD = $(GOTEST) $(PKG)
endif

# Default target
all: lint test build

# Build the bumpers binary
build:
	@echo "Building bumpers..."
	mkdir -p bin
	$(GOBUILD) -o bin/bumpers ./cmd/bumpers

# Run all tests
test:
	@echo "Running tests on $(PKG)..."
ifdef TDDGUARD_AVAILABLE
	@echo "TDD Guard detected - integrating test reporting..."
	$(GOTEST) -json -v $(RACE_FLAG) -coverprofile=coverage.txt -covermode=atomic $(PKG) 2>&1 | tdd-guard-go -project-root $(PROJECT_ROOT)
else
	$(GOTEST) -v $(RACE_FLAG) -coverprofile=coverage.txt -covermode=atomic $(PKG)
endif

# Install the bumpers binary
install:
	@echo "Installing bumpers..."
	$(GOCMD) install ./cmd/bumpers

# Run linters (includes formatting)
lint:
	@echo "Running linters..."
	$(GOCMD) mod tidy
	golangci-lint run ./...

# Run linters with auto-fix
lint-fix:
	@echo "Running linters with auto-fix..."
	$(GOCMD) mod tidy
	golangci-lint run --fix ./...


# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCMD) clean
	rm -f coverage.txt
	rm -rf bin/

# Quick check before committing
check: lint test
	@echo "All checks passed!"

# Show help
help:
	@echo "bumpers Makefile"
	@echo "================="
	@echo ""
	@echo "Available targets:"
	@echo "  all                 - Lint, test, and build (default)"
	@echo "  build               - Build bumpers binary to bin/bumpers"
	@echo "  install             - Install bumpers binary to \$$GOPATH/bin"
	@echo "  test                - Run all tests"
	@echo "  lint                - Format code and run linters (golangci-lint)"
	@echo "  lint-fix            - Run linters with auto-fix (golangci-lint --fix)"
	@echo "  clean               - Remove build artifacts"
	@echo "  check               - Run lint and test (pre-commit check)"
	@echo "  help                - Show this help message"
	@echo ""
	@echo "Package targeting (PKG parameter):"
	@echo "  PKG=./...           - Test all packages (default)"
	@echo "  PKG=./internal/...  - Test internal packages"
	@echo "  PKG=./cmd/bumpers   - Test CLI package"
	@echo ""
	@echo "Examples:"
	@echo "  make test PKG=./internal/config    - Test config package only"
	@echo "  make test PKG=./internal/cli       - Test CLI package only"
	@echo ""
	@echo "Note: Test commands automatically integrate with tdd-guard-go if available"