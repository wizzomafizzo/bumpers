# Hook Events Guide

Bumpers integrates with Claude Code through hook events that intercept tool usage.

## Hook Types

### PreToolUse Hook (Default)
Intercepts tool usage before execution:

```yaml
rules:
  - match: "rm -rf"
    send: "Use safer alternatives"
```

- **Exit codes**: 0 (allow), 2 (block)
- **Use cases**: Block dangerous commands, enforce standards

### PostToolUse Hook  
Analyzes results after execution:

```yaml
rules:
  - match:
      pattern: "permission denied"
      event: "post"
      sources: ["tool_output"]
    send: "Check file permissions"
```

- **Behavior**: Always informational
- **Use cases**: Error analysis, follow-up suggestions

### UserPromptSubmit Hook
Handles `$command` syntax:

```yaml
commands:
  - name: "test"
    send: 'Run "just test"'
```

### SessionStart Hook
Injects context at session start:

```yaml
session:
  - add: "Today's date: {{.Today}}"
```

## Event Configuration

### Match Sources

Target specific fields using `sources`:

```yaml
rules:
  - match:
      pattern: "password"
      sources: ["command", "content"]
    send: "Avoid hardcoding secrets"
```

**Pre-event sources**: `command`, `description`, `file_path`, `content`
**Post-event sources**: `tool_output`, `error`, `exit_code`

### Special Sources

**`#intent`**: Matches Claude's reasoning from transcript:

```yaml
rules:
  - match:
      pattern: "not sure"
      sources: ["#intent"]
    send: "Check documentation first"
```

**Empty sources**: `sources: []` uses smart defaults per tool, or `sources: ["#all"]` to force all fields

### Default Tool Fields

When no `sources` are specified, Bumpers uses sensible defaults to avoid false positives:

- **Bash**: Only checks `command` field (ignores potentially noisy `description`)
- **Edit**: Checks `file_path` and `new_string` (ignores `old_string` which may be unrelated)
- **Read**: Checks `file_path` only
- **Grep**: Checks `pattern` and `path` fields
- **Unknown tools**: Check all fields (backward compatibility)

## Tool-Specific Configuration

```yaml
rules:
  - match: "go test"
    tool: "^Bash$"  # Only Bash commands
    send: "Use 'just test'"
    
  - match: "secret"
    tool: "^(Write|Edit)$"  # File operations
    send: "Use environment variables"
```

## Debugging

- **Logs**: Hook events logged for analysis
- **Validation**: `bumpers validate` checks configuration
- **Rule skipping**: Invalid rules are skipped with warnings