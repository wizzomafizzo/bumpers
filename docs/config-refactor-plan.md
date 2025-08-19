# Configuration Structure Refactor Plan

## Overview
This document outlines the planned changes to refactor the Bumpers configuration structure to be more intuitive and align with Claude Code's pattern matching conventions.

## Current vs Proposed Configuration Structure

### Current Structure
```yaml
rules:
  - name: "block-go-test"
    pattern: "go test.*"           # Regex pattern
    action: "deny"
    message: "Use make test instead"
    alternatives:                   # Separate field for alternatives
      - "make test"
      - "make test-unit"
    use_claude: false               # Boolean
    claude_prompt: "..."            # Prefixed field name
```

### Proposed Structure
```yaml
rules:
  - name: "block-go-test"
    pattern: "go test*"             # Glob-style pattern
    action: "deny"
    response: |                     # Renamed from 'message'
      Use make test instead for better TDD integration
      
      Try one of these alternatives:
      • make test          # Run all tests
      • make test-unit     # Run unit tests only
    use_claude: "no"                # String value
    prompt: "..."                   # Simplified field name
```

## Detailed Changes

### 1. Config Field Updates (`internal/config/config.go`)

#### Remove Alternatives Field
- **Current**: `Alternatives []string` - Separate array field
- **Proposed**: Remove this field entirely
- **Rationale**: Alternatives can be included as part of the response text, giving more flexibility in formatting

#### Rename Message to Response
- **Current**: `Message string`
- **Proposed**: `Response string`
- **Rationale**: "Response" better describes what the field contains - the full response shown to the user

#### Change UseClaude Type
- **Current**: `UseClaude bool`
- **Proposed**: `UseClaude string` with values:
  - `"no"` (default)
  - `"yes"`
  - Future options: `"auto"`, `"fallback"`, `"enhanced"`
- **Rationale**: String type allows for future expansion of Claude integration modes

#### Rename ClaudePrompt to Prompt
- **Current**: `ClaudePrompt string`
- **Proposed**: `Prompt string`
- **Rationale**: Simpler, cleaner field name; context makes it clear it's for Claude

### 2. Pattern Matching System (`internal/matcher/`)

#### New Pattern Types

##### Exact Match
```yaml
pattern: "make test"
```
- Matches: `make test`
- Doesn't match: `make test-unit`, `make testing`

##### Wildcard Pattern
```yaml
pattern: "go test*"
```
- Matches: `go test`, `go test ./...`, `go test -v`
- Doesn't match: `make go test`

##### OR Operator
```yaml
pattern: "npm|yarn|pnpm"
```
- Matches: `npm`, `yarn`, `pnpm`
- Doesn't match: `npm install`, `yarn add`

##### Combined Patterns
```yaml
pattern: "npm *|yarn *|pnpm *"
```
- Matches: `npm install`, `yarn add`, `pnpm update`
- Doesn't match: `npm`, `bun install`

#### Implementation Details

```go
// New matcher logic pseudo-code
func matchPattern(pattern, command string) bool {
    // Split pattern by OR operator
    orPatterns := strings.Split(pattern, "|")
    
    for _, p := range orPatterns {
        p = strings.TrimSpace(p)
        
        // Check if pattern contains wildcards
        if strings.Contains(p, "*") {
            // Convert to glob pattern matching
            if globMatch(p, command) {
                return true
            }
        } else {
            // Exact match
            if p == command {
                return true
            }
        }
    }
    return false
}
```

### 3. Test File Updates

#### Files to Update:
- `internal/config/config_test.go`
  - Remove tests for Alternatives field
  - Update field names in test YAML
  - Add tests for string-based UseClaude values
  
- `internal/matcher/matcher_test.go`
  - Add tests for glob patterns
  - Add tests for OR operator
  - Test edge cases (empty patterns, special characters)
  
- `internal/cli/app_test.go`
  - Update all embedded YAML configs
  - Test new response formatting
  
- `internal/response/response_test.go`
  - Update field references from Message to Response

### 4. Migration Examples

#### Before:
```yaml
rules:
  - name: "dangerous-rm"
    pattern: "rm -rf /.*"
    action: "deny"
    message: "Dangerous rm command detected"
    alternatives:
      - "Be more specific with your rm command"
      - "Use trash-cli instead"
    use_claude: true
    claude_prompt: "Explain why this rm command is dangerous"
```

