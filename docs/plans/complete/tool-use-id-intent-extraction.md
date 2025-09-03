# Tool Use ID Based Intent Extraction Plan

## Problem Statement

The current intent extraction system in Bumpers has fundamental reliability issues:

1. **Arbitrary Line Limits**: Uses `ExtractIntentContentOptimized` which reads the last 50 lines from transcript files
2. **Unpredictable Results**: If intent is too small, pulls in old unrelated content; if too large, misses parts
3. **No Correlation**: No proper way to associate hook events with specific tool executions in the transcript
4. **Prone to Failure**: No telling how large intent fields could be, causing frequent mismatches

## Investigation Findings

### Current Claude Code Hook Flow

Through investigation of actual transcript files (`~/.claude/projects/`), the following structure was discovered:

1. **Assistant Intent Message**: Contains the reasoning text
   ```json
   {
     "type": "assistant",
     "uuid": "ca7ed6ef-67bd-4719-bc9b-cf3c2b22579d",
     "message": {
       "content": [{"type": "text", "text": "I'll help you address the intent logging issues..."}]
     }
   }
   ```

2. **Assistant Tool Use Message**: Contains the actual tool invocation
   ```json
   {
     "type": "assistant", 
     "uuid": "12453064-5276-4cb5-826c-d1b636867f71",
     "parentUuid": "ca7ed6ef-67bd-4719-bc9b-cf3c2b22579d",
     "message": {
       "content": [{"type": "tool_use", "id": "toolu_01KTePc3uLq34eriLmSLbgnx", "name": "TodoWrite"}]
     }
   }
   ```

3. **System Hook Messages**: Reference the tool use ID
   ```json
   {
     "type": "system",
     "toolUseID": "toolu_01KTePc3uLq34eriLmSLbgnx",
     "content": "PreToolUse:TodoWrite [bumpers hook] completed successfully"
   }
   ```

### Key Discovery: Correlation is Possible

- Each tool execution has a unique `tool_use_id` (e.g., `toolu_01KTePc3uLq34eriLmSLbgnx`)
- The intent message is the assistant message that is the parent of the tool_use message
- This provides exact correlation between hook events and intent content

## Proposed Solution

### Architecture Overview

Replace the current arbitrary line-limit approach with precise tool-use-ID based extraction:

```
Hook Event → tool_use_id → Find tool_use in transcript → Get parent message → Extract intent
```

### Implementation Plan

#### 1. Update Hook Structure

**File**: `internal/core/engine/hooks/hooks.go`

```go
type HookEvent struct {
    ToolInput      map[string]any `json:"tool_input"`      //nolint:tagliatelle // API uses snake_case
    ToolName       string         `json:"tool_name"`       //nolint:tagliatelle // API uses snake_case
    TranscriptPath string         `json:"transcript_path"` //nolint:tagliatelle // API uses snake_case
    ToolUseID      string         `json:"tool_use_id"`     // NEW: Add tool use correlation ID
}
```

**Assumptions**: 
- Claude Code may need to be updated to include `tool_use_id` in hook payloads
- If not available, we'll extract it from the transcript by finding the most recent tool_use entry

#### 2. Create Targeted Intent Extraction Function

**File**: `internal/platform/claude/transcript/reader.go`

