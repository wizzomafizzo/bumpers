# Configuration Reference

Bumpers uses a YAML configuration file (`bumpers.yml`) to define rules, commands, and session behavior. This reference documents all configuration options based on the actual implementation.

## Configuration Structure

```yaml
# Optional: Context to inject at the start of every Claude session
session:
  - add: "Today's date is: {{.Today}}"

# Optional: Custom commands triggered by $command syntax
commands:
  - name: "test"
    send: 'Run "just test" to run all tests'
    generate: "session"  # Optional AI generation

# Optional: Rules to match and block tool usage
rules:
  - match: "go test"
    send: "Use 'just test' instead"
    tool: "^Bash$"      # Optional tool filter
    generate: "once"    # Optional AI generation
```

## Rules Configuration

Rules define patterns to match against Claude's tool usage and provide helpful alternatives when matched.

### Match Field

The `match` field supports both simple string and advanced struct forms:

#### Simple Form (String)
```yaml
rules:
  - match: "rm -rf"
    send: "Consider using safer alternatives"
```
- **Default event**: `pre` (matches before tool execution)
- **Default sources**: `[]` (matches all tool input fields)

#### Advanced Form (Struct)
```yaml
rules:
  - match:
      pattern: "rm -rf"
      event: "pre"
      sources: ["command"]
    send: "Consider using safer alternatives"
```

**Fields:**
- **`pattern`** (required): Regex pattern to match against tool input/output
- **`event`** (optional): `"pre"` (default) or `"post"` hook event type
- **`sources`** (optional): Array of field names to match against, empty array matches all fields

### Tool Filter

```yaml
rules:
  - match: "password"
    tool: "^(Write|Edit|MultiEdit)$"  # Only match file editing tools
    send: "Avoid hardcoding secrets in files"
```

- **`tool`** (optional): Regex pattern to match tool names (case-insensitive)
- **Default**: `"^Bash$"` when not specified
- **Common tools**: `Bash`, `Write`, `Edit`, `MultiEdit`, `Read`, `Task`, `Glob`, `Grep`, `WebFetch`, `WebSearch`

### Response Configuration

```yaml
rules:
  - match: "dangerous_command"
    send: "Template message with {{.Command}}"  # Static message
    generate: "session"                         # AI enhancement
```

- **`send`** (required): Template message to display when rule matches
- **`generate`** (optional): AI generation mode - see [AI Generation](#ai-generation)

## Commands Configuration

Commands provide custom responses to `$command` syntax in Claude prompts.

```yaml
commands:
  - name: "search"
    send: |
      Search for "{{argv 1}}" in {{if gt (argc) 1}}{{argv 2}}{{else}}codebase{{end}}:
      {{if eq (argc) 0}}Usage: $search "term" [directory]
      {{else}}grep -r "{{argv 1}}" {{if gt (argc) 1}}{{argv 2}}{{else}}.{{end}}{{end}}
    generate: "off"
```

**Fields:**
- **`name`** (required): Command name (without `$` prefix)
- **`send`** (required): Template message with argument support
- **`generate`** (optional): AI generation mode

### Argument Parsing

Commands support rich argument parsing:
- `$command arg1 "arg with spaces"` → `argv[0]="command"`, `argv[1]="arg1"`, `argv[2]="arg with spaces"`
- Template functions: `{{argc}}` (argument count), `{{argv N}}` (Nth argument)

## Session Configuration

Session entries inject context at the start of every Claude session.

```yaml
session:
  - add: "Project uses TDD with just commands: {{readFile 'justfile'}}"
    generate: "always"
```

**Fields:**
- **`add`** (required): Template content to inject
- **`generate`** (optional): AI generation mode

## AI Generation

All configuration sections support AI-powered message enhancement:

```yaml
generate: "session"           # Simple form: mode only
generate:                     # Advanced form
  mode: "session"
  prompt: "Be encouraging and specific about alternatives"
```

**Modes:**
- **`off`** (default): No AI generation
- **`once`**: Generate once, cache permanently
- **`session`**: Generate once per Claude session
- **`always`**: Generate every time (no caching)

**Custom Prompts:**
- **`prompt`** (optional): Additional context for AI generation
- Combined with built-in prompts for better responses

## Template System

All `send` and `add` fields support Go template syntax with custom functions:

### Variables
- **`{{.Command}}`**: Matched command text (rules only)
- **`{{.Name}}`**: Command name (commands only)
- **`{{.Args}}`**: Raw arguments string (commands only)
- **`{{.Argv}}`**: Parsed arguments array (commands only)
- **`{{.Today}}`**: Current date in YYYY-MM-DD format

### Functions
- **`{{argc}}`**: Number of command arguments (commands only)
- **`{{argv N}}`**: Nth command argument, 0-indexed (commands only)
- **`{{readFile "path"}}`**: Read file content (project root relative, secure)
- **`{{testPath "path"}}`**: Check if file/directory exists
- **`{{if condition}}...{{end}}`**: Conditional logic
- **`{{range}}...{{end}}`**: Loop over collections

### Security
- File operations restricted to project root
- Path traversal protection
- Binary files returned as base64 data URIs

## Event Types and Sources

### Pre-Tool-Use Events (default)
Intercept tool usage before execution:
```yaml
rules:
  - match:
      pattern: "dangerous"
      event: "pre"
      sources: ["command", "description"]  # Match against tool input fields
```

### Post-Tool-Use Events
Analyze results after tool execution:
```yaml
rules:
  - match:
      pattern: "error"
      event: "post" 
      sources: ["tool_output", "#intent"]  # Match against tool outputs and Claude's reasoning
```

### Special Sources
- **`#intent`**: Claude's reasoning extracted from transcript (thinking + explanations)
- **Empty sources array**: Matches all available fields in the event

### Common Tool Fields
**Input fields** (pre-event): `command`, `description`, `file_path`, `content`, `pattern`, `url`, `prompt`
**Output fields** (post-event): `tool_output`, `error`, `result`, `status`

## Validation

Configuration is validated on load:
- **Required fields**: `match.pattern` for rules, `name`/`send` for commands, `add` for session
- **Regex validation**: `match.pattern` and `tool` must be valid regex
- **Generate modes**: Must be `off`, `once`, `session`, or `always`
- **Event values**: Must be `pre` or `post`
- **Response requirement**: Rules must have either `send` message or non-`off` generate mode

Invalid rules are logged as warnings and skipped, allowing partial configuration loading.