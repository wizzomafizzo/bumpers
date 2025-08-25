# Command Arguments Implementation Plan

**Feature**: Add argument support to commands defined in config file  
**Issue**: #14  
**Status**: ✅ COMPLETE - All phases implemented and tested  
**Last Updated**: 2025-08-25

## Overview

Enable commands to accept and parse arguments from user prompts, making them available in templates through variables and functions. This extends the existing `$commandName` syntax to support `$commandName arg1 "arg with spaces" arg3`.

## Implementation Checklist

### Phase 1: Core Infrastructure
- [x] **Argument parsing function** (new: `internal/cli/args.go`) ✅ 2025-08-25
  - [x] Parse space-separated arguments respecting quoted strings
  - [x] Handle empty arguments gracefully
  - [x] Support both single and double quotes
  - [x] Test edge cases: empty strings, escaped quotes, mixed quotes
  
- [x] **Extend CommandContext** (`internal/template/context.go`) ✅ 2025-08-25
  - [x] Add `Args` field (string) - raw arguments after command name
  - [x] Add `Argv` field ([]string) - parsed arguments including command at index 0
  - [x] Update `BuildCommandContext` function signature to accept arguments
  - [x] Maintain backward compatibility with existing `Name` field

### Phase 2: Template Functions
- [x] **Template function additions** (`internal/template/functions.go`) ✅ 2025-08-25
  - [x] `argc` function - returns count of arguments (excluding command name)
  - [x] `argv` function - accepts index, returns argument at that position
    - [x] `argv 0` returns command name
    - [x] `argv 1`, `argv 2`, etc. return actual arguments
    - [x] Out-of-bounds access returns empty string (no error)
    - [x] Comprehensive nil/edge case handling
  - [x] Register functions in `createFuncMap` ✅ 2025-08-25

### Phase 3: Command Processing Updates
- [x] **Modify `ProcessUserPrompt`** (`internal/cli/commands.go`) ✅ 2025-08-25
  - [x] Parse command name and arguments from full command string
  - [x] Pass arguments to template execution
  - [x] Update command matching to use only the command name part
  - [x] Ensure logging includes argument information for debugging

- [x] **Update template execution** (`internal/template/template.go`) ✅ 2025-08-25
  - [x] Add `ExecuteCommandTemplateWithArgs` function for arguments
  - [x] Update command processing to use new function

### Phase 4: Testing
- [x] **Unit tests for argument parsing** ✅ 2025-08-25
  - [x] Test basic space-separated arguments  
  - [x] Test quoted arguments with spaces
  - [x] Test mixed quotes and escaping
  - [x] Test empty argument lists
  - [x] Test malformed input handling

- [x] **Unit tests for template functions** ✅ 2025-08-25
  - [x] Test `argc` with various argument counts
  - [x] Test `argv` with valid indices
  - [x] Test `argv` with out-of-bounds indices
  - [x] Test `argv 0` returns command name

- [x] **Integration tests** ✅ 2025-08-25
  - [x] Test full command processing with arguments
  - [x] Test template rendering with argument variables
  - [x] Test backward compatibility with existing commands

- [x] **Update existing tests** ✅ 2025-08-25
  - [x] Added new template execution method (`ExecuteCommandTemplateWithArgs`)
  - [x] Ensured existing command tests still pass

### Phase 5: Documentation & Examples
- [x] **Update configuration examples** ✅ 2025-08-25
  - [x] Add example commands using argument features in `bumpers.yml`
  - [x] Document template syntax with comprehensive examples
  
- [x] **Update project documentation** ✅ 2025-08-25
  - [x] Add argument support documentation to CLAUDE.md
  - [x] Include usage examples and template function reference

## Technical Specifications

### Argument Parsing Behavior
- **Input**: `$test foo "bar baz" qux`
- **Command Name**: `test`
- **Raw Args String**: `foo "bar baz" qux`
- **Parsed Args Array**: `["test", "foo", "bar baz", "qux"]`