```go
// ExtractIntentByToolUseID extracts the intent for a specific tool use ID
// by finding the assistant message that precedes the tool_use message
func ExtractIntentByToolUseID(transcriptPath string, toolUseID string) (string, error) {
    // Algorithm:
    // 1. Open transcript file and read as JSONL
    // 2. Parse each line to find assistant message with tool_use where content[].id == toolUseID
    // 3. Extract the parentUuid from that message
    // 4. Find the assistant message with uuid == parentUuid
    // 5. Extract text content from that parent message
    // 6. Return the intent text (or empty string if not found)
    
    file, err := os.Open(transcriptPath)
    if err != nil {
        return "", fmt.Errorf("failed to open transcript file %s: %w", transcriptPath, err)
    }
    defer file.Close()

    var targetToolUseMessage *TranscriptEntry
    var intentMessage *TranscriptEntry
    
    reader := bufio.NewReader(file)
    for {
        line, err := reader.ReadString('\n')
        if err != nil && err != io.EOF {
            return "", fmt.Errorf("error reading transcript: %w", err)
        }
        if line == "" && err == io.EOF {
            break
        }
        
        var entry TranscriptEntry
        if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &entry); err != nil {
            continue // Skip malformed lines
        }
        
        // Look for tool_use message with matching ID
        if entry.Type == "assistant" && targetToolUseMessage == nil {
            if content := findToolUseWithID(entry, toolUseID); content != nil {
                targetToolUseMessage = &entry
            }
        }
        
        // Look for parent intent message
        if targetToolUseMessage != nil && entry.Type == "assistant" && 
           entry.UUID == targetToolUseMessage.ParentUUID {
            intentMessage = &entry
            break
        }
        
        if err == io.EOF {
            break
        }
    }
    
    if intentMessage != nil {
        return extractTextContent(intentMessage), nil
    }
    
    return "", nil // No intent found for this tool use ID
}

// Helper structures for parsing transcript entries
type TranscriptEntry struct {
    Type       string          `json:"type"`
    UUID       string          `json:"uuid"`
    ParentUUID string          `json:"parentUuid"`
    Message    MessageContent  `json:"message"`
}

type MessageContent struct {
    Content []ContentItem `json:"content"`
}

type ContentItem struct {
    Type string `json:"type"`
    Text string `json:"text,omitempty"`
    ID   string `json:"id,omitempty"`   // For tool_use content
    Name string `json:"name,omitempty"` // For tool_use content
}
```

#### 3. Update PreToolUse Processing

**File**: `internal/cli/app.go`

```go
// extractAndLogIntent extracts and logs intent content from transcript
func (*App) extractAndLogIntent(event hooks.HookEvent) {
    if event.TranscriptPath == "" {
        return
    }
    
    var intent string
    var err error
    
    if event.ToolUseID != "" {
        // Use precise tool-use-ID based extraction
        intent, err = transcript.ExtractIntentByToolUseID(event.TranscriptPath, event.ToolUseID)
        if err != nil {
            log.Debug().Err(err).
                Str("tool_use_id", event.ToolUseID).
                Str("transcript_path", event.TranscriptPath).
                Msg("Failed to extract intent by tool use ID")
            return
        }
    } else {
        // tool_use_id not available - log and skip
        log.Debug().
            Str("transcript_path", event.TranscriptPath).
            Msg("No tool_use_id available for precise intent extraction")
        return
    }
    
    if intent != "" {
        log.Debug().
            Str("tool_use_id", event.ToolUseID).
            Str("extracted_intent", intent).
            Msg("Extracted intent for PreToolUse using tool_use_id")
    }
}
```

#### 4. Update PostToolUse Processing

**File**: `internal/cli/app.go`

