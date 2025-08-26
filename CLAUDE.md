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

### Commands

Commands are defined in `bumpers.yml` and provide helpful shortcuts and responses:

```yaml
commands:
  - name: "help"
    send: "Available commands: $help, $status"
  - name: "search"
    send: |
      Search for "{{argv 1}}" in {{if gt (argc) 1}}{{argv 2}}{{else}}codebase{{end}}:
      {{if eq (argc) 0}}Usage: $search "term" [directory]
      {{else}}grep -r "{{argv 1}}" {{if gt (argc) 1}}{{argv 2}}{{else}}.{{end}}{{end}}
```

**Command Features:**
- **Arguments**: Commands support arguments parsed from `$command arg1 "arg with spaces"`
- **Template Variables**: `{{.Name}}` (command name), `{{.Args}}` (raw args), `{{.Argv}}` (parsed array)
- **Template Functions**: `{{argc}}` (argument count), `{{argv N}}` (Nth argument, 0=command)
- **Conditional Logic**: Use `{{if}}`, `{{range}}`, and comparison functions for dynamic responses
- **AI Generation**: Same AI integration options as rules

### Rules

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

### Hook Event Configuration

### Match Field Configuration

The `match` field supports both simple string and advanced struct forms:

#### Simple Form (String)

```yaml
rules:
  - match: "rm -rf"  # Defaults: event="pre", sources=[]
    send: "Consider using safer alternatives"
```

#### Advanced Form (Struct)

```yaml
rules:
  # Match against specific tool input fields
  - match:
      pattern: "rm -rf"
      event: "pre"
      sources: ["command"]
    send: "Consider using safer alternatives"
    
  # Match against multiple fields  
  - match:
      pattern: "password"
      event: "pre"  # Optional, defaults to "pre"
      sources: ["command", "description"]
    send: "Avoid hardcoding secrets"
    
  # Match against Claude's intent (thinking + explanations)
  - match:
      pattern: "I need to.*database"
      event: "pre"
      sources: ["#intent"]
    send: "Remember to check database connections first"
    
  # Match against all available fields (default behavior)
  - match:
      pattern: "error_pattern"
      event: "post"
      sources: []  # Empty = match all fields
    send: "Error handling guidance"
```

**Event Types:**
- **`event: "pre"`** (default): Matches PreToolUse hooks - intercepts commands before execution
- **`event: "post"`**: Matches PostToolUse hooks - analyzes results after tool execution

**Sources Configuration:**
- **Pre-event rules**: Match against tool input field names + `#intent`
- **Post-event rules**: Match against tool output field names + `#intent`
- **Empty sources**: Matches all available fields

#### Post-Tool-Use Hooks

Post-tool-use hooks analyze Claude's intent and tool outputs:

```yaml
rules:
  - match:
      pattern: "error_pattern"
      event: "post"
      sources: ["#intent"]  # Match Claude's intent from transcript (thinking + explanations)
    generate: once
    send: "Helpful guidance message"
    prompt: "AI prompt for contextual analysis"
    
  - match:
      pattern: "failed"
      event: "post"
      sources: ["tool_output"]  # Match tool output/errors
    send: "Consider alternative approaches"
```

**Source Field Behavior:**

- **Pre-event rules**: Any source name matches against tool input field names, plus special `#intent` source
- **Post-event rules**: Any source name matches against tool output field names, plus special `#intent` source  
- **No validation**: Source names are not validated - any field name can be used
- **Empty sources**: When sources array is empty, matches against all available fields

**Special Meta Sources:**
- **`#intent`**: Claude's reasoning from transcript (thinking + explanations) - available for both pre and post events

**Common Tool Input Fields:**
Based on Claude Code tool implementations, common field names include:

- **`command`** - Shell commands (Bash tool)
- **`description`** - Human-readable descriptions (most tools)
- **`file_path`**, **`content`**, **`old_string`**, **`new_string`** - File operations
- **`pattern`**, **`path`**, **`glob`** - Search operations  
- **`url`**, **`method`**, **`headers`**, **`body`**, **`query`** - Web operations
- **`prompt`**, **`subagent_type`** - Task operations

**Note**: MCP servers can define arbitrary field names, so any source name is valid.

## Key Patterns

- **Value Types**: Prefer values over pointers for performance
- **Clean Error Handling**: Minimal error wrapping
- **Comprehensive Testing**: >75% coverage with TDD integration
- **Minimal Dependencies**: Avoid external deps where possible
- **Positive Messaging**: Educational, encouraging user guidance