# PostToolUse Hook Implementation Plan

## 🎯 Project Overview

**Feature:** Post-Tool-Use Hook Support with AI Reasoning Matching  
**Started:** 2025-08-25  
**Current Phase:** 3 (PostToolUse Handler Implementation)  

### Core Innovation
Match AI reasoning patterns like "not related to my changes" immediately after tool execution to catch AI deflection or misattribution of issues.

---

## 📋 Live Todo List

### ✅ Phase 1: Configuration Structure (COMPLETED)
- [x] Add `When []string` field to Rule struct in `internal/config/config.go`
- [x] Implement `ExpandWhen()` method with smart defaults
- [x] Support exclusion syntax with `!` prefix
- [x] Add comprehensive tests for When field expansion
- [x] Verify backward compatibility (empty When defaults to `["pre", "input"]`)

### ✅ Phase 2: Hook Processing Updates (COMPLETED)
- [x] Add PostToolUse case to `ProcessHook()` in `internal/cli/app.go`
- [x] Fix hook detection logic in `internal/hooks/hooks.go`
- [x] Create `ProcessPostToolUse()` method stub
- [x] Add basic routing test `TestProcessHookRoutesPostToolUse`
- [x] Ensure existing hook detection tests still pass

### 🚧 Phase 3: PostToolUse Handler Implementation (75% COMPLETE)
- [x] ✅ Parse PostToolUse JSON events
- [x] ✅ Extract transcript path from event data
- [x] ✅ Implement fail-safe transcript reading
- [x] ✅ Create static testdata transcript files
- [x] ✅ Basic pattern matching against transcript content
- [x] ✅ Return appropriate messages based on matches
- [ ] ❌ **IN PROGRESS:** Integrate with rule matching system
- [ ] ❌ **TODO:** Implement `when` field processing
- [ ] ❌ **TODO:** Add regex pattern matching (currently using string contains)
- [ ] ❌ **TODO:** Support tool output matching in addition to reasoning

**Current Status:**
```
✅ Basic functionality working with hardcoded patterns
✅ Fail-safe design implemented (graceful failure on transcript issues)
✅ Tests passing for known patterns
❌ Rule matching system not integrated
❌ Only supports transcript content, not tool output
```

### ❌ Phase 4: Transcript Reader Module (NOT STARTED)
- [ ] Create `internal/transcript/reader.go`
- [ ] Implement `ReadLastAssistantText(path string) (string, error)`
- [ ] Add JSONL parsing with malformed line handling
- [ ] Implement efficient tail-based reading for large files
- [ ] Extract assistant messages from transcript structure
- [ ] Add comprehensive error handling and logging
- [ ] Create test suite for transcript parsing

### ❌ Phase 5: Rule Matching Integration (NOT STARTED)
- [ ] Update `internal/matcher/matcher.go` to support When field expansion
- [ ] Add context parameter for different content sources (input, output, reasoning)
- [ ] Implement `matchPostToolUseRules()` function
- [ ] Support regex matching against transcript content
- [ ] Add tool output matching for `when: ["post"]` rules
- [ ] Maintain API compatibility with existing matcher interface

### ❌ Phase 6: Comprehensive Testing (15% COMPLETE)
- [x] ✅ Basic PostToolUse routing test
- [x] ✅ Static testdata transcript files
- [ ] ❌ **TODO:** Config validation tests for When field
- [ ] ❌ **TODO:** Rule matching tests with different When configurations
- [ ] ❌ **TODO:** Transcript reader unit tests
- [ ] ❌ **TODO:** Integration tests for end-to-end PostToolUse processing
- [ ] ❌ **TODO:** Error handling tests (missing files, malformed JSON)
- [ ] ❌ **TODO:** Performance tests with large transcript files

---

## 🔧 Technical Implementation Details

### Current File Structure
```
internal/
├── cli/
│   ├── app.go                    # PostToolUse routing + basic handler ✅
│   └── app_test.go              # Basic tests ✅
├── config/
│   ├── config.go                # When field + ExpandWhen() ✅ 
│   └── config_test.go           # When field tests ✅
├── hooks/
│   ├── hooks.go                 # PostToolUse detection ✅
│   └── hooks_test.go            # Hook detection tests ✅
└── transcript/                  # ❌ NOT CREATED YET
    ├── reader.go                # ❌ Transcript parsing module
    └── reader_test.go           # ❌ Transcript tests
testdata/
├── transcript-permission-denied.jsonl  # ✅ Test transcript
└── transcript-not-related.jsonl        # ✅ Test transcript
```