#### After:
```yaml
rules:
  - name: "dangerous-rm"
    pattern: "rm -rf /*"
    action: "deny"
    response: |
      ⚠️  Dangerous rm command detected
      
      This command could delete important system files!
      
      Safer alternatives:
      • Be more specific with your rm command
      • Use trash-cli for recoverable deletions
      • Double-check the path before using -rf flags
    use_claude: "yes"
    prompt: "Explain why this rm command is dangerous and suggest safer alternatives"
```

## Implementation Plan

### Phase 1: Core Structure Changes
1. Update `config.Rule` struct with new field names and types
2. Add backward compatibility layer (optional, for smooth migration)
3. Update config loading/validation logic

### Phase 2: Pattern Matching
1. Create new pattern matching module
2. Implement glob-style matching with wildcard support
3. Add OR operator support using pipe character
4. Replace regex matcher with new implementation
5. Add comprehensive test coverage

### Phase 3: Test Updates
1. Update all test files with new field names
2. Add tests for new pattern matching behavior
3. Ensure all existing tests pass with modifications
4. Add migration/compatibility tests if needed

### Phase 4: Documentation & Examples
1. Update `configs/bumpers.yaml` with new syntax
2. Update `CLAUDE.md` with new configuration format
3. Create migration guide for existing users
4. Add pattern matching reference documentation

### Phase 5: Enhanced Claude Integration (Future)
With `use_claude` as a string, future modes could include:
- `"auto"` - Automatically use Claude for complex scenarios
- `"fallback"` - Use Claude only if local response is insufficient
- `"enhanced"` - Always append Claude's insights to local response
- `"interactive"` - Allow Claude to ask follow-up questions

## Testing Strategy

### Unit Tests
- Pattern matching edge cases
- Config parsing with new fields
- Backward compatibility (if implemented)

### Integration Tests
- End-to-end command blocking with new patterns
- Claude integration with string-based configuration
- Response formatting with embedded alternatives

### Test Cases for Pattern Matching
```yaml
test_cases:
  - pattern: "go test"
    matches: ["go test"]
    non_matches: ["go test ./...", "make go test"]
    
  - pattern: "go test*"
    matches: ["go test", "go test ./...", "go testing"]
    non_matches: ["make go test", "test go"]
    
  - pattern: "npm|yarn|pnpm"
    matches: ["npm", "yarn", "pnpm"]
    non_matches: ["npm install", "bun"]
    
  - pattern: "rm -rf /*|rm -fr /*"
    matches: ["rm -rf /", "rm -rf /usr", "rm -fr /home"]
    non_matches: ["rm -rf ./", "rm /tmp/file"]
```

## Benefits of This Refactor

1. **Simplified Configuration**: Fewer fields, more intuitive structure
2. **Flexible Responses**: Freeform text allows better formatting and context
3. **Future-Proof**: String-based `use_claude` field allows expansion
4. **Familiar Patterns**: Glob-style matching aligns with Claude Code and common CLI tools
5. **Better UX**: More natural pattern writing without regex complexity

## Risks and Mitigations

### Risk: Breaking Existing Configurations
**Mitigation**: 
- Provide clear migration guide
- Consider temporary backward compatibility layer
- Offer automated migration script

### Risk: Pattern Matching Edge Cases
**Mitigation**:
- Comprehensive test suite
- Clear documentation with examples
- Fallback to exact matching for unclear patterns

### Risk: Loss of Regex Power
**Mitigation**:
- Most use cases don't need complex regex
- Could add `pattern_type: "regex"` field for advanced users
- Glob + OR operator covers 95% of use cases

## Timeline Estimate

- Phase 1 (Core Structure): 2-3 hours
- Phase 2 (Pattern Matching): 3-4 hours  
- Phase 3 (Test Updates): 2-3 hours
- Phase 4 (Documentation): 1-2 hours
- Phase 5 (Future Enhancement): Separate project

**Total Estimate**: 8-12 hours of development time

## Success Criteria

- [x] All tests pass with new structure
- [x] Configuration is more intuitive to write
- [x] Pattern matching covers common use cases
- [x] Default partial matching implemented (new requirement)
- [x] Documentation clearly explains changes
- [x] Existing users can easily migrate
- [x] Future Claude integration modes are possible

## Updated Pattern Matching Behavior

**New Requirement**: Default to partial matching for intuitive behavior:

- `"go test"` matches any command containing "go test" (e.g., "make go test", "go test ./...", "some go test command")
- `"go test*"` matches commands starting with "go test" (glob behavior)
- `"npm|yarn|pnpm"` matches any command containing any of these terms

This provides more intuitive behavior while still supporting explicit glob patterns when needed.