```go
func (*App) extractPostToolContent(ctx context.Context, rawJSON json.RawMessage) (*postToolContent, error) {
    logger := zerolog.Ctx(ctx)

    // Parse the JSON to get transcript path and tool info
    var event map[string]any
    if err := json.Unmarshal(rawJSON, &event); err != nil {
        return nil, fmt.Errorf("failed to unmarshal post-tool-use event: %w", err)
    }

    transcriptPath, _ := event["transcript_path"].(string)
    toolName, _ := event["tool_name"].(string)
    toolUseID, _ := event["tool_use_id"].(string)  // NEW: Extract tool use ID
    toolResponse := event["tool_response"]

    content := &postToolContent{
        toolName:      toolName,
        toolOutputMap: make(map[string]any),
    }

    // Read transcript content for intent matching using precise tool-use-ID extraction
    if transcriptPath != "" && toolUseID != "" {
        logger.Debug().
            Str("tool_name", toolName).
            Str("tool_use_id", toolUseID).
            Str("transcript_path", transcriptPath).
            Msg("PostToolUse hook triggered, extracting intent by tool_use_id")
            
        intent, err := transcript.ExtractIntentByToolUseID(transcriptPath, toolUseID)
        if err != nil {
            logger.Debug().Err(err).
                Str("tool_use_id", toolUseID).
                Str("transcript_path", transcriptPath).
                Msg("Failed to extract intent by tool use ID")
        } else if intent != "" {
            logger.Debug().
                Str("tool_use_id", toolUseID).
                Str("extracted_intent", intent).
                Msg("ExtractIntentContent extracted content from transcript using tool_use_id")
        }
        content.intent = intent
    } else {
        logger.Debug().
            Str("transcript_path", transcriptPath).
            Str("tool_use_id", toolUseID).
            Msg("Missing transcript_path or tool_use_id for precise intent extraction")
    }

    // Extract tool output fields from structured response
    if toolResponse != nil {
        switch v := toolResponse.(type) {
        case map[string]any:
            content.toolOutputMap = v
        case string:
            // Handle simple string responses for backward compatibility
            content.toolOutputMap["tool_response"] = v
        }
    }

    return content, nil
}
```

#### 5. Testing Strategy

**Test Cases to Add**:

1. **Test transcript structure with tool_use_id correlation**:
   ```go
   func TestExtractIntentByToolUseID_Success(t *testing.T) {
       // Create test transcript with proper structure:
       // 1. Assistant intent message (uuid: parent-uuid)
       // 2. Assistant tool_use message (parentUuid: parent-uuid, content.id: tool-use-id)
       // 3. Verify ExtractIntentByToolUseID returns correct intent
   }
   ```

2. **Test missing tool_use_id in transcript**:
   ```go
   func TestExtractIntentByToolUseID_NotFound(t *testing.T) {
       // Verify returns empty string when tool_use_id not found
   }
   ```

3. **Test malformed transcript entries**:
   ```go
   func TestExtractIntentByToolUseID_MalformedJSON(t *testing.T) {
       // Verify robust handling of invalid JSON lines
   }
   ```

4. **Update existing PostToolUse tests**:
   - Add `tool_use_id` field to test JSON payloads
   - Create test transcripts with proper correlation structure
   - Verify intent extraction works correctly

#### 6. Migration Plan

1. **Phase 1**: Implement new extraction function with tests
2. **Phase 2**: Update hook structures to expect tool_use_id 
3. **Phase 3**: Update PreToolUse and PostToolUse to use new extraction
4. **Phase 4**: Remove old `ExtractIntentContentOptimized` function
5. **Phase 5**: Update documentation

## Benefits

### Reliability
- **Precise Correlation**: Only extracts intent for the specific tool invocation
- **No Arbitrary Limits**: Works with any size intent content
- **Predictable**: Always returns the same intent for the same tool_use_id

### Maintainability
- **Simple Logic**: Clear algorithm with single responsibility
- **Robust Error Handling**: Gracefully handles missing or malformed data
- **Consistent Approach**: Same method for both PreToolUse and PostToolUse hooks

### Performance
- **Efficient Parsing**: Reads transcript once, stops when target found
- **Memory Efficient**: Doesn't load entire transcript into memory
- **Unlimited Line Length**: Uses `bufio.Reader.ReadString()` for unlimited line handling

## Assumptions and Risks

### Assumptions
1. **Claude Code Hook Payload**: Assumes `tool_use_id` will be available in hook payloads
2. **Transcript Structure**: Assumes current transcript structure remains stable
3. **Parent-Child Relationship**: Assumes tool_use messages have parentUuid pointing to intent

