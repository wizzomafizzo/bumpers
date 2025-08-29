# Reliable Intent Extraction Without tool_use_id

## Executive Summary

This document outlines the implementation plan for fixing intent extraction in Bumpers when Claude Code doesn't provide `tool_use_id` in hook events. The current implementation assumes the presence of `tool_use_id` but real Claude Code hooks don't include this field, making intent extraction fail silently.

## Problem Analysis

### Current State Issues

1. **Missing tool_use_id**: Claude Code hook events don't include `tool_use_id` field despite our implementation expecting it
2. **Unreliable Fallbacks**: Current fallback methods (scanning arbitrary line counts) are unreliable and produce inconsistent results
3. **Silent Failures**: When `tool_use_id` is missing, the system falls back to unreliable methods without clear indication

### Evidence from Transcript Analysis

Real Claude Code transcripts show:
```json
// Assistant message with tool use
{"type":"assistant","uuid":"47f131a2-0cc9-44ed-8388-08e119d3fac2","message":{
  "content":[{"type":"tool_use","id":"toolu_01Q2gScpPso9vR2d8LBprDeQ","name":"Read"}]
}}

// System message (bumpers hook completion)
{"type":"system","uuid":"3317f2c1-53ba-478f-ab65-7d99a54c2acf",
 "parentUuid":"47f131a2-0cc9-44ed-8388-08e119d3fac2",
 "toolUseID":"toolu_01Q2gScpPso9vR2d8LBprDeQ"}

// User message (tool result)
{"type":"user","uuid":"087c6c34-387b-4e47-8971-11020707f79f",
 "parentUuid":"3317f2c1-53ba-478f-ab65-7d99a54c2acf"}
```

However, the hook event received by Bumpers only contains:
```json
{
  "session_id": "54bef80d-1f38-4740-833e-00cdb8b0d569",
  "transcript_path": "/.../54bef80d-1f38-4740-833e-00cdb8b0d569.jsonl",
  "tool_name": "Read",
  "tool_input": {...}
}
```

**Key Missing Information**: No `tool_use_id`, no UUIDs, no timestamp of the hook trigger.

## Solution Strategy

### Core Approach: Recent Tool Use Detection

Since we cannot rely on `tool_use_id` being provided, we will:

1. **Scan the transcript backwards** from the most recent entries
2. **Match tool usage** by tool name, timestamp, and optionally input parameters  
3. **Extract the tool_use_id** from the matching transcript entry
4. **Use existing parent-child logic** with the found `tool_use_id`

### Why This Works

- **Temporal Locality**: The hook is triggered immediately after tool use, so the matching entry will be very recent
- **Unique Identification**: Tool name + timestamp provides sufficient uniqueness within a short time window
- **Leverages Existing Logic**: Once we find the `tool_use_id`, we can use our proven parent-child extraction method

## Detailed Implementation Plan

### Phase 1: Remove Unreliable Methods

**Files to Modify**: 
- `internal/platform/claude/transcript/reader.go`
- `internal/platform/claude/transcript/reader_test.go`
- `internal/cli/app.go`

**Actions**:
1. Delete `ExtractIntentContentOptimized` function and all supporting helper functions
2. Remove test cases for the unreliable methods
3. Clean up any references in documentation

**Rationale**: These methods scan arbitrary line counts and produce inconsistent results. Better to have no fallback than an unreliable one.

### Phase 2: Implement FindRecentToolUseAndExtractIntent

**Location**: `internal/platform/claude/transcript/reader.go`

**Function Signature**:
```go
func FindRecentToolUseAndExtractIntent(
    transcriptPath string, 
    toolName string, 
    toolInput map[string]any, 
    timeWindow time.Duration,
) (string, error)
```

**Implementation Strategy**:

#### Step 1: Read Recent Transcript Entries
```go
func readRecentTranscriptEntries(file *os.File, maxEntries int) ([]TranscriptEntry, error) {
    // Implementation options:
    // Option A: Read entire file and take last N entries (simple, works for small files)
    // Option B: Seek from end and read backwards (complex, better for large files)
    // 
    // Recommendation: Start with Option A for simplicity, optimize later if needed
}
```

