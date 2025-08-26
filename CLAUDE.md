# CLAUDE.md

## Project Overview

Bumpers is a Claude Code hook guard utility that intercepts hook events and provides intelligent command guarding with positive guidance. Instead of just blocking commands, it suggests better alternatives using encouraging, educational messaging with optional AI-powered responses.

## Development Workflow

### Commands
```bash
just                      # List all available commands
just build               # Build binary to bin/bumpers
just test                # Run all tests (unit + integration + e2e)
just test-unit           # Run unit tests only (fastest)
just lint                # Run golangci-lint
just lint fix            # Run golangci-lint with auto-fix
```

### Testing & TDD Requirements

**CRITICAL**: This project follows strict TDD practices:

- **Always use `just test-*` commands** - Never use `go test` directly
- **Three test categories**: Unit (fast), Integration (with dependencies), E2E (full system)
- **Target coverage**: >75% with mutation testing for quality validation
- **Write tests first, then implementation**

See `TESTING.md` for comprehensive testing guide.

## Project Structure

```
cmd/bumpers/           # CLI entry point with Cobra commands
internal/
├── ai/                # AI generation with caching and rate limiting
├── cli/               # Application orchestrator and command logic
├── config/            # YAML configuration management
├── hooks/             # Hook event processing and JSON parsing
├── matcher/           # Pattern matching engine for rules
├── logger/            # Structured logging
├── claude/            # Claude binary detection and settings
├── template/          # Template engine with custom functions
└── testutil/          # Testing utilities and assertions
```

## Core Architecture

1. **Hook Processing**: Parse JSON input from Claude Code hooks
2. **Rule Matching**: Check commands against YAML pattern rules with pre/post events
3. **Response Generation**: Provide helpful guidance with optional AI enhancement
4. **Exit Codes**: 0 = allowed, 1 = denied (with message)

## Configuration System

Configuration is defined in `bumpers.yml` with three main sections:

- **Rules**: Pattern matching for tool usage with pre/post hook events
- **Commands**: Custom `$command` shortcuts with argument parsing
- **Session**: Context injection at session start

**For detailed configuration options, see:**
- `docs/configuration.md` - Complete YAML reference
- `docs/hooks.md` - Hook events and rule matching
- `docs/templates.md` - Template system with variables and functions
- `docs/examples.md` - Real-world configuration patterns

## Development Guidelines

### Code Quality
- **Value types over pointers** for performance
- **Clean error handling** with minimal wrapping
- **Minimal dependencies** - avoid external deps where possible
- **Positive messaging** - educational, encouraging user guidance

### Architecture Patterns
- **Interface-based design** for testability
- **Dependency injection** for clean separation
- **Resource cleanup** with proper defer usage
- **Context propagation** for cancellation and timeouts