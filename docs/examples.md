# Configuration Examples

Practical Bumpers configurations for common use cases.

## Basic Rules

### Command Redirection
```yaml
rules:
  - match: "go test"
    send: 'Use "just test" instead'
    
  - match: "npm install"
    send: |
      {{if testPath "yarn.lock"}}Use "yarn install"
      {{else}}Use "npm ci" for reproducible installs{{end}}
```

### Safety Rules
```yaml
rules:
  - match: "rm -rf"
    send: "Use specific file deletion instead"
    
  - match: "git commit --no-verify"
    send: "Pre-commit hooks should not be skipped"
    
  - match: "password|secret"
    tool: "^(Write|Edit)$"
    send: "Use environment variables for secrets"
```

## Commands

### Basic Commands
```yaml
commands:
  - name: "test"
    send: 'Run "just test" to execute all tests'
    
  - name: "lint"
    send: 'Run "just lint fix" to fix formatting'
```

### Commands with Arguments
```yaml
commands:
  - name: "search"
    send: |
      {{if eq (argc) 0}}Usage: $search "term" [directory]
      {{else}}grep -r "{{argv 1}}" {{if gt (argc) 1}}{{argv 2}}{{else}}.{{end}}{{end}}
      
  - name: "run"
    send: |
      {{if eq (argc) 0}}Usage: $run <service>
      {{else}}docker-compose up {{argv 1}}{{end}}
```

## Session Context

### Basic Session
```yaml
session:
  - add: "Today's date: {{.Today}}"
  - add: "Project commands: {{readFile 'justfile'}}"
```

### AI-Enhanced Session
```yaml
session:
  - add: "Development context for today"
    generate:
      mode: "session"
      prompt: "Provide brief status and reminders for this session"
```

## AI Generation

### Error Analysis
```yaml
rules:
  - match:
      pattern: "error|failed"
      event: "post"
      sources: ["tool_output"]
    send: "Command failed: {{.Command}}"
    generate:
      mode: "session"
      prompt: "Analyze the error and suggest debugging steps"
```

### Context-Aware Commands
```yaml
commands:
  - name: "help"
    send: "Project help"
    generate:
      mode: "once"
      prompt: "List most useful commands for this project"
```

## Project-Specific Examples

### TDD Workflow
```yaml
session:
  - add: "TDD workflow: write tests first"

commands:
  - name: "red"
    send: "Write failing test: just test-unit"
  - name: "green"
    send: "Make test pass with minimal code"

rules:
  - match: "go test"
    send: 'Use TDD commands: just test-unit, $red, $green'
```

### Security-Focused
```yaml
rules:
  - match: "password|secret|key"
    tool: "^(Write|Edit)$"
    send: |
      Security: Use environment variables for secrets
      • Add to .env (gitignored)
      • Never commit credentials
    generate: "once"
    
  - match: "eval|exec|system"
    send: "Avoid dynamic code execution"
    
  - match: "http://"
    send: "Use HTTPS instead of HTTP"
```

## Template Examples

### File-Based Logic
```yaml
rules:
  - match: "unknown command"
    send: |
      {{if testPath "justfile"}}Available commands:
      {{readFile "justfile"}}{{end}}
      {{if testPath "Makefile"}}Build with: make{{end}}
```

### Project Detection
```yaml
session:
  - add: |
      {{if testPath "package.json"}}Node.js project
      {{else if testPath "go.mod"}}Go project
      {{else if testPath "Cargo.toml"}}Rust project{{end}}
```

### Template Patterns

Dynamic pattern matching using project context:

```yaml
rules:
  # Block main config but allow test configs
  - match: "^{{.ProjectRoot}}/bumpers\\.yml$"
    tool: "Read|Edit|Grep"  
    send: "Main configuration should not be accessed directly"

  # Project-specific file patterns
  - match: "{{.ProjectRoot}}/(test|spec).*\\.yml$"
    tool: "Read|Edit"
    send: "Test configurations can be modified freely"
    
  # Mixed template variables and regex quantifiers
  - match: "^{{.ProjectRoot}}/[a-z]{2,4}/config\\.json$"
    tool: "Write"
    send: "Language-specific configs: {{.Command}}"

  # Environment-aware patterns  
  - match: "{{.ProjectRoot}}/prod"
    send: "Production files require extra care ({{.Today}})"
```

**Use Cases:**
- **Project-specific blocking**: Target files by full path
- **Test file exceptions**: Allow test configs while blocking main ones  
- **Environment awareness**: Different rules for prod vs dev files
- **Flexible matching**: Combine templates with regex for powerful patterns

This configuration reference reflects the actual Bumpers implementation and avoids unsupported features.