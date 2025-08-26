# Hook Events Guide

Bumpers integrates with Claude Code through hook events that intercept different stages of tool usage. This guide explains how each hook type works and how to configure rules for them.

## Hook Types

### PreToolUse Hook (Default)
Intercepts tool usage **before** execution, allowing you to block or modify commands.

```yaml
rules:
  - match: "rm -rf"
    send: "Use safer alternatives like specific file deletion"
    # event: "pre" is the default
```

**When triggered**: Before Claude executes any tool
**Use cases**: Block dangerous commands, enforce coding standards, redirect to better tools
**Exit codes**: 0 (allow), 1 (block with message)

### PostToolUse Hook
Analyzes results **after** tool execution to provide guidance based on outcomes.

```yaml
rules:
  - match:
      pattern: "permission denied|access denied"
      event: "post"
      sources: ["tool_output"]
    send: "Try running with appropriate permissions or check file ownership"
    generate: "session"
```

**When triggered**: After tool execution completes
**Use cases**: Error analysis, result interpretation, follow-up suggestions
**Behavior**: Always informational (cannot block completed actions)

### UserPromptSubmit Hook
Handles custom commands using `$command` syntax in Claude prompts.

```yaml
commands:
  - name: "test"
    send: 'Run "just test" to execute all test suites'
```

**When triggered**: When user types `$test` in Claude
**Use cases**: Project-specific shortcuts, context injection, standardized workflows

### SessionStart Hook
Injects context at the beginning of every Claude session.

```yaml
session:
  - add: "Today's date is: {{.Today}}"
  - add: "Project uses TDD workflow with these commands: {{readFile 'justfile'}}"
```

**When triggered**: At the start of each Claude session
**Use cases**: Project context, current status, important reminders

## Event Configuration

### Match Sources

Rules can target specific fields in hook events using the `sources` array:

#### Pre-Event Sources (Tool Inputs)
```yaml
rules:
  - match:
      pattern: "password|secret|key"
      sources: ["command", "description", "file_path"]
    send: "Avoid hardcoding secrets"
```

**Common pre-event sources:**
- **`command`**: Shell command text (Bash tool)
- **`description`**: Human-readable descriptions (most tools)  
- **`file_path`**: Target file paths (Write, Edit, Read tools)
- **`content`**: File content being written (Write tool)
- **`pattern`**: Search patterns (Grep, Glob tools)
- **`url`**: Web URLs (WebFetch tool)
- **`prompt`**: AI prompts (Task tool)

#### Post-Event Sources (Tool Outputs)
```yaml
rules:
  - match:
      pattern: "error|failed|exception"
      event: "post"
      sources: ["tool_output"]
    send: "Debug the error and try again"
    generate: "session"
```

**Common post-event sources:**
- **`tool_output`**: Primary tool result/output
- **`error`**: Error messages from failed operations
- **`exit_code`**: Command exit codes (Bash tool)
- **`status`**: Operation status indicators

### Special Sources

#### Intent Matching
The special `#intent` source matches against Claude's reasoning extracted from the conversation transcript:

```yaml
rules:
  - match:
      pattern: "I need to.*database"
      sources: ["#intent"]
    send: "Remember to check database connection settings first"
    generate: "once"
```

**What `#intent` captures:**
- Claude's thinking process
- Explanations of planned actions
- Reasoning behind tool choices
- Context from conversation history

**Use cases:**
- Detect when Claude is confused or uncertain
- Provide guidance based on intent rather than specific commands
- Catch potential issues before they happen

### Source Matching Behavior

#### Empty Sources (Match All)
```yaml
rules:
  - match:
      pattern: "dangerous_pattern"
      sources: []  # Empty array = match all available fields
```

#### Field Name Flexibility
- **No validation**: Any field name can be specified in sources
- **Case sensitive**: Field names must match exactly as provided by Claude Code
- **Tool-specific**: Different tools provide different field names
- **Future-proof**: New tools and fields automatically supported

## Hook Event Processing Flow

### Pre-Tool-Use Flow
1. **Event Detection**: Claude Code sends tool input before execution
2. **Rule Matching**: Bumpers checks patterns against specified sources
3. **Response Generation**: If matched, generates response (with optional AI)
4. **Decision**: Returns exit code 0 (allow) or 1 (block with message)

### Post-Tool-Use Flow  
1. **Event Detection**: Claude Code sends tool output after execution
2. **Transcript Analysis**: Extract Claude's intent from conversation
3. **Rule Matching**: Check patterns against outputs and intent
4. **Guidance Generation**: Provide contextual advice (always informational)

### Command Processing Flow
1. **Syntax Detection**: User types `$command args` in Claude
2. **Argument Parsing**: Extract command name and arguments
3. **Template Processing**: Apply argument substitution and templates
4. **AI Enhancement**: Optional AI generation for dynamic responses
5. **Context Injection**: Add response to Claude's context

## Advanced Hook Patterns

### Multi-Source Rules
```yaml
rules:
  - match:
      pattern: "sudo|admin|root"
      sources: ["command", "description", "file_path"]
    send: "Avoid privileged operations - use project-specific tools"
```

### Event-Specific Rules  
```yaml
rules:
  # Block before execution
  - match:
      pattern: "rm.*important"
      event: "pre"
    send: "This could delete important files"
    
  # Analyze after execution
  - match:
      pattern: "rm.*important" 
      event: "post"
      sources: ["tool_output"]
    send: "Check if important files were affected"
```

### Intent-Based Guidance
```yaml
rules:
  - match:
      pattern: "(struggling|difficult|not working)"
      sources: ["#intent"]
    send: "Take a step back - check the logs: {{readFile '.claude/bumpers/bumpers.log'}}"
    generate: "session"
```

## Tool-Specific Hook Configuration

### Bash Tool Focus
```yaml
rules:
  - match: "go test"
    tool: "^Bash$"  # Only Bash commands
    send: "Use 'just test' for TDD integration"
```

### File Operation Monitoring
```yaml
rules:
  - match:
      pattern: "secret|password|key"
      sources: ["content", "file_path"]
    tool: "^(Write|Edit|MultiEdit)$"
    send: "Use environment variables for secrets"
```

### Multi-Tool Rules
```yaml
rules:
  - match:
      pattern: "/tmp"
    tool: "^(Bash|Task)$"
    send: "Use project 'tmp' directory instead"
```

## Hook Debugging

### Log Analysis
Bumpers logs all hook events to `~/.local/share/bumpers/bumpers.log`:
```
INFO hook event processed type=PreToolUse tool=Bash matched=true
WARN invalid rule skipped ruleIndex=3 pattern="[invalid"
```

### Rule Validation
Invalid rules are skipped with warnings, allowing partial configuration loading:
- Invalid regex patterns
- Unknown generate modes  
- Missing required fields
- Invalid event types

### Testing Hooks
Use the `bumpers validate` command to check configuration syntax before deploying rules.