### Risks and Mitigations
1. **tool_use_id Not Available**: If Claude doesn't provide it, we can extract from transcript by finding most recent tool_use
2. **Transcript Format Changes**: Robust JSON parsing with error handling for forward compatibility
3. **Performance with Large Files**: Use streaming approach, stop on first match

### Fallback Strategy
If `tool_use_id` is not available in hook payloads:
- Parse transcript to find the most recent tool_use entry
- Use that ID for correlation
- This maintains the precision benefits while working with current Claude Code

## Implementation Timeline

- **Day 1**: Implement `ExtractIntentByToolUseID` function with tests ✅ **COMPLETED**
- **Day 2**: Update hook structures and PreToolUse processing ✅ **COMPLETED**
- **Day 3**: Update PostToolUse processing and integration tests ✅ **COMPLETED**
- **Day 4**: Remove old extraction methods and update documentation
- **Day 5**: End-to-end testing and refinement

## Implementation Status

### ✅ Phase 1: Core Implementation (COMPLETED)
1. **ExtractIntentByToolUseID Function**: Implemented in `/home/callan/dev/bumpers/internal/platform/claude/transcript/reader.go:295-374`
   - Added helper structures: `TranscriptEntry`, `MessageContent`, `ContentItem`
   - Added required struct fields: `UUID`, `ParentUUID`, `ID`
   - Comprehensive error handling and streaming file processing
   - Full unit test coverage with various scenarios

2. **HookEvent Structure Update**: Added `ToolUseID` field in `/home/callan/dev/bumpers/internal/core/engine/hooks/hooks.go:41`
   - Properly tagged with JSON snake_case annotation
   - Integrated with existing parsing logic

### ✅ Phase 2: PreToolUse Processing (COMPLETED)
3. **PreToolUse Integration**: Updated in `/home/callan/dev/bumpers/internal/cli/app.go:269-276`
   - Added conditional extraction logic using tool_use_id when available
   - Falls back to existing extraction method for backward compatibility
   - Comprehensive unit test demonstrating precise extraction vs old method

### ✅ Phase 3: PostToolUse Processing (COMPLETED)  
4. **PostToolUse Integration**: Updated in `/home/callan/dev/bumpers/internal/cli/app.go:1052-1059`
   - Same conditional logic pattern as PreToolUse
   - Maintains backward compatibility
   - Unit tests verify correct extraction behavior

### ✅ Phase 4: Test Integration (COMPLETED)
5. **Integration Tests**: Updated E2E tests in `/home/callan/dev/bumpers/cmd/bumpers/hook_e2e_test.go`
   - Added `tool_use_id` field to all hook input JSON payloads
   - Verified all tests pass with new field structure
   - Maintains backward compatibility for missing tool_use_id

### 📋 Remaining Tasks
6. **Legacy Method Cleanup**: Remove `ExtractIntentContentOptimized` once confirmed no longer needed
7. **Documentation Updates**: Update configuration and API documentation
8. **Production Validation**: Monitor real-world usage with Claude Code

## Key Benefits Achieved

### ✅ Reliability Improvements
- **Precise Correlation**: Tool use ID provides exact correlation between hook events and intent content
- **No Size Limitations**: Works with any intent content size, unlike the 50-line arbitrary limit
- **Predictable Results**: Same tool_use_id always returns same intent content

### ✅ Maintainability Enhancements  
- **Clean Architecture**: Single responsibility functions with clear error handling
- **Backward Compatible**: Graceful fallback to existing methods when tool_use_id unavailable
- **Comprehensive Testing**: Full unit and integration test coverage

### ✅ Performance Optimizations
- **Efficient Parsing**: Streaming JSON processing, stops on first match
- **Memory Efficient**: Processes large transcript files without loading entirely into memory
- **Unlimited Line Support**: Uses `bufio.Reader.ReadString()` for any line length

This plan provides a robust, maintainable solution that properly correlates hook events with their specific intent content in the transcript.