#### Step 2: Find Matching Tool Use
```go
func findMatchingToolUse(
    entries []TranscriptEntry, 
    toolName string, 
    toolInput map[string]any,
    timeWindow time.Duration,
) (string, error) {
    now := time.Now()
    
    // Scan entries from newest to oldest
    for _, entry := range entries {
        // Check if within time window
        if entryTime, err := time.Parse(time.RFC3339, entry.Timestamp); err == nil {
            if now.Sub(entryTime) > timeWindow {
                continue // Too old
            }
        }
        
        // Check if it's an assistant message with tool_use
        if entry.Type != "assistant" {
            continue
        }
        
        // Look for matching tool use in content
        for _, content := range entry.Message.Content {
            if content.Type == "tool_use" && 
               matchesTool(content, toolName, toolInput) {
                return content.ID, nil // Found the tool_use_id!
            }
        }
    }
    
    return "", nil // No matching tool use found
}
```

#### Step 3: Tool Matching Logic
```go
func matchesTool(content ContentItem, toolName string, toolInput map[string]any) bool {
    // Basic check: tool name must match
    if content.Name != toolName {
        return false
    }
    
    // Enhanced matching (optional): Check distinctive input parameters
    // This helps when multiple tools of same type are used quickly
    if len(toolInput) > 0 && content.Input != nil {
        // Match key parameters like file_path, command, etc.
        return matchesInput(content.Input, toolInput)
    }
    
    return true
}
```

### Phase 3: Update Main Intent Extraction Logic

**Location**: `internal/cli/app.go`

**Modify**: `checkIntentSource` function

**Before**:
```go
func (a *App) checkIntentSource(...) {
    // ... existing logic
    var intentContent string
    var err error

    if event.ToolUseID != "" {
        intentContent, err = transcript.ExtractIntentByToolUseID(...)
    } else {
        intentContent, err = transcript.ExtractIntentContent(...)
    }
    // ... rest
}
```

**After**:
```go
func (a *App) checkIntentSource(...) {
    // ... existing logic
    var intentContent string
    var err error

    if event.ToolUseID != "" {
        // Test data path - use provided tool_use_id
        intentContent, err = transcript.ExtractIntentByToolUseID(event.TranscriptPath, event.ToolUseID)
    } else {
        // Real Claude Code path - find tool use from transcript
        intentContent, err = transcript.FindRecentToolUseAndExtractIntent(
            event.TranscriptPath,
            event.ToolName,
            event.ToolInput,
            60*time.Second, // 1-minute window
        )
    }
    
    // NO fallback to unreliable methods - if we can't find intent, return empty
    if err != nil {
        log.Debug().Err(err).
            Str("transcript_path", event.TranscriptPath).
            Str("tool_name", event.ToolName).
            Msg("Failed to extract intent - no reliable method available")
        return false, ""
    }
    
    // ... rest of function
}
```

### Phase 4: Update PostToolUse Processing

**Location**: `internal/cli/app.go`

**Modify**: `extractPostToolContent` function

**Current Issue**: Uses `ExtractIntentContentOptimized` fallback

**Solution**: Apply same logic as PreToolUse:

```go
func (*App) extractPostToolContent(...) {
    // ... existing extraction logic
    
    if transcriptPath != "" {
        var intent string
        var err error

        if toolUseID != "" {
            // Test data path
            intent, err = transcript.ExtractIntentByToolUseID(transcriptPath, toolUseID)
        } else {
            // Real Claude Code path - extract tool info from rawJSON
            intent, err = transcript.FindRecentToolUseAndExtractIntent(
                transcriptPath,
                toolName,
                extractToolInputFromRawJSON(rawJSON), // Helper function needed
                60*time.Second,
            )
        }
        
        // Log but don't fail if intent extraction fails
        if err != nil {
            logger.Debug().Err(err).
                Str("transcript_path", transcriptPath).
                Str("tool_name", toolName).
                Msg("Failed to extract intent for PostToolUse")
        }
        
        content.intent = intent
    }
    
    return content, nil
}
```

### Phase 5: Configuration and Tuning

**Add Configuration Options**:

```go
// In config package
type IntentExtractionConfig struct {
    TimeWindowSeconds    int  `yaml:"time_window_seconds" json:"time_window_seconds"`
    MaxTranscriptEntries int  `yaml:"max_transcript_entries" json:"max_transcript_entries"`
    EnableInputMatching  bool `yaml:"enable_input_matching" json:"enable_input_matching"`
}

// Defaults
var DefaultIntentExtractionConfig = IntentExtractionConfig{
    TimeWindowSeconds:    60,   // 1 minute
    MaxTranscriptEntries: 100,  // Last 100 entries
    EnableInputMatching:  true, // Match tool input for disambiguation
}
```

### Phase 6: Comprehensive Testing

#### Unit Tests

**Test File**: `internal/platform/claude/transcript/reader_test.go`

