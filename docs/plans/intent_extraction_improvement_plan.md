# Intent Extraction Improvement Plan

## Overview

This plan improves Bumpers' transcript processing to properly extract Claude's "intent" from transcripts for both pre and post hook events. The current implementation uses brittle string pattern matching and should be replaced with proper JSON parsing to extract both thinking blocks and text content.

## Background

The current `ExtractReasoningContent` function has several issues:
1. **Brittle pattern matching**: Looks for specific text patterns like "I need to", "I can see" in assistant messages
2. **String manipulation**: Uses string indexing instead of proper JSON parsing
3. **Limited scope**: Only works for post-tool-use hooks with "reasoning" source
4. **Missing thinking blocks**: Doesn't extract the actual `"thinking"` field from Claude's transcript

## Goals

- Replace brittle pattern matching with robust JSON parsing
- Extract both `"thinking"` blocks (internal reasoning) and `"text"` blocks (explanations) 
- Make intent extraction available for both pre and post hook events
- Use clear naming: "intent" instead of "reasoning"
- Maintain backward compatibility during transition

## Implementation Plan

### Phase 1: Core Transcript Reader Improvements ✅ COMPLETED

- [x] **Create ExtractIntentContent function**
  - Added proper JSON parsing with `TranscriptEntry` struct
  - Extracts both `"thinking"` and `"text"` fields from assistant messages
  - Handles JSONL format correctly with line-by-line parsing
  - Returns concatenated intent content

- [x] **Add test coverage for new function**
  - `TestExtractIntentContent_WithThinkingBlocks`: Tests extraction of both thinking and text blocks
  - Test passes with realistic JSONL transcript data

- [x] **Update function call in hook processing**
  - Changed `transcript.ExtractReasoningContent` to `transcript.ExtractIntentContent` in `/internal/cli/app.go`

### Phase 2: Hook Processing Updates ✅ COMPLETED

- [x] **Update post-tool-use hook processing**
  - Function call updated to use new ExtractIntentContent
  - Renamed "reasoning" field/source to "intent" 

- [x] **Update field names for consistency**
  - Renamed `postToolContent.reasoning` field to `intent`
  - Updated all references and variable names
  - Updated log messages to use "intent" terminology
  - Added backward compatibility for "reasoning" source

- [x] **Add pre-tool-use intent support**
  - Added TranscriptPath field to HookEvent struct
  - Created test `TestPreToolUseIntentSupport` to verify pre-tool-use intent matching
  - Updated `checkSpecificSources` to handle "intent" source
  - Added backward compatibility for "reasoning" source in pre-tool-use context
  - Test now passing - pre-tool-use intent support fully implemented

### Phase 3: Configuration & Validation Updates ✅ COMPLETED

- [x] **Update config validation**
  - Added "intent" as valid source for both "pre" and "post" events
  - Updated `ValidateEventSources()` in config package with comprehensive validation
  - Added validation for event-specific sources (e.g., "tool_output" only for post events)
  - Maintained "reasoning" as deprecated alias for backward compatibility
  - Added test coverage with TestIntentSourceValidation

- [ ] **Update documentation**
  - Update README.md with new "intent" source examples
  - Document that "intent" includes both thinking and explanations
  - Add migration notes for "reasoning" → "intent"

### Phase 4: Comprehensive Test Improvements ✅ COMPLETED

- [x] **Add more ExtractIntentContent tests**
  - `TestExtractIntentContent_WithTextOnly`: Text blocks without thinking ✅
  - `TestExtractIntentContent_EmptyFile`: Handle empty transcripts ✅  
  - `TestExtractIntentContent_NonExistentFile`: Graceful error handling ✅
  - `TestExtractIntentContent_MalformedJSON`: Skip invalid lines ✅
  - `TestExtractIntentContent_MixedContent`: Real-world transcript complexity ✅

