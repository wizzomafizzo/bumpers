# Bumpers Post-Tool-Use Hook Support - Project Plan

## 📋 Overview

**Feature:** Add Post-Tool-Use Hook Support with AI Reasoning Matching  
**Status:** 🚧 In Progress  
**Started:** 2025-08-25  
**Priority:** High  

### Goals
- Enable post-tool-use hook processing in Bumpers
- Allow matching against AI reasoning/explanations from transcript files
- Implement fail-safe design for optional transcript support
- Maintain full backward compatibility
- Use intuitive `when` field with smart defaults and exclusions

### Key Innovation
Match problematic AI patterns like "not related to my changes" right after tools execute, catching AI deflection or misattribution of issues.

---

## 🏗️ Implementation Plan

### Phase 1: Configuration Structure
**Status:** ✅ **COMPLETED**

#### Task 1.1: Extend Rule Config Structure
- **File:** `internal/config/config.go`
- **Changes:**
  - Add `When []string` field to Rule struct
  - Implement smart defaults logic
  - Support `!` prefix for exclusions
  - Maintain backward compatibility

**Smart Default Rules:**
- Omitted `when` → `["pre", "input"]` (current behavior)  
- `["reasoning"]` → `["post", "reasoning"]` (reasoning implies post)
- `["post"]` → `["post", "output"]` (post defaults to output)
- `["pre"]` → `["pre", "input"]` (pre defaults to input)
- `["!flag"]` → Exclude flag from expanded set

**Example Configuration:**
```yaml
rules:
  # Traditional (backward compatible)
  - pattern: "^rm -rf"
    tools: "^Bash$"
    message: "Use safer deletion"
    # when: ["pre", "input"] - implicit
    
  # AI reasoning match (smart default)
  - pattern: "(not related|pre-existing) to (my|the) changes"
    tools: ".*"
    when: ["reasoning"]  # → ["post", "reasoning"]
    message: "AI claiming changes unrelated - verify"
    
  # Tool output only
  - pattern: "error|failed"
    when: ["post"]  # → ["post", "output"]
    message: "Command failed - review error"
    
  # Exclusion example
  - pattern: "exit code"
    when: ["post", "!reasoning"]  # post+output, exclude reasoning
    message: "Check exit code in output"
```

---

### Phase 2: Hook Processing Updates
**Status:** ✅ **COMPLETED**

#### Task 2.1: Update Main Hook Router
- **File:** `internal/cli/app.go`
- **Function:** `ProcessHook` (lines 142-210)
- **Changes:**
  - Add case for `PostToolUseHook` after line 170
  - Route to new `ProcessPostToolUse` handler
  - Maintain existing pre-tool-use logic

#### Task 2.2: PostToolUse Hook Data Structure
- **Reference Format** (from Claude Code docs):
```json
{
  "session_id": "abc123",
  "transcript_path": "/path/to/transcript.jsonl",
  "cwd": "/current/working/directory", 
  "hook_event_name": "PostToolUse",
  "tool_name": "Write",
  "tool_input": { /* tool-specific */ },
  "tool_response": { /* tool-specific */ }
}
```

---

### Phase 3: Post-Tool-Use Handler
**Status:** 🚧 **IN PROGRESS** - Basic routing complete, full implementation pending

#### Task 3.1: Create PostToolUse Processor
- **File:** `internal/cli/posttooluse.go` (new file)
- **Functions:**
  - `ProcessPostToolUse(rawJSON json.RawMessage) (string, error)`
  - `matchPostToolUseRules(event, rules, matcher) (Rule, string, error)`

**Critical Requirements:**
- ✅ Parse PostToolUse JSON structure  
- ✅ Check rules where `when` contains "post"
- ✅ For "reasoning" rules: attempt transcript read, skip gracefully if unavailable
- ✅ For "output" rules: match against tool_response
- ✅ Log warnings for skipped reasoning rules
- ✅ Never fail entire hook due to missing transcript

**Fail-Safe Design:**
```go
func extractReasoningFromTranscript(path string) (string, error) {
    // Quick existence check - fail fast
    if _, err := os.Stat(path); err != nil {
        log.Debug().Str("path", path).Msg("Transcript unavailable, skipping reasoning")
        return "", nil  // Empty string, not error
    }
    
    // Attempt read - don't fail hook on error
    content, err := readLastNLines(path, 100)
    if err != nil {
        log.Warn().Err(err).Msg("Could not read transcript")
        return "", nil  // Continue processing other rules
    }
    
    return extractAssistantText(content), nil
}
```

