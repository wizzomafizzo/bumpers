# Template System

Bumpers uses Go's template engine with custom functions to create dynamic, context-aware messages. All `send` (rules/commands) and `add` (session) fields support template syntax.

## Template Variables

### Rule Context
Available in rule `send` messages:

```yaml
rules:
  - match: "go test"
    send: "Blocked command: '{{.Command}}' - use 'just test' instead"
```

- **`{{.Command}}`**: The matched command text

### Command Context  
Available in command `send` messages:

```yaml
commands:
  - name: "search"
    send: |
      Command: {{.Name}}
      Raw args: {{.Args}}
      Parsed: {{range .Argv}}[{{.}}] {{end}}
```

- **`{{.Name}}`**: Command name (without `$` prefix)
- **`{{.Args}}`**: Raw arguments string as typed
- **`{{.Argv}}`**: Parsed arguments array (includes command name at index 0)

### Shared Context
Available in all templates:

```yaml
session:
  - add: "Today's date: {{.Today}}"
```

- **`{{.Today}}`**: Current date in `YYYY-MM-DD` format

## Template Functions

### Argument Functions
Only available in command templates:

```yaml
commands:
  - name: "test"
    send: |
      {{if eq (argc) 0}}Usage: $test [pattern]
      {{else}}Running tests{{if gt (argc) 1}} for {{argv 1}}{{end}}
      {{end}}
```

- **`{{argc}}`**: Number of arguments (excluding command name)
- **`{{argv N}}`**: Nth argument (0-indexed, 0 = command name)

**Examples:**
- `$test` → `argc=0`, no additional args
- `$test unit` → `argc=1`, `argv 1="unit"`  
- `$test "integration tests" ./pkg` → `argc=2`, `argv 1="integration tests"`, `argv 2="./pkg"`

### File System Functions
Secure file operations within project boundaries:

```yaml
rules:
  - match: "unknown command"
    send: |
      Available commands:
      {{readFile "justfile"}}
      
      {{if testPath "Makefile"}}Also see Makefile{{end}}
```

- **`{{readFile "path"}}`**: Read file content (relative to project root)
- **`{{testPath "path"}}`**: Check if file/directory exists (returns boolean)

**Security features:**
- **Project root restriction**: Can only access files within detected project root
- **Path traversal protection**: `../` and absolute paths outside project are blocked
- **Binary file handling**: Non-UTF-8 files returned as base64 data URIs
- **Error safety**: Returns empty string on errors (missing files, permissions, etc.)

## Control Structures

### Conditionals
```yaml
commands:
  - name: "deploy"
    send: |
      {{if testPath ".env.prod"}}
      Deploying with production config
      {{else}}
      ⚠️  No .env.prod found - using defaults
      {{end}}
```

**Comparison operators:**
- `eq` (equal): `{{if eq (argc) 0}}`
- `gt` (greater): `{{if gt (argc) 1}}`  
- `lt` (less): `{{if lt (argc) 3}}`
- `ge` (greater/equal): `{{if ge (argc) 2}}`
- `le` (less/equal): `{{if le (argc) 5}}`

### Loops
```yaml
commands:
  - name: "help"
    send: |
      Available args: {{range .Argv}}
      - {{.}}{{end}}
```

- **`{{range .Argv}}...{{end}}`**: Iterate over argument array
- **`{{.}}`**: Current item in range context

## Template Examples

### Dynamic Command Help
```yaml
commands:
  - name: "run"
    send: |
      {{if eq (argc) 0}}Usage: $run <service> [args...]
      Available services: {{readFile "docker-compose.yml"}}
      {{else}}Starting service: {{argv 1}}{{if gt (argc) 1}} with args: {{range slice .Argv 2}}{{.}} {{end}}{{end}}
      {{end}}
```

### Context-Aware Rules
```yaml
rules:
  - match: "npm install"
    send: |
      {{if testPath "yarn.lock"}}Use 'yarn install' (project uses Yarn)
      {{else if testPath "pnpm-lock.yaml"}}Use 'pnpm install' (project uses pnpm)  
      {{else}}Use 'npm ci' for faster, reproducible installs{{end}}
```

### File Content Integration
```yaml
session:
  - add: |
      Project commands available:
      {{readFile "package.json" | fromJson | .scripts | keys | join ", "}}
      
      Recent changes: {{readFile "CHANGELOG.md" | truncate 200}}
```

### Multi-Line Templates
```yaml
rules:
  - match: "git push.*force"
    send: |
      🚨 Force push detected!
      
      Safer alternatives:
      • git push --force-with-lease
      • git push --force-if-includes
      
      Current branch: {{if testPath ".git/HEAD"}}{{readFile ".git/HEAD"}}{{end}}
```

## Template Security

### File Access Controls
- **Sandboxed**: Only project root and subdirectories accessible
- **Path validation**: Absolute paths and `../` traversal blocked
- **Symlink protection**: Resolved paths must stay within project  
- **Error handling**: Invalid paths return empty strings (no exceptions)

### Content Safety
- **UTF-8 detection**: Binary files encoded as base64 data URIs
- **Size limits**: Large files may be truncated in template output
- **Permission respect**: Unreadable files return empty strings

### Example Security Behavior
```yaml
# ✅ Allowed - project relative
{{readFile "config/app.yml"}}

# ❌ Blocked - absolute path  
{{readFile "/etc/passwd"}}

# ❌ Blocked - traversal attempt
{{readFile "../../../etc/passwd"}}

# ✅ Safe fallback - returns empty string
{{readFile "nonexistent.txt"}}
```

## Advanced Template Patterns

### Conditional Argument Processing
```yaml
commands:
  - name: "deploy"
    send: |
      {{$env := "dev"}}
      {{if gt (argc) 1}}{{$env = argv 1}}{{end}}
      
      Deploying to {{$env}} environment
      {{if eq $env "prod"}}⚠️  Production deployment!{{end}}
```

### File-Based Configuration
```yaml
rules:
  - match: "test.*integration"  
    send: |
      {{if testPath "docker-compose.test.yml"}}
      Start test environment: docker-compose -f docker-compose.test.yml up
      {{else}}
      Configure integration tests first: cp docker-compose.test.example.yml docker-compose.test.yml
      {{end}}
```

### Dynamic Help Generation
```yaml
commands:
  - name: "help"
    send: |
      Project: {{if testPath "package.json"}}{{readFile "package.json" | fromJson | .name}}{{else}}{{readFile "go.mod" | head 1}}{{end}}
      
      {{if testPath "justfile"}}Build commands:
      {{readFile "justfile" | grep "^[a-z]" | head 10}}{{end}}
```

## Template Debugging

### Testing Templates
Use the `bumpers validate` command to check template syntax:
```bash
bumpers validate  # Validates all templates in bumpers.yml
```

### Common Template Errors
- **Undefined variables**: Use `{{if .Variable}}` to check existence
- **Invalid functions**: Check function name spelling and availability
- **Syntax errors**: Ensure proper `{{` and `}}` pairing
- **File access**: Remember project root restrictions

### Template Output in Logs
Template execution errors are logged to `~/.local/share/bumpers/bumpers.log`:
```
ERROR template execution failed template="{{.InvalidVar}}" error="undefined variable"
WARN file access blocked path="/etc/passwd" reason="outside project root"
```