# Bumpers: Simple Claude Code Hook Guard

## Overview

A focused CLI utility that intercepts Claude Code hook events and provides intelligent command guarding with positive guidance. Instead of just blocking commands, it suggests better alternatives using encouraging, educational messaging.

## Core Goals

- **Positive Guidance**: Suggest better commands rather than just saying "no"
- **Educational**: Explain why alternatives are better
- **Simple**: Easy configuration via YAML
- **Extensible**: Support multiple rules and command patterns

## Current State ✅ **COMPLETED**

~~We have a shell script (`.claude/hooks/check-go-test.sh`) that blocks `go test` and redirects to `make` targets. It works well but is:~~
~~- Hard-coded for one rule~~
~~- Shell script limitations~~  
~~- No easy configuration~~

**✅ MVP Implementation Complete** - Bumpers is now fully functional with:
- Complete TDD-driven implementation with excellent test coverage
- YAML-based configuration system with regex pattern matching
- Clean Go architecture following best practices
- Minimal CLI with proper Cobra integration
- All core functionality working and tested

## What We're Building

### MVP Features ✅ **IMPLEMENTED**

1. **Hook Integration** ✅
   - ✅ Parse JSON from stdin (Claude hook input) - `internal/hooks`
   - ⏸️ Auto-manage `.claude/settings.local.json` registration (future)
   - ⏸️ Simple backup system (`.backup` files) (future)

2. **Rule System** ✅  
   - ✅ YAML config file for defining rules - `internal/config`
   - ✅ Regex pattern matching for commands - `internal/matcher`
   - ✅ Allow/deny actions with alternatives - Full implementation

3. **Response Generation** ✅
   - ✅ Template-based responses for common cases - `internal/response`
   - ⏸️ Optional Claude CLI integration for dynamic responses (future)
   - ✅ Consistent positive, encouraging tone

4. **Basic Tooling** ✅
   - ✅ CLI for testing rules - `bumpers test`
   - ✅ Configuration management - YAML + validation
   - ✅ Simple debug logging - Built into CLI app

### Configuration Format

```yaml
# bumpers.yaml
rules:
  - name: "block-go-test"
    pattern: "go test.*"
    action: "deny"
    message: "Use make test instead for better TDD integration"
    alternatives:
      - "make test          # Run all tests"
      - "make test-unit     # Run unit tests only"
    use_claude: false

  - name: "dangerous-rm"
    pattern: "rm -rf /"
    action: "deny"
    use_claude: true
    claude_prompt: "Explain why this rm command is dangerous"
```

### CLI Commands

```bash
# Test a command against current rules ✅ IMPLEMENTED
bumpers test "go test ./..."

# Check hook status ✅ IMPLEMENTED  
bumpers status

# Process hook from stdin (main usage) ✅ IMPLEMENTED
echo '{"command": "go test", "args": ["go", "test"]}' | bumpers

# Build and install ✅ IMPLEMENTED
make build    # Creates bin/bumpers
make install  # Installs to $GOPATH/bin

# Future commands:
# bumpers install  # Install hooks into current project
# bumpers config   # Show current config
```

## Technical Stack ✅ **IMPLEMENTED**

- ✅ **Framework**: Cobra for CLI - Clean command structure
- ✅ **Config**: Standard Go + YAML (gopkg.in/yaml.v3) - No Viper needed, simpler
- ✅ **Matching**: Standard Go regexp - Fast pattern matching
- ⏸️ **External**: Local `claude` CLI calls (future enhancement)

## Project Structure ✅ **IMPLEMENTED**

```
bumpers/
├── cmd/bumpers/          # ✅ Main CLI app (minimal, testable)
├── internal/
│   ├── cli/             # ✅ Application orchestration (TDD-driven)
│   ├── config/          # ✅ YAML config handling (LoadFromFile, LoadFromYAML)
│   ├── hooks/           # ✅ Hook event processing (ParseHookInput)
│   ├── matcher/         # ✅ Pattern matching (NewRuleMatcher, regex)
│   └── response/        # ✅ Response generation (FormatResponse)
├── configs/
│   └── bumpers.yaml     # ✅ Default config with sample rules
├── docs/
│   └── project-plan.md  # ✅ This file (updated status)
├── Makefile             # ✅ Application-focused build system
├── go.mod               # ✅ Latest dependencies (Cobra v1.9.1, etc)
└── bin/                 # ✅ Built binaries (make build)
```

**Test Coverage**: 77.8% - 100% across packages with comprehensive TDD approach

## Implementation Steps ✅ **COMPLETED**

1. ✅ **Setup**: Go project with Cobra + YAML (simpler than Viper)
2. ✅ **Hook Handler**: Parse stdin JSON, extract commands (`internal/hooks`)
3. ✅ **Rule Matcher**: Regex from YAML config (`internal/matcher`)  
4. ✅ **Response System**: Template-based responses (`internal/response`)
5. ⏸️ **Settings Manager**: Auto-configure Claude settings (future)

## Implementation Notes & Lessons Learned

**🎯 TDD Success**: Strict test-driven development resulted in:
- High-quality, testable architecture
- Minimal cmd layer (0% coverage needed - just wiring)
- Excellent separation of concerns
- Comprehensive test suite with strong coverage

**🏗️ Architecture Decisions**:
- Chose standard Go YAML over Viper (simpler, less dependencies)
- Built minimal CLI orchestrator in `internal/cli` package
- Used value types instead of pointers for better performance
- Followed Go project layout conventions exactly

**🚧 Future Enhancements**:
- Claude CLI integration for dynamic responses
- Hook installation/management commands  
- Configuration validation and better error messages
- Settings backup system

## What We're NOT Building

- No complex analytics or dashboards
- No plugins or extension system
- No web UI
- No multi-project management (yet)
- No semantic/AI matching (just regex)

This keeps it simple and focused on the core value: intelligent command guarding that guides rather than blocks.