**Test Cases**:
1. **Basic Functionality**:
   ```go
   func TestFindRecentToolUseAndExtractIntent_BasicMatch(t *testing.T)
   func TestFindRecentToolUseAndExtractIntent_NoMatch(t *testing.T)
   func TestFindRecentToolUseAndExtractIntent_TimeWindowExpired(t *testing.T)
   ```

2. **Edge Cases**:
   ```go
   func TestFindRecentToolUseAndExtractIntent_MultipleToolsSameName(t *testing.T)
   func TestFindRecentToolUseAndExtractIntent_MalformedTranscript(t *testing.T)
   func TestFindRecentToolUseAndExtractIntent_EmptyTranscript(t *testing.T)
   ```

3. **Input Matching**:
   ```go
   func TestFindRecentToolUseAndExtractIntent_InputMatching(t *testing.T)
   func TestFindRecentToolUseAndExtractIntent_InputMismatch(t *testing.T)
   ```

#### Integration Tests

**Test File**: `internal/cli/app_test.go`

**Test Cases**:
1. **Without tool_use_id**:
   ```go
   func TestProcessHookWithoutToolUseID(t *testing.T)
   func TestPostToolUseWithoutToolUseID(t *testing.T)
   ```

2. **Real Transcript Formats**:
   ```go
   func TestRealTranscriptFormatCompatibility(t *testing.T)
   ```

#### End-to-End Tests

**Test File**: `cmd/bumpers/hook_e2e_test.go`

**Modifications**:
1. Create test scenarios without `tool_use_id` in hook JSON
2. Use real transcript formats from `~/.claude` examples
3. Verify intent extraction works correctly

### Phase 7: Performance Considerations

#### Memory Usage
- **Buffer Size**: Default to last 100 transcript entries (~10-50KB typical)
- **Large File Handling**: For transcripts >1MB, consider streaming approach

#### CPU Usage  
- **JSON Parsing**: Parse only entries within time window
- **String Matching**: Use efficient string comparison for tool names
- **Timestamp Parsing**: Cache parsed timestamps to avoid repeated parsing

#### Network/Disk I/O
- **File Access**: Single file read per intent extraction
- **Caching**: Consider caching recent entries for rapid successive tool uses

## Success Criteria

### Functional Requirements
1. ✅ **Accurate Intent Extraction**: Correctly extracts intent when `tool_use_id` is not provided
2. ✅ **Backward Compatibility**: Continues to work with test data that includes `tool_use_id`
3. ✅ **No False Positives**: Does not extract intent from unrelated tool uses
4. ✅ **Graceful Degradation**: Returns empty string instead of incorrect content when matching fails

### Performance Requirements
1. ✅ **Fast Response**: Intent extraction completes within 100ms for typical transcripts
2. ✅ **Memory Efficient**: Uses <50MB memory for transcript processing
3. ✅ **Scalable**: Performance degrades linearly with transcript size, not exponentially

### Reliability Requirements
1. ✅ **Error Handling**: Graceful handling of malformed transcripts
2. ✅ **Logging**: Clear debug information for troubleshooting
3. ✅ **Deterministic**: Same input always produces same output

## Risk Mitigation

### Risk 1: Multiple Tools of Same Type in Quick Succession
**Impact**: Medium - Could match wrong tool use
**Mitigation**: 
- Implement tool input parameter matching
- Use shortest time window that still captures the correct tool use
- Add detailed debug logging to identify mismatches

### Risk 2: Large Transcript Files
**Impact**: Medium - Performance degradation
**Mitigation**:
- Limit scanning to last 100 entries by default
- Implement configurable limits
- Consider streaming/seeking for very large files

### Risk 3: Timestamp Parsing Failures
**Impact**: Low - Fallback to position-based matching
**Mitigation**:
- Handle timestamp parsing errors gracefully
- Use entry position as secondary sort criteria
- Log timestamp parsing issues for monitoring

### Risk 4: Transcript Format Changes
**Impact**: High - Complete extraction failure
**Mitigation**:
- Robust JSON parsing with error handling
- Version detection for transcript format changes
- Comprehensive test suite with various transcript formats

## Deployment Strategy

### Phase 1: Development and Testing
- Implement core functionality
- Unit test coverage >90%
- Integration testing with real transcript samples

### Phase 2: Internal Testing
- Deploy to development environment
- Test with various Claude Code versions
- Monitor performance and accuracy metrics

### Phase 3: Gradual Rollout
- Feature flag for new vs old extraction methods
- A/B testing with subset of users
- Monitor error rates and performance

