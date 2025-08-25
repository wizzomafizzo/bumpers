# Command Arguments Implementation Plan

**Feature**: Add argument support to commands defined in config file  
**Issue**: #14  
**Status**: Planning Complete, Ready for Implementation  
**Last Updated**: 2025-08-25

## Overview

Enable commands to accept and parse arguments from user prompts, making them available in templates through variables and functions. This extends the existing `$commandName` syntax to support `$commandName arg1 "arg with spaces" arg3`.

## Implementation Checklist

### Phase 1: Core Infrastructure
- [ ] **Argument parsing function** (new: `internal/cli/args.go`)
  - [ ] Parse space-separated arguments respecting quoted strings
  - [ ] Handle empty arguments gracefully
  - [ ] Support both single and double quotes
  - [ ] Test edge cases: empty strings, escaped quotes, mixed quotes
  
- [ ] **Extend CommandContext** (`internal/template/context.go`)
  - [ ] Add `Args` field (string) - raw arguments after command name
  - [ ] Add `Argv` field ([]string) - parsed arguments including command at index 0
  - [ ] Update `BuildCommandContext` function signature to accept arguments
  - [ ] Maintain backward compatibility with existing `Name` field

### Phase 2: Template Functions
- [ ] **Template function additions** (`internal/template/functions.go` or new file)
  - [ ] `argc` function - returns count of arguments (excluding command name)
  - [ ] `argv` function - accepts index, returns argument at that position
    - [ ] `argv 0` returns command name
    - [ ] `argv 1`, `argv 2`, etc. return actual arguments
    - [ ] Out-of-bounds access returns empty string (no error)
    - [ ] Support variadic parameters for future extensibility
  - [ ] Register functions in `createFuncMap`

### Phase 3: Command Processing Updates
- [ ] **Modify `ProcessUserPrompt`** (`internal/cli/commands.go`)
  - [ ] Parse command name and arguments from full command string
  - [ ] Pass arguments to template execution
  - [ ] Update command matching to use only the command name part
  - [ ] Ensure logging includes argument information for debugging

- [ ] **Update template execution** (`internal/template/template.go`)
  - [ ] Modify `ExecuteCommandTemplate` to accept command name and arguments separately
  - [ ] Update function signature and all call sites

### Phase 4: Testing
- [ ] **Unit tests for argument parsing**
  - [ ] Test basic space-separated arguments
  - [ ] Test quoted arguments with spaces
  - [ ] Test mixed quotes and escaping
  - [ ] Test empty argument lists
  - [ ] Test malformed input handling

- [ ] **Unit tests for template functions**
  - [ ] Test `argc` with various argument counts
  - [ ] Test `argv` with valid indices
  - [ ] Test `argv` with out-of-bounds indices
  - [ ] Test `argv 0` returns command name

- [ ] **Integration tests**
  - [ ] Test full command processing with arguments
  - [ ] Test template rendering with argument variables
  - [ ] Test backward compatibility with existing commands

- [ ] **Update existing tests**
  - [ ] Modify tests that call `ExecuteCommandTemplate`
  - [ ] Ensure existing command tests still pass

### Phase 5: Documentation & Examples
- [ ] **Update configuration examples**
  - [ ] Add example commands using argument features
  - [ ] Document template syntax in comments
  
- [ ] **Update project documentation**
  - [ ] Add argument support to CLAUDE.md
  - [ ] Include usage examples in appropriate docs

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

### Current Status: ❌ Not Started

This document should be updated throughout implementation to track progress and note any design changes or issues encountered.

---

**Next Steps**: Begin with Phase 1 - implement argument parsing function and extend CommandContext structure.