### Template API
```yaml
commands:
  - name: test
    send: |
      Command: {{.Name}}          # "test"
      Raw args: {{.Args}}         # foo "bar baz" qux
      Arg count: {{argc}}         # 3
      First arg: {{argv 1}}       # foo
      Second arg: {{argv 2}}      # bar baz
      All args loop:
      {{range $i := seq 1 (add argc 1)}}
      - {{argv $i}}
      {{end}}
```

### Template Variables
- `{{.Name}}` - command name (existing, backward compatible)
- `{{.Args}}` - raw argument string after command name
- `{{.Argv}}` - parsed argument array including command name at index 0

### Template Functions
- `{{argc}}` - count of actual arguments (excluding command name)
- `{{argv N}}` - Nth argument (0=command name, 1=first arg, etc.)

## Implementation Notes

### File Locations
- **New file**: `internal/cli/args.go` - argument parsing utilities
- **Modified**: `internal/template/context.go` - extend CommandContext
- **Modified**: `internal/template/functions.go` - add argc/argv functions
- **Modified**: `internal/template/template.go` - update ExecuteCommandTemplate
- **Modified**: `internal/cli/commands.go` - parse arguments in ProcessUserPrompt

### Backward Compatibility
- Existing commands without arguments continue to work unchanged
- Existing template variables (`{{.Name}}`) remain functional
- Template processing maintains same behavior for commands without args

### Future Extensions Ready
- Named arguments: `{{arg "name"}}` for `--name=value` syntax
- Boolean flags: `{{flag "verbose"}}` for `--verbose` syntax
- Option parsing: `{{opt "output"}}` for `--output filename` syntax

## Progress Tracking

**Legend**: ❌ Not Started | 🟡 In Progress | ✅ Complete | 🧪 Testing | 📝 Documented

### Current Status: ✅ COMPLETE - All Phases Implemented

This document should be updated throughout implementation to track progress and note any design changes or issues encountered.

---

## Implementation Summary

**Feature completed successfully on 2025-08-25**

### What Was Built

1. **Complete Argument Support Pipeline**
   - Argument parsing with quote handling (`ParseCommandArgs`)
   - Template context extension with `Args`/`Argv` fields
   - Template functions `argc` and `argv` with safety checks
   - Command processing integration with backward compatibility

2. **Key Technical Achievements**
   - Zero breaking changes to existing functionality
   - Comprehensive test coverage (unit, integration, e2e)
   - Robust error handling for edge cases
   - Production-ready documentation and examples

3. **User-Facing Features**
   - Commands now accept arguments: `$search "term" directory`
   - Template variables: `{{.Name}}`, `{{.Args}}`, `{{.Argv}}`
   - Template functions: `{{argc}}` and `{{argv N}}`
   - Conditional logic based on argument count/values
   - Rich configuration examples in `bumpers.yml`

### Files Modified

**Core Implementation:**
- `internal/cli/args.go` - NEW: Argument parsing utilities
- `internal/template/context.go` - Extended CommandContext
- `internal/template/functions.go` - Added argc/argv functions  
- `internal/template/template.go` - Added ExecuteCommandTemplateWithArgs
- `internal/cli/commands.go` - Updated ProcessUserPrompt for arguments

**Tests Added:**
- `internal/template/functions_test.go` - Template function tests
- `internal/template/template_test.go` - Template execution tests
- `internal/cli/app_test.go` - Integration tests

**Documentation:**
- `bumpers.yml` - Added argument examples
- `CLAUDE.md` - Added command argument documentation

### Live Examples Working

```bash
# Search with arguments
echo '{"prompt": "$search testing internal"}' | bumpers hook
# → "grep -r \"testing\" internal/"

# File creation with type checking  
echo '{"prompt": "$create file test.go"}' | bumpers hook
# → "touch \"test.go\""

# Dynamic responses based on argument count
echo '{"prompt": "$note working on features"}' | bumpers hook  
# → "Note (3 args): working on features"
```

**Status**: ✅ PRODUCTION READY - Feature complete and fully tested