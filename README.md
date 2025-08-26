# Bumpers

Bumpers is a CLI utility to manage and enforce guard rails for Claude Code. Tearing your hair out because Claude won't remember what's in CLAUDE.md? Use Bumpers to make Claude love following your rules.

![Bumpers blocking a tool in Claude Code](example.png)

## Features

- **Flexible Rule Matching**: Match against tool inputs, outputs, or Claude's reasoning
- **Pre & Post Hook Events**: Intercept before execution or analyze after completion
- **AI-Enhanced Responses**: Optional AI-generated contextual guidance  
- **Template System**: Dynamic responses with variables, functions, and file access
- **Custom Commands**: `$command` syntax with argument parsing and templates
- **Session Context**: Inject project-specific context at session start
- **Tool Filtering**: Target specific Claude Code tools (Bash, Write, Edit, etc.)
- **Shareable**: Project-based configuration committed to your repo

## How It Works

If you use Claude Code, you know it can be bad at using tools consistently and tends to work around rules. The most common advice is to give it a list of guidelines to follow in the CLAUDE.md file, with a lot of conflicting advice on how to structure it, and with very inconsistent results.

Bumpers addresses this with a few methods:

1. **Deterministic Rules**: Regex patterns match consistently - no gambling on whether they work
2. **Context Separation**: Your "don'ts" are no longer in Claude's context where they can backfire
3. **Positive Messaging**: Helpful alternatives instead of just blocking, with optional AI enhancement
4. **Multi-Stage Interception**: Catch issues before execution (pre-hook) or provide guidance after (post-hook)

In combination, these methods aren't perfect, but they catch the vast majority of mistakes and keep Claude working without interruption or mistakes. **Bumpers is intended as a productivity tool, not a security tool.**

### Example: Test Command Enforcement

In many projects, I use TDD workflows that require specific test commands. Claude almost always tries to run `go test` directly, bypassing the workflow.

Now I use Bumpers with a rule like this:

```yaml
rules:
  - match: "go test"
    send: |
      Use "just test" instead for TDD integration:
      • just test           # All test categories
      • just test-unit      # Unit tests only  
      • just test ./pkg     # Specific package
```

Problem solved. Claude will try once per session, trigger the rule with helpful guidance, and then follow the correct approach.

## Installation

```shell
go install github.com/wizzomafizzo/bumpers/cmd/bumpers@latest
```

## Quick Start

1. **Install and setup:**
   ```bash
   bumpers install
   ```
   This creates `bumpers.yml` and configures Claude Code hooks.

2. **Restart Claude Code** if it's currently running.

3. **Edit `bumpers.yml`** to add your rules:
   ```yaml
   rules:
     - match: "dangerous_command"
       send: "Use safer alternative: safe_command"
   
   commands:  
     - name: "help"
       send: "Project shortcuts: test, lint, deploy"
   
   session:
     - add: "Remember to run tests before committing"
   ```

4. **Rules are live** - no restart needed when editing config.

## Configuration Overview

### Rules
Match and redirect tool usage:
```yaml
rules:
  # Simple pattern matching
  - match: "rm -rf"
    send: "Use specific file deletion instead"
    
  # Advanced matching with event and source targeting
  - match:
      pattern: "error|failed"
      event: "post"          # Analyze after tool execution
      sources: ["tool_output", "#intent"]  # Match outputs and Claude's reasoning
    send: "Debug the error and try alternative approaches"
    generate: "session"      # AI-enhanced response
```

### Commands  
Custom `$command` shortcuts:
```yaml
commands:
  - name: "test"
    send: 'Run "just test" to execute all test suites'
    
  - name: "search"
    send: |
      {{if eq (argc) 0}}Usage: $search "term" [directory]
      {{else}}grep -r "{{argv 1}}" {{if gt (argc) 1}}{{argv 2}}{{else}}.{{end}}{{end}}
```

### Session Context
Inject information at session start:
```yaml
session:
  - add: "Today's date: {{.Today}}"
  - add: "Project uses TDD - write tests first!"
```

## Documentation

- **[Configuration Reference](docs/configuration.md)** - Complete YAML configuration guide
- **[Hook Events](docs/hooks.md)** - Pre/post hook events and rule matching  
- **[Template System](docs/templates.md)** - Dynamic templates with variables and functions
- **[AI Generation](docs/ai-generation.md)** - AI-enhanced responses and caching
- **[Examples](docs/examples.md)** - Real-world configuration patterns
- **[CLI Commands](docs/cli.md)** - Command-line usage and debugging

## Key Concepts

### Hook Events
- **PreToolUse** (default): Block before execution
- **PostToolUse**: Analyze results after execution  
- **UserPromptSubmit**: Handle `$command` syntax
- **SessionStart**: Inject context at session beginning

### Match Sources
Rules can target specific fields:
- **Tool inputs**: `command`, `description`, `file_path`, `content`
- **Tool outputs**: `tool_output`, `error`, `exit_code`
- **Special**: `#intent` matches Claude's reasoning from conversation

### AI Generation
Enhance static messages with AI:
- **`off`**: Template only (fast)
- **`once`**: Generate once, cache permanently  
- **`session`**: Cache per session
- **`always`**: Generate every time (slow)

## Examples

Check [docs/examples.md](docs/examples.md) for comprehensive configuration patterns, or see the current project's `bumpers.yml` for real-world usage.
