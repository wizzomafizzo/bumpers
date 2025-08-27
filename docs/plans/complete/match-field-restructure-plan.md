# Match Field Restructure Plan

## Overview
Restructure the `match` field in rules to support both string (simple) and struct (advanced) forms, similar to how the `generate` field currently works. This change moves `event` and `sources` fields from the rule level into an optional match struct.

**CRITICAL**: NO BACKWARD COMPATIBILITY - This is a breaking change for an unreleased feature.

## Implementation Checklist

### Phase 1: Core Structure Changes

#### 1. Update Config Types (`internal/config/config.go`)
- [ ] Change `Rule.Match` from `string` to `any` (interface{})
- [ ] **REMOVE** `Event string` field from Rule struct
- [ ] **REMOVE** `Sources []string` field from Rule struct
- [ ] Create new `Match` struct:
  ```go
  type Match struct {
      Pattern string   `yaml:"pattern" mapstructure:"pattern"`
      Event   string   `yaml:"event,omitempty" mapstructure:"event"`
      Sources []string `yaml:"sources,omitempty" mapstructure:"sources"`
  }
  ```
- [ ] Add `GetMatch()` method to Rule struct
- [ ] **UPDATE DOCS**: Update inline comments in config.go

#### 2. Implement Match Parsing (`internal/config/config.go`)
- [ ] Create `parseMatchField()` function modeled after `parseGenerateField()`
  - [ ] Handle nil case: return error (match is required)
  - [ ] Handle string case: `Match{Pattern: str, Event: "pre", Sources: []}`
  - [ ] Handle map case: parse pattern, event, sources with defaults
  - [ ] Default event to "pre" if not specified
  - [ ] Default sources to empty slice (matches all non-meta fields)
- [ ] Implement `GetMatch()` method that calls `parseMatchField()`
- [ ] **UPDATE DOCS**: Add detailed comments explaining the parsing logic

### Phase 2: Update Validation

#### 3. Fix Validation Logic (`internal/config/config.go`)
- [ ] **REMOVE** `ValidateEventSources()` method completely
- [ ] Update `validateRequiredFields()`:
  - [ ] Check that Match is not nil
  - [ ] Get Match struct via `GetMatch()`
  - [ ] Validate Pattern is not empty
- [ ] Update `validateRegexPatterns()`:
  - [ ] Use `GetMatch().Pattern` instead of `r.Match`
- [ ] Update `Validate()` method:
  - [ ] Remove call to `ValidateEventSources()`
  - [ ] Add validation for Match.Event ("pre" or "post" only)
  - [ ] No validation for Sources (any field name is valid)
- [ ] **UPDATE DOCS**: Update validation error messages

### Phase 3: Update Matcher

#### 4. Update Matcher Implementation (`internal/matcher/matcher.go`)
- [ ] Update `NewRuleMatcher()`:
  - [ ] Use `rules[i].GetMatch().Pattern` for validation
- [ ] Update `Match()` method:
  - [ ] Use `m.rules[i].GetMatch().Pattern` instead of `m.rules[i].Match`
- [ ] **UPDATE DOCS**: Update any comments in matcher.go

### Phase 4: Update CLI Application

#### 5. Update CLI App (`internal/cli/app.go`)
- [ ] Find ALL occurrences of `rule.Event` and replace with `rule.GetMatch().Event`
- [ ] Find ALL occurrences of `rule.Sources` and replace with `rule.GetMatch().Sources`
- [ ] Update `processPreToolUse()`:
  - [ ] Line ~216: Use `rule.GetMatch().Event` for filtering
- [ ] Update `checkSourcesForMatch()`:
  - [ ] Line ~241: Use `rule.GetMatch().Sources`
- [ ] Update `checkSpecificSources()`:
  - [ ] Line ~251: Use `rule.GetMatch().Sources`
- [ ] Update `processPostToolUse()`:
  - [ ] Line ~394: Use `rule.GetMatch().Event`
  - [ ] Line ~403: Use `rule.GetMatch().Sources`
  - [ ] Line ~407: Use `rule.GetMatch().Sources`
  - [ ] Line ~420: Use `rule.GetMatch().Sources`
- [ ] **UPDATE DOCS**: Ensure all function comments are updated

### Phase 5: Update Tests

#### 6. Update Config Tests (`internal/config/config_test.go`)
- [ ] **REMOVE** `TestEventSourcesDefaults` test entirely
- [ ] **REMOVE** `TestEventSourcesValidation` test entirely
- [ ] **REMOVE** `TestSourcesAreOptional` test entirely
- [ ] **REMOVE** `TestArbitrarySourceNamesAllowed` test entirely
- [ ] Add new test `TestMatchFieldParsing`:
  - [ ] Test string form: `match: "pattern"`
  - [ ] Test struct form with all fields
  - [ ] Test struct form with defaults
  - [ ] Test invalid cases
- [ ] Update ALL existing test configs:
  - [ ] Remove `event:` at rule level
  - [ ] Remove `sources:` at rule level
  - [ ] Use either string match or struct match