---

### Phase 4: Transcript Reader (Optional Component)
**Status:** ❌ Pending

#### Task 4.1: Create Transcript Parser
- **File:** `internal/transcript/reader.go` (new file)
- **Functions:**
  - `ReadLastAssistantText(path string) (string, error)`
  - `parseJSONLLines(lines []string) []AssistantMessage`
  - `readLastNLines(path string, n int) ([]string, error)`

**Design Principles:**
- ✅ **Optional by design** - entire module can fail without breaking core functionality
- ✅ **Fast failure** - check file existence first
- ✅ **Efficient reading** - use tail approach for large files
- ✅ **Graceful parsing** - skip malformed JSONL lines
- ✅ **Recent context** - extract last assistant text before current timestamp

**Implementation Strategy:**
1. Use `tail -n 100` command for efficiency
2. Parse JSONL line by line (skip errors)
3. Extract assistant messages with text content
4. Return most recent text before current time
5. Return empty string on any failure

---

### Phase 5: Matcher Updates
**Status:** ❌ Pending

#### Task 5.1: When Field Expansion Logic
- **File:** `internal/matcher/matcher.go`
- **New Function:** `expandWhenFlags(when []string) []string`

**Logic Flow:**
```go
func expandWhenFlags(when []string) []string {
    if len(when) == 0 {
        return []string{"pre", "input"}  // backward compatible default
    }
    
    expanded := make(map[string]bool)
    excludes := make(map[string]bool)
    
    // Process flags and exclusions
    for _, flag := range when {
        if strings.HasPrefix(flag, "!") {
            excludes[strings.TrimPrefix(flag, "!")] = true
        } else {
            expanded[flag] = true
            // Apply smart defaults
            switch flag {
            case "reasoning":
                expanded["post"] = true
            case "post":
                if !hasSourceFlag(when) { expanded["output"] = true }
            case "pre":
                if !hasSourceFlag(when) { expanded["input"] = true }
            }
        }
    }
    
    // Remove excluded items
    for exclude := range excludes {
        delete(expanded, exclude)
    }
    
    return mapKeysToSlice(expanded)
}
```

#### Task 5.2: Update Matcher Interface
- **File:** `internal/matcher/matcher.go`
- **Changes:**
  - Add context parameter to matching functions
  - Support different content sources (input, output, reasoning)
  - Maintain existing API compatibility

---

### Phase 6: Comprehensive Testing
**Status:** ❌ Pending

#### Task 6.1: Configuration Tests
- **File:** `internal/config/config_test.go`
- Test `When` field parsing
- Test smart default expansion
- Test exclusion logic with `!` prefix
- Test backward compatibility

#### Task 6.2: Hook Processing Tests  
- **File:** `internal/cli/app_test.go`
- Test PostToolUse hook routing
- Test rule matching with different `when` configurations
- Test graceful failure when transcript unavailable

#### Task 6.3: Transcript Reader Tests
- **File:** `internal/transcript/reader_test.go`
- Test JSONL parsing with valid/invalid lines
- Test file reading with missing files
- Test assistant text extraction
- Test performance with large files

#### Task 6.4: Integration Tests
- **File:** `internal/cli/posttooluse_test.go`
- Test end-to-end PostToolUse processing
- Test reasoning rule matching
- Test output rule matching  
- Test mixed rule scenarios

---

## 🎯 Success Criteria

### Functional Requirements
- ✅ PostToolUse hooks properly routed and processed
- ✅ Rules with `when: ["reasoning"]` match AI explanations from transcripts  
- ✅ Rules with `when: ["post"]` match tool output/response
- ✅ Backward compatibility: existing rules work unchanged
- ✅ Smart defaults work as specified
- ✅ Exclusions with `!` prefix work correctly

### Non-Functional Requirements  
- ✅ **Fail-safe:** Missing transcripts don't break core functionality
- ✅ **Performance:** Transcript reading uses tail approach, minimal overhead
- ✅ **Logging:** Clear warnings when reasoning rules skipped
- ✅ **Future-proof:** Works with AIs that don't provide transcripts
- ✅ **Maintainable:** Clean separation of concerns, testable components

