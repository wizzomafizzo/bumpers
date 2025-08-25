# CLAUDE.md

## Project Overview

Bumpers is a Claude Code hook guard utility that intercepts hook events and provides intelligent command guarding with positive guidance. Instead of just blocking commands, it suggests better alternatives using encouraging, educational messaging with optional AI-powered responses.

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
just test                # Run all tests (unit + integration + e2e)
just test-unit           # Run unit tests only (fastest)
just test-integration    # Run integration tests
just test-e2e            # Run end-to-end tests
just test ./internal/cli # Test specific package
just lint                # Run golangci-lint
just lint fix            # Run golangci-lint with auto-fix
just clean               # Remove build artifacts and coverage files
just coverage            # Show test coverage in browser
```

## Testing & TDD Workflow

**CRITICAL**: This project follows strict TDD practices. See `TESTING.md` for comprehensive testing guide.

- **Always use `just test-*` commands** - Never use `go test` directly
- **Three test categories**: Unit (fast), Integration (with dependencies), E2E (full system)
- **Test utilities**: Custom assertions, mock helpers, resource management
- **Claude AI mocks**: Comprehensive test helpers for AI interactions
- **Target coverage**: >75% with mutation testing for quality validation

## Project Structure

```
cmd/bumpers/           # CLI entry point with Cobra commands
internal/
├── ai/                # AI generation with caching and rate limiting
├── cli/               # Application orchestrator and command logic
├── config/            # YAML configuration management
├── hooks/             # Hook event processing and JSON parsing
├── matcher/           # Pattern matching engine for rules
├── logger/            # Structured logging to .claude/bumpers/
├── claude/            # Claude binary detection and settings
│   └── settings/      # Claude settings.json management
├── constants/         # Shared constants and paths
├── context/           # Template context for messaging
├── filesystem/        # File system operations
├── project/           # Project root detection
├── storage/           # Persistent storage for session data
├── template/          # Template engine with custom functions
└── testutil/          # Testing utilities and assertions
```

## Core Architecture

1. **Hook Processing**: Parse JSON input from Claude Code hooks
2. **Rule Matching**: Check commands against YAML pattern rules
3. **Response Generation**: Provide helpful guidance for blocked commands
4. **Exit Codes**: 0 = allowed, 1 = denied (with message)

## Configuration

Rules defined in `bumpers.yml` with regex patterns:
- **pattern**: Regex to match commands (matches result in denial)
- **tools**: Regex to match tool names (optional, defaults to "^Bash$" if empty)
- **message**: User-friendly explanation and alternatives
- **generate**: AI tool integration ("off", "once", "session", "always")
- **prompt**: Custom prompt for AI responses

### Tool Matching

The `tools` field allows rules to target specific Claude Code tools:

```yaml
rules:
  # Rule applies only to Bash commands (default behavior when tools is empty)
  - pattern: "^go test"
    tools: "^Bash$"
    message: "Use 'just test' instead"
    
  # Rule applies to multiple tools using regex alternation
  - pattern: " /tmp"
    tools: "^(Bash|Task)$"
    message: "Use project 'tmp' directory instead"
    
  # Rule applies only to file editing tools
  - pattern: "password"
    tools: "^(Write|Edit|MultiEdit)$"
    message: "Avoid hardcoding secrets in files"
    
  # Rule applies to all tools (empty tools field defaults to Bash only)
  - pattern: "help"
    tools: ".*"
    message: "Use built-in help system"
```

**Common Tool Names:** Bash, Write, Edit, MultiEdit, Read, Task, Glob, Grep, WebFetch, WebSearch

**Case-Insensitive Matching:** Tool patterns are matched case-insensitively, so `bash` matches `Bash`.

**Default Behavior:** Rules without a `tools` field only match Bash commands (backward compatibility).

## Key Patterns

- **Value Types**: Prefer values over pointers for performance
- **Clean Error Handling**: Minimal error wrapping
- **Comprehensive Testing**: >75% coverage with TDD integration
- **Minimal Dependencies**: Avoid external deps where possible
- **Positive Messaging**: Educational, encouraging user guidance