### Smart Defaults Implementation
```go
// Current ExpandWhen() logic in internal/config/config.go
func (r *Rule) ExpandWhen() []string {
    if len(r.When) == 0 {
        return []string{"pre", "input"}  // Backward compatibility
    }
    // Smart defaults:
    // ["reasoning"] → ["post", "reasoning"]
    // ["post"] → ["post", "output"] 
    // ["pre"] → ["pre", "input"]
    // ["!flag"] → Exclude flag from expanded set
}
```

### Current PostToolUse Handler
```go
// Current implementation in internal/cli/app.go
func (a *App) ProcessPostToolUse(rawJSON json.RawMessage) (string, error) {
    // ✅ Parse JSON event
    // ✅ Extract transcript path
    // ✅ Read transcript with fail-safe design
    // ✅ Basic pattern matching
    // ❌ TODO: Integrate with rule matching system
    // ❌ TODO: Process When field to determine matching scope
}
```

---

## 🧪 Test Cases Status

### ✅ Passing Tests
1. `TestProcessHookRoutesPostToolUse` - Basic routing and transcript matching
2. `TestPostToolUseWithDifferentTranscript` - Different content patterns
3. All existing hook detection tests - Backward compatibility

### ❌ Failing/Missing Tests
1. `TestPostToolUseRuleNotMatching` - Should return empty when patterns don't match
2. Tests for When field processing with different rule configurations
3. Tests for tool output matching (not just transcript reasoning)
4. Error handling tests for missing/corrupted transcripts
5. Performance tests with large transcript files

### 🎯 Example Use Cases to Test
```yaml
# AI deflection detection
- match: "(not related|pre-existing|unrelated) to (my|the) changes"
  when: ["reasoning"]
  send: "AI claiming changes unrelated - please verify"

# Command failure monitoring  
- match: "error|failed|exit code [^0]"
  when: ["post"]  # tool output only
  send: "Command failed - review the output"

# Mixed matching with exclusions
- match: "production|database"
  when: ["pre", "post", "!reasoning"]  # tool data only, not AI explanations
  send: "Sensitive operation detected"
```

---

## 🚀 Next Actions

### Immediate (Next Session)
1. **Fix failing test:** Make `TestPostToolUseRuleNotMatching` pass by implementing proper rule pattern matching
2. **Integrate rule system:** Connect PostToolUse handler with config rules and When field processing
3. **Add regex support:** Replace hardcoded string matching with proper regex matching

### Short Term (This Week)
1. **Complete Phase 3:** Full PostToolUse handler with rule integration
2. **Implement Phase 4:** Create transcript reader module with robust JSONL parsing
3. **Add tool output matching:** Support `when: ["post"]` rules for tool response content

### Medium Term (Next Week)
1. **Comprehensive testing:** Full test suite covering all scenarios
2. **Performance optimization:** Efficient transcript reading for large files
3. **Documentation:** Update README and API docs

---

## 🏆 Success Criteria

### Functional Requirements
- [x] ✅ PostToolUse hooks properly routed and processed
- [ ] ❌ Rules with `when: ["reasoning"]` match AI explanations from transcripts
- [ ] ❌ Rules with `when: ["post"]` match tool output/response  
- [x] ✅ Backward compatibility maintained
- [ ] ❌ Smart defaults work as specified
- [ ] ❌ Exclusions with `!` prefix work correctly

### Non-Functional Requirements
- [x] ✅ **Fail-safe:** Missing transcripts don't break functionality
- [ ] ❌ **Performance:** Efficient transcript reading with large files
- [ ] ❌ **Logging:** Clear warnings when reasoning rules are skipped
- [x] ✅ **Future-proof:** Works with different AI providers
- [x] ✅ **Testable:** Clean separation with comprehensive test coverage

### Quality Gates
- [ ] ❌ All tests passing (currently 2/3 PostToolUse tests pass)
- [ ] ❌ >75% code coverage for new modules
- [ ] ❌ No hardcoded patterns in production code
- [ ] ❌ Proper error handling and logging
- [ ] ❌ Performance benchmarks for transcript reading

---

## 📊 Progress Tracking

**Overall Completion:** 45%

- [x] **Phase 1:** Configuration Structure (100%) 
- [x] **Phase 2:** Hook Processing Updates (100%)
- [ ] **Phase 3:** PostToolUse Handler (75%) 🚧 **CURRENT**
- [ ] **Phase 4:** Transcript Reader (0%)
- [ ] **Phase 5:** Rule Matching Integration (0%) 
- [ ] **Phase 6:** Comprehensive Testing (15%)

**Blockers:**
- Need to implement proper rule matching to fix failing test
- Hardcoded pattern logic needs to be replaced with config-driven matching

**Risks:**
- Complex When field logic may introduce bugs
- Large transcript files could impact performance
- Different AI providers may have different transcript formats

---

*Last Updated: 2025-08-25*  
*Next Review: After completing rule matching integration*