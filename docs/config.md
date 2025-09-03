# Configuration Reference

Bumpers uses a YAML configuration file (`bumpers.yml`) to define rules, commands, and session behavior.

## Configuration Structure

```yaml
session:
  - add: "Today's date is: {{.Today}}"

commands:
  - name: "test"
    send: 'Run "just test" to run all tests'
    generate: "session"

rules:
  - match: "go test"
    send: "Use 'just test' instead"
    tool: "^Bash$"
    generate: "once"
```

## Rules

Rules match against tool usage and provide guidance.

### Match Field

Simple string form:
```yaml
rules:
  - match: "rm -rf"
    send: "Consider safer alternatives"
```

Structured form:
```yaml
rules:
  - match:
      pattern: "rm -rf"
      event: "pre"
      sources: ["command"]
    send: "Consider safer alternatives"
```

**Fields:**
- `pattern` (required): Regex pattern
- `event` (optional): `pre` (default) or `post`
- `sources` (optional): Field names to match, empty = all fields

### Template Patterns

Patterns support template variables for dynamic matching:

```yaml
rules:
  # Block bumpers.yml access in project root but allow test files
  - match: "^{{.ProjectRoot}}/bumpers\\.yml$"
    tool: "Read|Edit|Grep"
    send: "Bumpers configuration file should not be accessed."
```

**Available Variables:**
- `{{.ProjectRoot}}`: Project root directory path
- `{{.Today}}`: Current date (YYYY-MM-DD format)

**Template/Regex Compatibility:**
- Templates use `{{}}` syntax, regex quantifiers use `{}`
- Fully compatible: `{{.ProjectRoot}}/[a-z]{2,4}/config\\.yml`

### Tool Filter

```yaml
rules:
  - match: "password"
    tool: "^(Write|Edit)$"
    send: "Avoid secrets in files"
```

- `tool` (optional): Regex for tool names, default `^Bash$`

### Response

```yaml
rules:
  - match: "command"
    send: "Message with {{.Command}}"
    generate: "session"
```

- `send` (required): Template message
- `generate` (optional): AI mode - `off`, `once`, `session`, `always`

## Commands

Custom responses to `$command` syntax:

```yaml
commands:
  - name: "search"
    send: 'Search for "{{argv 1}}" in codebase'
    generate: "off"
```

**Fields:**
- `name` (required): Command name
- `send` (required): Template message
- `generate` (optional): AI mode

### Arguments
- `{{argc}}`: Argument count
- `{{argv N}}`: Nth argument (0=command name)

## Session

Context injection at session start:

```yaml
session:
  - add: "Project info: {{readFile 'README.md'}}"
    generate: "once"
```

## AI Generation

```yaml
generate: "session"           # Simple
generate:                     # Advanced
  mode: "session"
  prompt: "Be specific"
```

**Modes:**
- `off`: No AI
- `once`: Cache permanently  
- `session`: Cache per session
- `always`: No caching

## Templates

Available variables:
- `{{.Command}}`: Matched command (rules)
- `{{.Name}}`, `{{.Args}}`, `{{.Argv}}`: Command context
- `{{.Today}}`: Current date

Functions:
- `{{readFile "path"}}`: Read file (secure)
- `{{testPath "path"}}`: Check if exists
- `{{argc}}`, `{{argv N}}`: Command arguments

## Event Types

**Pre-events** (before execution):
```yaml
rules:
  - match:
      pattern: "dangerous"
      sources: ["command"]
```

**Post-events** (after execution):
```yaml
rules:
  - match:
      pattern: "error"
      event: "post"
      sources: ["tool_output"]
```

**Special sources:**
- `#intent`: Claude's reasoning from transcript  
- `#all`: Force check all fields
- Empty array: Use smart defaults per tool

**Default fields per tool:**
- **Bash**: `command` (not `description`)
- **Edit**: `file_path`, `new_string` (not `old_string`)
- **Read**: `file_path`
- **Grep**: `pattern`, `path`
- Unknown tools: all fields

## Validation

- `match.pattern` required for rules
- `name`/`send` required for commands  
- `add` required for session
- Regex patterns must be valid
- Generate modes: `off`, `once`, `session`, `always`
- Events: `pre`, `post`

Invalid rules are skipped with warnings.