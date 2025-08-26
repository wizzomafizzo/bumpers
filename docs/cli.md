# CLI Commands

Bumpers provides several commands for managing configuration, hook integration, and system status. All commands support the `--config` flag to specify a custom configuration file.

## Main Command

### `bumpers`
Main entry point that shows help when run without arguments.

```bash
bumpers --help
```

**Global Options:**
- `--config`, `-c`: Path to configuration file (default: `bumpers.yml`)

## Subcommands

### `bumpers hook`
Process hook input from Claude Code and apply configured rules.

```bash
bumpers hook < hook_input.json
```

**Purpose**: Main hook processor called by Claude Code
**Input**: JSON hook event from stdin
**Output**: Processed response or exit code
**Usage**: Typically configured in Claude Code settings, not called directly

**Exit Codes:**
- `0`: Allow operation (or informational message)
- `2`: Block operation with message

**Example JSON Input:**
```json
{
  "tool_name": "Bash",
  "tool_input": {
    "command": "go test ./...",
    "description": "Run all tests"
  },
  "transcript_path": "/path/to/transcript.jsonl"
}
```

### `bumpers install`
Install bumpers configuration and Claude Code hooks.

```bash
bumpers install [--config bumpers.yml]
```

**What it does:**
1. **Creates template configuration**: Generates `bumpers.yml` if it doesn't exist
2. **Configures Claude hooks**: Updates Claude Code settings to use Bumpers
3. **Sets up directories**: Creates necessary cache and log directories
4. **Validates setup**: Ensures Claude Code can find and execute Bumpers

**Example Output:**
```
✓ Created bumpers.yml configuration template
✓ Updated Claude Code hook settings
✓ Created cache directory: ~/.local/share/bumpers/cache
✓ Installation complete - restart Claude Code if running
```

**Configuration Created:**
- Basic `bumpers.yml` with example rules
- Claude Code hook integration
- Logging and cache directories

### `bumpers status`
Check current hook integration status.

```bash
bumpers status [--config bumpers.yml]
```

**Information Displayed:**
- **Configuration file**: Path and validation status
- **Claude Code integration**: Hook installation status
- **Cache directories**: Location and usage
- **Recent activity**: Log entries and hook calls

**Example Output:**
```
Configuration: ./bumpers.yml ✓
Rules: 5 valid, 0 warnings
Commands: 3 defined
Session: 1 injection configured

Claude Code Integration: ✓ Active
Hook path: /usr/local/bin/bumpers
Last activity: 2024-01-15 14:30:22

Cache: ~/.local/share/bumpers/
Session cache: 0 entries
Permanent cache: 3 entries (15.2 KB)
```

### `bumpers validate`
Validate configuration file syntax and rules.

```bash
bumpers validate [--config bumpers.yml]
```

**Validation Checks:**
- **YAML syntax**: File format and structure
- **Required fields**: Essential configuration elements
- **Regex patterns**: Rule patterns and tool filters
- **Template syntax**: Template variables and functions
- **Generate modes**: AI generation configuration
- **Event types**: Hook event specifications

**Example Output:**
```
✓ Configuration file: ./bumpers.yml
✓ YAML syntax valid
✓ 5 rules validated
  - Rule 1: ✓ Pattern valid, tool filter valid
  - Rule 2: ✓ Pattern valid, using default tool
  - Rule 3: ⚠ Warning: Complex regex pattern may be slow
✓ 3 commands validated
✓ 1 session configuration validated
✓ All template syntax valid

Configuration is valid with 1 warning.
```

**Exit Codes:**
- `0`: Configuration is valid
- `1`: Configuration has errors
- `2`: Configuration has warnings (but is usable)

## Common Usage Patterns

### Initial Setup
```bash
# Install and configure
bumpers install

# Check installation
bumpers status

# Validate configuration
bumpers validate
```

### Development Workflow
```bash
# Edit configuration
vim bumpers.yml

# Validate changes
bumpers validate

# Check status
bumpers status

# Test with sample input (debugging)
echo '{"tool_name":"Bash","tool_input":{"command":"go test"}}' | bumpers hook
```

### Troubleshooting
```bash
# Check configuration status
bumpers status

# Validate configuration
bumpers validate

# Check logs (platform specific)
# Linux/macOS:
tail -f ~/.local/share/bumpers/bumpers.log

# Windows:
tail -f %LOCALAPPDATA%/bumpers/bumpers.log
```

## Configuration File Discovery

When using the default configuration, Bumpers searches for:

1. `bumpers.yml` (preferred)
2. `bumpers.yaml`

**Project Root Detection:**
- Searches up directory tree from current location
- Looks for `.git/`, `go.mod`, `package.json`, etc.
- Falls back to current directory if no project root found

## Environment Variables

### Available Environment Variables
- **`ANTHROPIC_API_KEY`**: Required for AI-powered responses
- **`BUMPERS_SKIP`**: Set to `1` to temporarily disable all hooks

**Example:**
```bash
# Set API key for AI generation
export ANTHROPIC_API_KEY=your_key_here

# Temporarily disable hooks
export BUMPERS_SKIP=1
```

## Integration with Build Systems

### Just Integration
```bash
# justfile
validate:
    bumpers validate
    
status:
    bumpers status
    
install-bumpers:
    bumpers install
```

### Make Integration
```makefile
# Makefile
.PHONY: validate-config
validate-config:
	bumpers validate

.PHONY: setup-hooks  
setup-hooks:
	bumpers install
```

### Package.json Scripts
```json
{
  "scripts": {
    "validate-bumpers": "bumpers validate",
    "setup-bumpers": "bumpers install",
    "check-bumpers": "bumpers status"
  }
}
```

## Debugging Hook Events

### Manual Hook Testing
```bash
# Test PreToolUse event
cat << EOF | bumpers hook
{
  "tool_name": "Bash",
  "tool_input": {
    "command": "go test ./...",
    "description": "Run tests"
  },
  "transcript_path": "/dev/null"
}
EOF

# Test PostToolUse event  
cat << EOF | bumpers hook
{
  "tool_name": "Bash", 
  "tool_input": {
    "command": "go test ./..."
  },
  "tool_output": {
    "exit_code": 1,
    "stdout": "FAIL: TestExample",
    "stderr": "permission denied"
  },
  "transcript_path": "/dev/null"
}
EOF
```

### Log Analysis
```bash
# Follow live logs
tail -f ~/.local/share/bumpers/bumpers.log

# Search for specific events
grep "hook event" ~/.local/share/bumpers/bumpers.log

# Check for errors
grep "ERROR" ~/.local/share/bumpers/bumpers.log
```

This CLI reference is based on the actual command implementations and reflects the current behavior of the Bumpers system.