- [ ] **UPDATE DOCS**: Add comments explaining test cases

#### 7. Update CLI App Tests (`internal/cli/app_test.go`)
- [ ] Update `TestPostEventRulesDontMatchPreHooks` (line ~1707):
  - [ ] Change from `event: "post"` to `match: {pattern: "ls", event: "post"}`
- [ ] Update `TestPreToolUseWithSources` (line ~1734):
  - [ ] Change from `sources: ["command"]` to `match: {pattern: "delete", sources: ["command"]}`
- [ ] Update ALL post-tool-use test configs (lines ~2874-3100+):
  - [ ] Convert all rules to use new match struct format
- [ ] Search for ALL occurrences of `event:` and `sources:` in test strings
- [ ] **UPDATE DOCS**: Ensure test names and comments reflect new structure

### Phase 6: Update Documentation

#### 8. Update CLAUDE.md
- [ ] **REMOVE** entire "Event Types" section (line ~143)
- [ ] **REMOVE** entire "Sources Configuration" section (line ~148)
- [ ] **REMOVE** "Post-Tool-Use Hooks" section references to old format
- [ ] Add new "Match Field Configuration" section:
  - [ ] Explain string vs struct forms
  - [ ] Show examples of both formats
  - [ ] Document defaults (event="pre", sources=[])
  - [ ] Explain special meta sources (#intent)
- [ ] Update ALL example configs throughout the file:
  - [ ] Line ~157-180: Update rule examples
  - [ ] Line ~187-199: Update post-hook examples
- [ ] **UPDATE DOCS**: Ensure consistent terminology throughout

#### 9. Update Test Data
- [ ] Check `testdata/configs/` directory:
  - [ ] Update any configs using old event/sources format
  - [ ] Add new example configs showing both match forms
- [ ] **UPDATE DOCS**: Add comments in example configs

#### 10. Update Other Documentation
- [ ] Search for references in other docs:
  - [ ] `docs/plans/complete/` - Update completed plans if referenced
  - [ ] `TESTING.md` - Update if any examples use old format
  - [ ] `README.md` - Update if configuration is mentioned
- [ ] **UPDATE DOCS**: Keep all docs in sync

### Phase 7: Testing & Verification

#### 11. Run Tests Progressively
- [ ] After Phase 1-2: Run `just test-unit ./internal/config`
- [ ] After Phase 3: Run `just test-unit ./internal/matcher`
- [ ] After Phase 4: Run `just test-unit ./internal/cli`
- [ ] After Phase 5: Run `just test` (all tests)
- [ ] Run `just lint` to catch any issues
- [ ] **UPDATE DOCS**: Document any issues found and fixes applied

#### 12. Manual Testing
- [ождать Test string match form works as expected
- [ ] Test struct match form with all fields
- [ ] Test struct match form with defaults
- [ ] Test that old format configs FAIL with clear error
- [ ] Test pre-hook rules work correctly
- [ ] Test post-hook rules work correctly
- [ ] Test source filtering works correctly
- [ ] **UPDATE DOCS**: Add any edge cases discovered to tests

### Phase 8: Final Review

#### 13. Code Review Checklist
- [ ] No references to `rule.Event` remain (only `rule.GetMatch().Event`)
- [ ] No references to `rule.Sources` remain (only `rule.GetMatch().Sources`)
- [ ] All tests pass
- [ ] Linting passes
- [ ] Documentation is fully updated
- [ ] Example configs all use new format
- [ ] **UPDATE DOCS**: Create migration guide for users (even though no backward compat)

## New Configuration Format

### Simple Form (String)
```yaml
rules:
  - match: "rm -rf"  # Defaults: event="pre", sources=[]
    send: "Use safer deletion"
```

### Advanced Form (Struct)
```yaml
rules:
  # Post-hook rule with intent matching
  - match:
      pattern: "not related to my changes"
      event: "post"
      sources: ["#intent"]
    send: "AI claiming unrelated"
    
  # Pre-hook rule with specific field matching
  - match:
      pattern: "password|secret"
      event: "pre"  # Optional, defaults to "pre"
      sources: ["command", "content"]
    send: "Avoid hardcoding secrets"
    
  # Minimal struct form
  - match:
      pattern: "production"
      sources: ["url"]  # event defaults to "pre"
    send: "Production URL detected"
```

## Important Notes

1. **NO BACKWARD COMPATIBILITY** - Old format must not work
2. **Update docs immediately** as each phase is completed
3. **Test after each phase** to catch issues early
4. **Commit after each major phase** for clean history
5. **Use `just test-unit` for fast feedback during development**

## Expected Errors for Old Format

Users trying old format should see clear errors:
- `event: "post"` at rule level → "unknown field 'event' in rule"
- `sources: ["command"]` at rule level → "unknown field 'sources' in rule"

## Success Criteria

- [ ] All tests pass with new format
- [ ] No references to old fields remain in code
- [ ] Documentation fully updated
- [ ] Clean git history with logical commits
- [ ] Feature branch ready for PR