### Example Use Cases
1. **AI Deflection Detection:**
   ```yaml
   - pattern: "(not related|pre-existing|unrelated) to (my|the) changes"
     when: ["reasoning"]
     message: "AI claiming changes unrelated - please verify"
   ```

2. **Command Failure Monitoring:**
   ```yaml
   - pattern: "error|failed|exit code [^0]"
     when: ["post"]
     message: "Command failed - review the output"
   ```

3. **Sensitive Operations:**
   ```yaml
   - pattern: "production|database"
     when: ["pre", "post", "!reasoning"]  # tool data only, not explanations
     message: "Sensitive operation detected"
   ```

---

## 🚀 Next Steps

### Immediate Actions (Today)
1. ✅ **[IN PROGRESS]** Write this comprehensive plan ← Current task
2. ❌ Implement `When` field in config structure
3. ❌ Add PostToolUse case to hook router
4. ❌ Create basic PostToolUse handler skeleton

### This Week
- Complete core PostToolUse processing
- Implement transcript reader with fail-safe design
- Update matcher with When field expansion
- Add basic tests

### Next Week  
- Comprehensive testing suite
- Documentation updates
- Integration testing
- Performance optimization

---

## 📝 Notes & Decisions

### Design Decisions Made
1. **List-based `when` field** - More intuitive than single string with sub-syntax
2. **Smart defaults** - Reduce verbosity for common cases
3. **Fail-safe transcript support** - Core functionality works without transcripts  
4. **`!` prefix for exclusions** - Familiar syntax for developers
5. **Tail-based reading** - Efficient for large transcript files

### Open Questions
- None currently - design approved and ready for implementation

### Risks & Mitigations
- **Risk:** Large transcript files slow down processing  
  **Mitigation:** Use tail approach, limit to last 100 lines
- **Risk:** Different AI providers have different transcript formats  
  **Mitigation:** Fail-safe design, skip reasoning rules if transcript unavailable
- **Risk:** Complex `when` syntax confuses users  
  **Mitigation:** Smart defaults make simple cases simple

---

## 📊 Progress Tracking

**Overall Progress:** 35% (Phases 1-2 Complete)

- [x] Requirements gathering and design (100%)
- [x] Technical design and architecture (100%) 
- [x] Detailed implementation plan (100%)
- [x] Config structure implementation (100%) ✅ **COMPLETED**
- [x] Hook processing updates (100%) ✅ **COMPLETED**
- [ ] PostToolUse handler (25%) 🚧 **IN PROGRESS** - Basic routing implemented
- [ ] Transcript reader (0%)
- [ ] Matcher updates (0%)
- [ ] Testing suite (15%) - Basic PostToolUse test added
- [ ] Documentation (0%)
- [ ] Integration testing (0%)

**Last Updated:** 2025-08-25  
**Next Review:** After Phase 3 completion

## 🎯 Recent Accomplishments

### ✅ Phase 1: Config Structure (COMPLETED)
- Added `When []string` field to Rule struct in `internal/config/config.go`
- Implemented `ExpandWhen()` method with smart defaults:
  - `["reasoning"]` → `["post", "reasoning"]`
  - `["post"]` → `["post", "output"]` 
  - `["pre"]` → `["pre", "input"]`
- Added exclusion support with `!` prefix: `["post", "!reasoning"]`
- Added comprehensive tests in `internal/config/config_test.go`
- All config tests passing (89.5% coverage)

### ✅ Phase 2: Hook Processing (COMPLETED)  
- Added PostToolUse case to `ProcessHook()` in `internal/cli/app.go`
- Fixed hook detection logic in `internal/hooks/hooks.go` to properly detect PostToolUse events (events with both `tool_input` and `tool_output`)
- Added basic `ProcessPostToolUse()` stub method
- Added test `TestProcessHookRoutesPostToolUse` in `internal/cli/app_test.go`
- All hook detection tests still passing

### 🚧 Phase 3: PostToolUse Handler (25% COMPLETE)
- ✅ Basic hook routing implemented
- ✅ Test framework established  
- ❌ **NEXT:** Implement transcript reading and rule matching logic
- ❌ **NEXT:** Add fail-safe design for missing transcripts
- ❌ **NEXT:** Integrate with matcher for When field processing

---

*This document serves as the living project plan and TODO list for the PostToolUse hook feature. Update progress as work completes.*