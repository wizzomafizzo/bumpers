# bumpers justfile

set shell := ["bash", "-c"]

alias t := test
alias l := lint  
alias b := build

# List all recipes
default:
    @just --list

# Build the bumpers binary
build:
    mkdir -p bin
    go build -o bin/bumpers ./cmd/bumpers

# Run tests with optional package targeting and race flag
test package="./..." race="auto":
    #!/usr/bin/env bash
    set -euo pipefail

    case "{{race}}" in
        true)  RACE_FLAG="-race" ;;
        false) RACE_FLAG="" ;;
        auto)  [[ -z "${GOOS:-}" ]] && RACE_FLAG="-race" || RACE_FLAG="" ;;
        *)     echo "Invalid race option: {{race}}" >&2; exit 1 ;;
    esac

    # Run tests with or without TDD Guard
    if command -v tdd-guard-go >/dev/null 2>&1; then
        go test -json -v $RACE_FLAG -coverprofile=coverage.txt -covermode=atomic {{package}} 2>&1 | \
            tdd-guard-go -project-root {{justfile_directory()}}
    else
        go test -v $RACE_FLAG -coverprofile=coverage.txt -covermode=atomic {{package}}
    fi

# Install the bumpers binary
install:
    go install ./cmd/bumpers

# Run linters with optional auto-fix
lint fix="":
    #!/usr/bin/env bash
    set -euo pipefail
    go mod tidy
    if [ "{{fix}}" = "fix" ]; then
        golangci-lint run --fix ./...
    else
        golangci-lint run ./...
    fi

# Clean build artifacts
clean:
    go clean
    rm -f coverage.txt
    rm -rf bin/

# Show test coverage
coverage: test
    go tool cover -html=coverage.txt
