# Configuration Examples

This guide provides practical examples of Bumpers configurations for common use cases, based on real-world patterns and the actual implementation.

## Basic Rule Patterns

### Command Redirection
Block commands and suggest better alternatives:

```yaml
rules:
  # Redirect test commands to project standard
  - match: "go test"
    send: |
      Use "just test" instead for TDD integration:
      • just test                    # All test categories
      • just test-unit              # Unit tests only  
      • just test-integration       # Integration tests only
      • just test ./internal/pkg    # Specific package
    
  # Enforce linting workflow
  - match: "^(gci|go vet|goimports|gofumpt|go.*fmt|golangci-lint)"
    send: 'Use "just lint fix" to resolve all formatting and lint issues'
    
  # Package manager consistency
  - match: "npm install"
    send: |
      {{if testPath "yarn.lock"}}Use "yarn install" (project uses Yarn)
      {{else if testPath "pnpm-lock.yaml"}}Use "pnpm install" (project uses pnpm)
      {{else}}Use "npm ci" for reproducible installs{{end}}
```

### Safety Rules
Prevent dangerous operations:

```yaml
rules:
  # Block dangerous deletions
  - match: " /tmp"
    send: 'Use project "tmp" directory instead of system /tmp'
    
  # Prevent bypassing safety measures
  - match: "git commit --no-verify|LEFTHOOK=0"
    send: "Pre-commit hooks must not be skipped"
    
  # Dangerous file operations
  - match: 'find\s+.*-exec\s+(rm(\s+-\w+)*|sed\s+-i|>\s*\S+)\s+'
    send: "Don't use dangerous exec in find commands - use specific tools"
    
  # Tool redirection for file creation
  - match: "cat.*EOF.*>\\s*[^\\s]"
    send: "Use Write tool instead of cat heredoc redirection"
    tool: "^Bash$"
```

## Tool-Specific Rules

### File Operations
Target specific Claude Code tools:

```yaml
rules:
  # Prevent secrets in files
  - match: "password|secret|key|token"
    tool: "^(Write|Edit|MultiEdit)$"
    send: |
      Avoid hardcoding secrets in files:
      • Use environment variables
      • Use project config with placeholders  
      • Store in separate .env files (gitignored)
    
  # Configuration file safety  
  - match: "^bumpers\\.yml"
    tool: "^(Read|Edit|Grep)$"
    send: "Bumpers configuration should not be directly accessed"
    
  # Enforce structured queries
  - match: "SELECT.*password|DROP TABLE"
    tool: "^(Write|Edit|MultiEdit|Task)$" 
    send: "Use parameterized queries and avoid sensitive operations"
```

### Multi-Tool Patterns
Rules that apply across different tools:

```yaml
rules:
  # Temporary file consistency
  - match: "/tmp|temp"
    tool: "^(Bash|Write|Edit|Task)$"
    send: "Use project tmp/ directory for temporary files"
    
  # Debug prevention in production code
  - match: "console\\.log|print\\(|fmt\\.Print"
    tool: "^(Write|Edit|MultiEdit)$"
    send: "Use proper logging instead of debug prints"
    
  # Sensitive path protection  
  - match: "/etc|/var|/usr"
    tool: "^(Bash|Read|Write|Glob|Grep)$"
    send: "Avoid system directory access - stay within project"
```

## Advanced Match Patterns

### Post-Tool-Use Analysis
Analyze results after tool execution:

```yaml
rules:
  # Error detection and guidance
  - match:
      pattern: "permission denied|access denied|not found"
      event: "post"  
      sources: ["tool_output"]
    send: |
      {{if testPath ".env"}}Check environment configuration
      {{else}}Verify file permissions and paths{{end}}
    generate: "session"
    
  # Test failure analysis
  - match:
      pattern: "FAIL|failed|error"
      event: "post"
      sources: ["tool_output"]  
    send: "Test failures detected - check logs and dependencies"
    generate:
      mode: "session"
      prompt: "Analyze test failure and suggest specific debugging steps"
      
  # Build error detection
  - match:
      pattern: "build failed|compilation error"
      event: "post"
    send: |
      Build failed - common solutions:
      • Run "just lint fix" for syntax issues
      • Check "go mod tidy" for dependencies  
      • Verify import paths
```

### Intent-Based Matching
Match against Claude's reasoning:

```yaml
rules:
  # Detect uncertainty
  - match:
      pattern: "(not sure|uncertain|don't know|confused)"
      sources: ["#intent"]
    send: |
      When unsure, try these approaches:
      • Check project documentation: {{if testPath "README.md"}}README.md{{end}}
      • Review recent logs: $logs
      • Ask for clarification
    generate: "session"
    
  # Database operation guidance  
  - match:
      pattern: "database|sql|query"
      sources: ["#intent"]
    send: |
      Database operations checklist:
      {{if testPath "migrations/"}}• Run migrations first
      {{end}}• Use connection pooling
      • Validate input parameters
      • Handle connection errors
      
  # Performance concerns
  - match:
      pattern: "(slow|performance|optimize|bottleneck)"
      sources: ["#intent"]  
    send: "Consider profiling before optimization - measure first, optimize second"
    generate: "once"
```

## Command Examples

### Basic Commands
Simple project shortcuts:

