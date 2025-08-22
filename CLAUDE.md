# CLAUDE.md

## Project Overview

Bumpers is a Claude Code hook guard utility that intercepts hook events and provides intelligent command guarding with positive guidance. Instead of just blocking commands, it suggests better alternatives using encouraging, educational messaging.

## CLI Commands

```bash
bumpers                    # Process hook input (main command)
bumpers install           # Install configuration and Claude hooks
bumpers hook              # Process hook events from Claude Code
bumpers status            # Show current configuration status
bumpers validate          # Validate configuration files
```

## Development Tools (justfile)

```bash
just                      # List all available commands
just build               # Build binary to bin/bumpers
just install             # Install to $GOPATH/bin
just test                # Run all tests with coverage and TDD guard
just test ./internal/cli # Test specific package
just test "" "TestName"  # Run specific test by name
just test "./..." false  # Run tests without race detection
just lint                # Run golangci-lint
just lint fix            # Run golangci-lint with auto-fix
just clean               # Remove build artifacts and coverage files
just coverage            # Show test coverage in browser
```

**Note**: Always use `just test` instead of `go test` for TDD guard integration.

## Project Structure

```
cmd/bumpers/           # CLI entry point with Cobra commands
internal/
├── cli/               # Application orchestrator and command logic
├── config/            # YAML configuration management
├── hooks/             # Hook event processing and JSON parsing
├── matcher/           # Pattern matching engine for rules
├── logger/            # Structured logging to .claude/bumpers/
├── claude/            # Claude binary detection and settings
│   └── settings/      # Claude settings.json management
├── constants/         # Shared constants and paths
├── filesystem/        # File system operations
└── project/           # Project root detection
```

## Core Architecture

1. **Hook Processing**: Parse JSON input from Claude Code hooks
2. **Rule Matching**: Check commands against YAML pattern rules
3. **Response Generation**: Provide helpful guidance for blocked commands
4. **Exit Codes**: 0 = allowed, 1 = denied (with message)

## Configuration

Rules defined in `bumpers.yml` with regex patterns:
- **pattern**: Regex to match commands (matches result in denial)
- **message**: User-friendly explanation and alternatives
- **generate**: AI tool integration ("off", "once", "session", "always")
- **prompt**: Custom prompt for AI responses

## Key Patterns

- **Value Types**: Prefer values over pointers for performance
- **Clean Error Handling**: Minimal error wrapping
- **Comprehensive Testing**: >75% coverage with TDD integration
- **Minimal Dependencies**: Avoid external deps where possible
- **Positive Messaging**: Educational, encouraging user guidance