### Phase 4: Full Deployment
- Remove old unreliable methods
- Update documentation
- Monitor production metrics

## Monitoring and Alerting

### Key Metrics
1. **Intent Extraction Success Rate**: % of hook events where intent was successfully extracted
2. **Extraction Latency**: Time taken for intent extraction
3. **Fallback Usage**: Frequency of different extraction methods used
4. **Error Rates**: Rate of transcript parsing or matching errors

### Alerts
1. **Low Success Rate**: Alert if success rate drops below 95%
2. **High Latency**: Alert if P95 latency exceeds 500ms
3. **High Error Rate**: Alert if error rate exceeds 1%

## Future Enhancements

### Short Term (Next Sprint)
1. **Tool Input Matching**: Implement sophisticated parameter matching for disambiguation
2. **Caching**: Add in-memory cache for recently parsed transcript entries
3. **Configuration**: Add user-configurable time windows and limits

### Medium Term (Next Quarter)
1. **Performance Optimization**: Implement backward file seeking for very large transcripts
2. **Machine Learning**: Use ML to improve tool use matching accuracy
3. **Analytics**: Add detailed metrics on extraction patterns and accuracy

### Long Term (Future Releases)
1. **Claude Code Integration**: Work with Claude Code team to include `tool_use_id` in hook events
2. **Alternative Transport**: Consider WebSocket or streaming updates for real-time tool use detection
3. **Context Enhancement**: Use conversation context to improve intent extraction accuracy

---

## Appendix A: Example Transcript Structures

### Real Claude Code Transcript Entry
```json
{
  "parentUuid": "a6d0d9d5-e144-4c48-8a05-bf475ba38116",
  "isSidechain": false,
  "userType": "external",
  "cwd": "/home/callan/dev/staydata/staydata-server",
  "sessionId": "54bef80d-1f38-4740-833e-00cdb8b0d569",
  "version": "1.0.89",
  "gitBranch": "testing-improvement-implementation",
  "message": {
    "id": "msg_01KNZzrD2mDFCQnJqGzS4ZdK",
    "type": "message",
    "role": "assistant",
    "model": "claude-sonnet-4-20250514",
    "content": [
      {
        "type": "tool_use",
        "id": "toolu_01Q2gScpPso9vR2d8LBprDeQ",
        "name": "Read",
        "input": {
          "file_path": "/home/callan/dev/staydata/staydata-server/internal/documents/search.go",
          "offset": 145,
          "limit": 10
        }
      }
    ]
  },
  "requestId": "req_011CSZkMK9bb7CdKtTUaad3C",
  "type": "assistant",
  "uuid": "47f131a2-0cc9-44ed-8388-08e119d3fac2",
  "timestamp": "2025-08-28T07:55:03.623Z"
}
```

### Hook Event Received by Bumpers
```json
{
  "hook": {
    "session_id": "54bef80d-1f38-4740-833e-00cdb8b0d569",
    "transcript_path": "/home/callan/.claude/projects/-home-callan-dev-staydata-staydata-server/54bef80d-1f38-4740-833e-00cdb8b0d569.jsonl",
    "cwd": "/home/callan/dev/staydata/staydata-server",
    "hook_event_name": "PreToolUse",
    "tool_name": "Read",
    "tool_input": {
      "file_path": "/home/callan/dev/staydata/staydata-server/internal/documents/search.go",
      "offset": 145,
      "limit": 30
    }
  }
}
```

## Appendix B: Code Examples

### Example Tool Input Matching
```go
func matchesInput(contentInput interface{}, hookInput map[string]any) bool {
    contentMap, ok := contentInput.(map[string]interface{})
    if !ok {
        return false
    }
    
    // Check key distinctive parameters
    distinctiveFields := []string{"file_path", "command", "query", "url"}
    
    for _, field := range distinctiveFields {
        if hookVal, hookExists := hookInput[field]; hookExists {
            if contentVal, contentExists := contentMap[field]; contentExists {
                if fmt.Sprintf("%v", hookVal) == fmt.Sprintf("%v", contentVal) {
                    return true // Found matching distinctive parameter
                }
            }
        }
    }
    
    return false // No distinctive parameters matched
}
```

### Example Time Window Configuration
```go
func (c *Config) GetIntentExtractionTimeWindow() time.Duration {
    if c.IntentExtraction.TimeWindowSeconds > 0 {
        return time.Duration(c.IntentExtraction.TimeWindowSeconds) * time.Second
    }
    return 60 * time.Second // Default 1 minute
}
```