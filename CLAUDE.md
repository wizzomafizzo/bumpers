# CLAUDE.md

## Project Overview

Bumpers is a Claude Code hook guard utility that intercepts hook events and provides intelligent command guarding with positive guidance. Instead of just blocking commands, it suggests better alternatives using encouraging, educational messaging.

## Core Commands

### CLI Usage
```bash
bumpers                    # Process hook input (main command)
bumpers install           # Install configuration and Claude hooks
bumpers test [command]    # Test a command against current rules
bumpers claude backup     # Backup Claude settings.json
bumpers claude restore    # Restore Claude settings from backup
```

### Build & Development
```bash
just build          # Build binary to bin/bumpers
just install        # Install to $GOPATH/bin
just                # Lint, test, and build (default)
```

### Testing
```bash
just test                           # Run all tests with coverage
just test ./internal/config        # Test specific package
just test ./internal/cli           # Test CLI package only
just test ./... false              # Run all tests without race detection
just test ./internal/config true   # Test specific package with race detection
```

**IMPORTANT**: Always use `just test` instead of `go test` to ensure proper TDD guard integration and consistent test execution.

### Code Quality
```bash
just lint           # Run golangci-lint
just lint fix       # Run golangci-lint with auto-fix
```

### Clean Up
```bash
just clean          # Remove build artifacts, coverage files, and bin/
```

## Architecture

### Package Structure

The codebase follows standard Go project layout with clear separation of concerns:

- **cmd/bumpers/** - Minimal CLI entry point using Cobra framework. Contains only command definitions and basic orchestration.

- **internal/cli/** - Application orchestrator that coordinates between all internal packages. Handles the main business logic flow.

- **internal/config/** - YAML configuration management. Loads and validates bumpers.yaml files containing rule definitions.

- **internal/hooks/** - Hook event processing. Parses JSON input from Claude Code hooks and extracts command information.

- **internal/matcher/** - Pattern matching engine using Go regex. Matches commands against configured rules.


- **internal/logger/** - Structured logging to .claude/bumpers/bumpers.log using slog with JSON format.

- **internal/claude/** - Claude binary detection and execution:
  - **launcher.go** - Auto-detects Claude binary location with smart fallback
  - **settings/** - Claude settings.json management for hook configuration

- **configs/** - Embedded default configuration (default-bumpers.yaml) for installation.

### Core Flow

1. **Hook Input**: Claude Code sends JSON to stdin when a hook is triggered
2. **Command Extraction**: `hooks.ParseInput()` extracts the command
3. **Rule Matching**: `matcher.RuleMatcher` checks command against YAML rules
4. **Response Generation**: If denied, creates helpful guidance
5. **Exit Codes**: 0 = allowed, 1 = denied (with message to stdout)

### Configuration System

Rules are defined in YAML with regex patterns:
- **pattern**: Regex to match commands (any match results in denial)
- **response**: User-friendly explanation and alternatives
- **use_claude**: Enable Claude CLI integration for dynamic responses
- **prompt**: Custom prompt for Claude when use_claude is enabled

Installation creates bumpers.yaml and configures .claude/settings.local.json hooks automatically.

### Testing Approach

The project uses strict TDD with comprehensive test coverage:
- Unit tests for all packages in `*_test.go` files
- Integration test in `internal/bumpers_test.go`
- Tests automatically integrate with tdd-guard-go if available
- Coverage target: >75% for all internal packages

## Key Patterns

- **Value Types**: Uses value types instead of pointers for better performance
- **Error Handling**: Clean error propagation without excessive wrapping
- **Minimal CLI**: Cobra commands are thin wrappers around internal/cli
- **Thread Safety**: Launcher uses mutex for cached path management
- **Configuration**: YAML-based with validation on load

## Development Notes

- Lefthook is configured for pre-commit hooks (lint, go mod tidy)
- golangci-lint is used for comprehensive linting
- The project avoids external dependencies where possible (no Viper, minimal deps)
- Focus on positive, educational messaging in all user-facing output