```yaml
commands:
  # Quick test command
  - name: "test"
    send: 'Run "just test" to execute all test suites and fix any failures'
    
  # Linting shortcut
  - name: "lint"
    send: 'Run "just lint fix" and address ALL linting issues'
    
  # Combined check
  - name: "check"
    send: 'Run "just lint fix" and "just test" - fix ALL issues before continuing'
    
  # Log access
  - name: "logs" 
    send: "Read Bumpers log: ~/.local/share/bumpers/bumpers.log"
```

### Commands with Arguments
Dynamic commands that process arguments:

```yaml
commands:
  # Search with optional directory
  - name: "search"
    send: |
      {{if eq (argc) 0}}Usage: $search "term" [directory]
      {{else}}Search for "{{argv 1}}"{{if gt (argc) 1}} in {{argv 2}}{{else}} in codebase{{end}}:
      grep -r "{{argv 1}}" {{if gt (argc) 1}}{{argv 2}}{{else}}.{{end}}{{end}}
      
  # Service management
  - name: "run"
    send: |
      {{if eq (argc) 0}}Usage: $run <service> [args]
      Available: {{readFile "docker-compose.yml" | grep "service:" | head 5}}
      {{else}}Start {{argv 1}}{{if gt (argc) 1}} with: {{range slice .Argv 2}}{{.}} {{end}}{{end}}
      docker-compose up {{argv 1}}{{end}}
      
  # Test with pattern
  - name: "testonly"
    send: |
      {{if eq (argc) 0}}Usage: $testonly <pattern>
      {{else}}Run tests matching "{{argv 1}}":
      just test -run "{{argv 1}}"{{end}}
```

### AI-Enhanced Commands
Commands with dynamic AI responses:

```yaml
commands:
  # Context-aware help
  - name: "help"
    send: "Project-specific help and current status"
    generate:
      mode: "session"
      prompt: |
        Provide helpful guidance for this development session.
        Check for common project files and suggest relevant commands.
        Be concise but cover the most important workflows.
        
  # Smart deployment
  - name: "deploy"  
    send: |
      {{if gt (argc) 0}}Deploying to {{argv 1}} environment{{else}}Deployment guidance{{end}}
    generate:
      mode: "once"
      prompt: |
        Provide deployment checklist and environment-specific guidance.
        Consider testing requirements, backup procedures, and rollback plans.
```

## Session Context Examples

### Basic Session Injection
Add context at session start:

```yaml
session:
  # Date context
  - add: "Today's date: {{.Today}}"
  
  # Project status
  - add: |
      Project: {{if testPath "package.json"}}{{readFile "package.json"}}{{end}}
      {{if testPath "go.mod"}}{{readFile "go.mod" | head 1}}{{end}}
      
  # Build status reminder
  - add: |
      {{if testPath "Makefile"}}Build with: make
      {{else if testPath "justfile"}}Build with: just
      {{else if testPath "package.json"}}Build with: npm run build{{end}}
```

### Dynamic Session Context
AI-generated session context:

```yaml
session:
  # Daily development context
  - add: "Current development context and reminders"
    generate:
      mode: "session"
      prompt: |
        Provide a brief development context for this session:
        - Check for uncommitted changes
        - Note any failing tests or build issues
        - Remind about important project conventions
        - Mention any pending tasks or TODOs
        
  # Project-specific guidance
  - add: "Project-specific development guidance"  
    generate:
      mode: "once"
      prompt: |
        Analyze the project structure and provide key development guidelines:
        - Testing approach and commands
        - Code style and linting setup  
        - Build and deployment process
        - Important project conventions
```

## Real-World Configuration

### TDD-Focused Project
Complete configuration for Test-Driven Development:

```yaml
session:
  - add: "TDD workflow active - write tests first, then implementation"

commands:
  - name: "red"
    send: "Write failing test: just test-unit ./pkg/component"
  - name: "green" 
    send: "Make test pass with minimal code"
  - name: "refactor"
    send: "Refactor while keeping tests green: just test-unit"

rules:
  # Enforce TDD workflow
  - match: "go test"
    send: |
      Use TDD commands:
      • just test-unit     # Fast unit tests
      • just test          # Full test suite
      • $red / $green / $refactor for TDD cycle
      
  # Implementation-before-test detection
  - match:
      pattern: "implement.*before.*test"  
      sources: ["#intent"]
    send: "TDD: Write the test first, then make it pass"
    
  # Test quality enforcement
  - match: "Skip|TODO.*test"
    tool: "^(Write|Edit)$"
    send: "Avoid skipped tests - write proper test cases or remove"
```

### Security-Focused Configuration
Configuration emphasizing security practices:

```yaml
rules:
  # Secret detection
  - match: "password|secret|key|token|api_key"
    tool: "^(Write|Edit|MultiEdit)$"
    send: |
      Security violation: Use environment variables for secrets
      • Add to .env (gitignored)
      • Use config management
      • Never commit credentials
    generate: "once"
    
  # Dangerous operations  
  - match: "eval|exec|system|shell_exec"
    send: "Avoid dynamic code execution - use specific, validated functions"
    
  # File permission checks
  - match:
      pattern: "permission denied"
      event: "post"
    send: |
      Security check: Verify you should have access to this resource
      • Check file ownership
      • Validate required permissions
      • Consider principle of least privilege
      
  # Network security
  - match: "http://|ftp://|telnet"
    send: "Use secure protocols (HTTPS, SFTP, SSH) instead of unencrypted connections"
```

This comprehensive set of examples demonstrates the flexibility and power of Bumpers configuration while maintaining the practical, code-first approach that reflects the actual implementation.