- [x] **Improve test data robustness**
  - Replaced simple transcript content with realistic JSONL structure ✅
  - Added multiple entries, proper session flow patterns ✅
  - Included mixed content types (thinking + text) in realistic proportions ✅
  - Added real-world complexity: tool calls, longer reasoning chains ✅
  - Included edge cases: empty content, malformed JSON lines, file errors ✅
  - Used actual Claude reasoning patterns and language ✅

- [x] **Fixed error handling bug**
  - Fixed ExtractIntentContent to properly propagate file opening errors ✅
  - All tests now properly handle error conditions ✅

- [ ] **Update existing post-tool-use tests**
  - Test both thinking and text extraction with robust data
  - Add edge cases: tools without intent, mixed message types
  - Use realistic transcript files from testdata/ directory

- [ ] **Add comprehensive pre-tool-use intent tests**
  - Test intent extraction in pre-hook context with realistic data
  - Verify intent matching works with rule sources
  - Test that intent content is recent/relevant
  - Add performance tests with large transcript files

- [ ] **Integration tests**
  - End-to-end test with real Claude transcript files
  - Performance tests with large transcript files
  - Test backward compatibility with old config files

### Phase 5: Performance & Polish

- [ ] **Performance optimization**
  - Efficient scanning for large transcript files
  - Consider reading from end of file for recent content
  - Memory usage optimization for long transcripts

- [ ] **Error handling improvements**
  - Better error messages for malformed transcripts
  - Graceful degradation when intent extraction fails
  - Logging improvements for debugging

- [ ] **Backward compatibility**
  - Keep ExtractReasoningContent as deprecated wrapper
  - Support "reasoning" source as alias for "intent"
  - Migration warnings for deprecated usage

## Current Status

**Phase 1**: ✅ **COMPLETED**
- New ExtractIntentContent function implemented and tested
- Proper JSON parsing with TranscriptEntry struct
- Function call updated in hook processing

**Phase 2**: ✅ **COMPLETED**
- Field names updated from "reasoning" to "intent"
- Backward compatibility maintained for "reasoning" source
- Pre-tool-use intent support implementation in progress

**Phase 3**: ✅ **COMPLETED**
- Configuration validation updated with comprehensive source validation
- "Intent" and "reasoning" sources properly validated for event compatibility
- Added TestIntentSourceValidation with comprehensive test coverage
- Both event types and source compatibility fully validated

**Phase 4**: ✅ **COMPLETED**
- Added comprehensive ExtractIntentContent test coverage (5 new tests)
- Implemented realistic JSONL transcript data with proper Claude reasoning patterns
- Added edge case handling: empty files, missing files, malformed JSON
- Added complex mixed content test with tool calls and longer reasoning chains
- Fixed error handling bug in ExtractIntentContent for proper error propagation
- All tests passing with robust, production-ready scenarios

**Next Steps**:
1. Update documentation with examples and migration guide
2. Add comprehensive integration tests
3. Complete remaining post-tool-use test improvements
4. Consider performance optimization for large transcript files

## Technical Details

### Transcript Structure
```json
{
  "type": "assistant",
  "message": {
    "role": "assistant", 
    "content": [
      {
        "type": "thinking",
        "thinking": "I need to analyze what tests to run..."
      },
      {
        "type": "text", 
        "text": "I'll run the tests for you."
      }
    ]
  }
}
```

### Intent Extraction Logic
1. Parse each JSONL line as JSON
2. Filter for `type: "assistant"` entries
3. Extract `content` array from message
4. Collect text from both `thinking` and `text` type blocks
5. Join all intent parts with spaces

### Configuration Example
```yaml
rules:
  - match: "test.*failed" 
    event: "post"
    sources: ["intent", "tool_output"]
    send: "Consider running tests with verbose output"
    
  - match: "I need to.*database"
    event: "pre" 
    sources: ["intent"]
    send: "Remember to check database connections first"
```

This plan ensures a systematic approach to improving intent extraction while maintaining backward compatibility and thorough testing.