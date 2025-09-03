# bumpers justfile

set shell := ["bash", "-c"]

alias t := test
alias l := lint  
alias b := build
alias i := install

# List all recipes
default:
    @just --list

# Build the bumpers binary
build:
    mkdir -p bin
    go build -o bin/bumpers ./cmd/bumpers

# Run go test with TDD Guard if available
test *args:
    #!/usr/bin/env bash
    set -euo pipefail
    if command -v tdd-guard-go >/dev/null 2>&1; then
        go test -json {{args}} 2>&1 | tdd-guard-go -project-root {{justfile_directory()}}
    else
        go test {{args}}
    fi

# Run unit tests
test-unit *args="":
    #!/usr/bin/env bash
    if [ "{{args}}" = "" ]; then
        just test -race ./cmd/... ./internal/...
    else
        just test -race {{args}}
    fi

# Run integration tests
test-integration *args="":
    #!/usr/bin/env bash
    if [ "{{args}}" = "" ]; then
        just test -race -tags=integration ./cmd/... ./internal/...
    else
        just test -race {{args}} -tags=integration
    fi

# Run end-to-end tests
test-e2e *args="":
    #!/usr/bin/env bash
    if [ "{{args}}" = "" ]; then
        just test -race -tags=e2e ./cmd/... ./internal/...
    else
        just test -race {{args}} -tags=e2e
    fi

# Run all test categories  
test-all *args:
    just test-unit {{args}}
    just test-integration {{args}}
    just test-e2e {{args}}

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

# Generate coverage report and open in browser
coverage:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Generating coverage report..."
    go test -race -coverprofile=coverage.out ./cmd/... ./internal/...
    go tool cover -html=coverage.out -o coverage.html
    if command -v xdg-open >/dev/null 2>&1; then
        xdg-open coverage.html
    elif command -v open >/dev/null 2>&1; then
        open coverage.html
    else
        echo "Coverage report generated: coverage.html"
    fi

# Generate per-package coverage reports
coverage-by-package:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Generating per-package coverage reports..."
    mkdir -p coverage
    
    # Get list of packages with tests
    packages=$(go list ./cmd/... ./internal/... | grep -v "/testutil$")
    
    for pkg in $packages; do
        pkg_name=$(basename "$pkg")
        echo "Coverage for $pkg..."
        if go test -race -coverprofile="coverage/${pkg_name}.out" "$pkg" 2>/dev/null; then
            if [ -f "coverage-reports/${pkg_name}.out" ]; then
                coverage=$(go tool cover -func="coverage/${pkg_name}.out" | grep total | awk '{print $3}')
                echo "  $pkg: $coverage"
            fi
        else
            echo "  $pkg: no tests or test failed"
        fi
    done
    
    echo ""
    echo "Summary report:"
    for pkg in $packages; do
        pkg_name=$(basename "$pkg")
        if [ -f "coverage/${pkg_name}.out" ]; then
            if coverage=$(go tool cover -func="coverage/${pkg_name}.out" | grep total | awk '{print $3}'); then
                printf "%-30s %s\n" "$pkg" "$coverage"
            else
                printf "%-30s %s\n" "$pkg" "error reading coverage"
            fi
        else
            printf "%-30s %s\n" "$pkg" "no coverage data"
        fi
    done

# Clean build and test artifacts
clean:
    go clean
    go clean -testcache
    rm -f coverage.txt coverage-*.txt coverage.out coverage.html
    rm -f report.json
    rm -rf bin/ coverage/
