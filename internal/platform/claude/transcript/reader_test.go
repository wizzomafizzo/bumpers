package transcript

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	testutil "github.com/wizzomafizzo/bumpers/internal/testing"
)

func TestExtractReasoningContent_WithRealisticTranscript(t *testing.T) {
	_, _ = testutil.NewTestContext(t) // Context-aware logging available
	t.Parallel()

	// Create a realistic Claude Code transcript with assistant reasoning
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "test-transcript.jsonl")

	// Realistic transcript content with assistant reasoning near the end
	transcriptContent := `{"type":"user","message":{"role":"user","content":"Run some tests"},` +
		`"uuid":"user1","timestamp":"2024-01-01T10:00:00Z"}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text",` +
		`"text":"I need to run the tests using the project's test command. ` +
		`Let me check what testing framework is being used and run the appropriate command."}]},` +
		`"uuid":"assistant1","timestamp":"2024-01-01T10:01:00Z"}
{"type":"user","message":{"role":"user","content":[{"tool_use_id":"tool1","type":"tool_result",` +
		`"content":[{"type":"text","text":"go test ./... \nFAIL: some tests failed"}]}]},` +
		`"uuid":"user2","timestamp":"2024-01-01T10:02:00Z"}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text",` +
		`"text":"I can see that the tests failed. Looking at the error output, ` +
		`it appears there are some issues with the test setup. ` +
		`I should examine the failing tests and fix them."}]},` +
		`"uuid":"assistant2","timestamp":"2024-01-01T10:03:00Z"}`

	err := os.WriteFile(transcriptPath, []byte(transcriptContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test transcript: %v", err)
	}

	// Extract reasoning content
	reasoning, err := ExtractReasoningContent(transcriptPath)
	if err != nil {
		t.Fatalf("ExtractReasoningContent failed: %v", err)
	}

	// Should find the assistant's reasoning
	if reasoning == "" {
		t.Error("Expected to find reasoning content, but got empty string")
	}

	// Should contain patterns indicating reasoning
	if !containsPattern(reasoning, "I need to") && !containsPattern(reasoning, "I can see") {
		t.Errorf("Expected reasoning content to contain reasoning patterns, got: %s", reasoning)
	}
}

func TestExtractReasoningContent_OnlyExtractsFromAssistantMessages(t *testing.T) {
	_, _ = testutil.NewTestContext(t) // Context-aware logging available
	t.Parallel()

	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "test-transcript.jsonl")

	// Content with reasoning in both user and assistant messages - should only extract from assistant
	transcriptContent := `{"type":"user","message":{"role":"user","content":"I need to test something"},"uuid":"user1"}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text",` +
		`"text":"I can see you want to test. Let me help with that."}]},"uuid":"assistant1"}`

	err := os.WriteFile(transcriptPath, []byte(transcriptContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test transcript: %v", err)
	}

	reasoning, err := ExtractReasoningContent(transcriptPath)
	if err != nil {
		t.Fatalf("ExtractReasoningContent failed: %v", err)
	}

	// Should only contain assistant reasoning, not user content
	if !containsPattern(reasoning, "I can see") {
		t.Errorf("Expected assistant reasoning 'I can see', got: %s", reasoning)
	}

	if containsPattern(reasoning, "I need to test something") {
		t.Errorf("Should not contain user content, but found: %s", reasoning)
	}
}

func TestExtractReasoningContent_LargeFileEfficiency(t *testing.T) {
	_, _ = testutil.NewTestContext(t) // Context-aware logging available
	t.Parallel()

	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "large-transcript.jsonl")

	// Create a large file with many old entries and reasoning at the end
	var content strings.Builder

	// Add 1000 old user messages without reasoning patterns we're looking for
	for i := 0; i < 1000; i++ {
		_, _ = content.WriteString(`{"type":"user","message":{"role":"user","content":"old message"},"uuid":"old`)
		_, _ = content.WriteString(string(rune('0' + i%10)))
		_, _ = content.WriteString(`"}`)
		_, _ = content.WriteString("\n")
	}

	// Add recent assistant message with reasoning at the end (most recent)
	_, _ = content.WriteString(`{"type":"assistant","message":{"role":"assistant","content":` +
		`[{"type":"text","text":"I can see this is a test. Let me analyze the situation."}]},` +
		`"uuid":"recent"}`)

	err := os.WriteFile(transcriptPath, []byte(content.String()), 0o600)
	if err != nil {
		t.Fatalf("Failed to create large test transcript: %v", err)
	}

	// Extract reasoning - should find the recent content efficiently
	reasoning, err := ExtractReasoningContent(transcriptPath)
	if err != nil {
		t.Fatalf("ExtractReasoningContent failed: %v", err)
	}

	// Should find the recent reasoning without reading the entire file
	if !containsPattern(reasoning, "I can see this is a test") {
		t.Errorf("Expected to find recent reasoning content, got: %s", reasoning)
	}
}

// TestExtractIntentContent_NonExistentFile tests graceful error handling for missing files
func TestExtractIntentContent_NonExistentFile(t *testing.T) {
	_, _ = testutil.NewTestContext(t) // Context-aware logging available
	t.Parallel()

	nonExistentPath := "/tmp/non-existent-transcript.jsonl"

	_, err := ExtractIntentContent(context.Background(), nonExistentPath)
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}

	// Should be a meaningful error about file not existing
	if !strings.Contains(err.Error(), "no such file") && !strings.Contains(err.Error(), "cannot find") {
		t.Errorf("Expected file not found error, got: %v", err)
	}
}

// TestExtractIntentContent_MalformedJSON tests handling of invalid JSON lines
func TestExtractIntentContent_MalformedJSON(t *testing.T) {
	_, _ = testutil.NewTestContext(t) // Context-aware logging available
	t.Parallel()

	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "malformed-transcript.jsonl")

	// Mix of valid and invalid JSON lines
	transcriptContent := `{"type":"user","message":{"role":"user","content":"Valid user message"},"uuid":"user1"}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Valid assistant response"}]}` +
		`,"uuid":"assistant1"}
{invalid json line without proper structure}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text",` +
		`"text":"This should still be extracted despite the bad line above."}]},"uuid":"assistant2"}
not even close to json
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Final valid response"}]}` +
		`,"uuid":"assistant3"}`

	err := os.WriteFile(transcriptPath, []byte(transcriptContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create malformed transcript: %v", err)
	}

	intent, err := ExtractIntentContent(context.Background(), transcriptPath)
	if err != nil {
		t.Fatalf("ExtractIntentContent should gracefully handle malformed JSON, got: %v", err)
	}

	// Should extract content from valid lines while skipping invalid ones
	if !containsPattern(intent, "Valid assistant response") {
		t.Errorf("Expected intent to contain valid assistant text, got: %s", intent)
	}
	if !containsPattern(intent, "This should still be extracted") {
		t.Errorf("Expected intent to contain text after malformed line, got: %s", intent)
	}
	if !containsPattern(intent, "Final valid response") {
		t.Errorf("Expected intent to contain final valid response, got: %s", intent)
	}

	// Should not contain user content
	if containsPattern(intent, "Valid user message") {
		t.Errorf("Intent should not contain user content, got: %s", intent)
	}
}

// TestExtractIntentContent_MixedContent tests complex realistic transcript with various content types
func TestExtractIntentContent_MixedContent(t *testing.T) {
	_, _ = testutil.NewTestContext(t) // Context-aware logging available
	t.Parallel()

	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "mixed-content-transcript.jsonl")

	// Realistic transcript with text content, tool calls, and longer message chains
	transcriptContent := `{"type":"user","message":{"role":"user",` +
		`"content":"I need to implement a new feature for user authentication"},` +
		`"uuid":"user1","timestamp":"2024-01-01T10:00:00Z"}
` + `{"type":"assistant","message":{"role":"assistant","content":[` +
		`{"type":"text","text":"I'll help you implement user authentication. Let me start by ` +
		`examining your current setup to recommend the best approach."}]},"uuid":"assistant1",` +
		`"timestamp":"2024-01-01T10:00:05Z"}
` + `{"type":"tool_call","tool":"Read","parameters":{"file_path":"src/auth/config.js"},` +
		`"uuid":"tool1","timestamp":"2024-01-01T10:00:10Z"}
` + `{"type":"tool_response","tool_response":{"content":"// Current auth config\nmodule.exports = ` +
		`{\n  secret: process.env.JWT_SECRET,\n  expiry: '24h'\n};"},"uuid":"tool_response1",` +
		`"timestamp":"2024-01-01T10:00:12Z"}
` + `{"type":"assistant","message":{"role":"assistant","content":[` +
		`{"type":"text","text":"Great! I can see you already have JWT configuration in place. ` +
		`Let's build on that foundation."},` +
		`{"type":"text","text":"I'll help you implement: 1) User registration with password ` +
		`hashing, 2) Login endpoint with JWT generation, 3) Protected route middleware."}]},` +
		`"uuid":"assistant2","timestamp":"2024-01-01T10:00:15Z"}
` + `{"type":"user","message":{"role":"user","content":"That sounds perfect. Can you start ` +
		`with the registration endpoint?"},"uuid":"user2","timestamp":"2024-01-01T10:00:20Z"}
` + `{"type":"assistant","message":{"role":"assistant","content":[` +
		`{"type":"text","text":"I'll create a registration endpoint with proper validation ` +
		`and security measures."}]},"uuid":"assistant3","timestamp":"2024-01-01T10:00:25Z"}`

	err := os.WriteFile(transcriptPath, []byte(transcriptContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create mixed content transcript: %v", err)
	}

	intent, err := ExtractIntentContent(context.Background(), transcriptPath)
	if err != nil {
		t.Fatalf("ExtractIntentContent failed: %v", err)
	}

	// Should extract text blocks only
	if !containsPattern(intent, "I'll help you implement user authentication") {
		t.Errorf("Expected intent to contain initial text response, got: %s", intent)
	}
	if !containsPattern(intent, "Great! I can see you already have JWT") {
		t.Errorf("Expected intent to contain recognition text, got: %s", intent)
	}
	if !containsPattern(intent, "I'll create a registration endpoint") {
		t.Errorf("Expected intent to contain implementation text, got: %s", intent)
	}

	// Should NOT contain user messages
	if containsPattern(intent, "I need to implement a new feature") {
		t.Errorf("Intent should not contain user content, got: %s", intent)
	}

	// Should NOT contain tool responses
	if containsPattern(intent, "Current auth config") {
		t.Errorf("Intent should not contain tool response content, got: %s", intent)
	}
}

func TestExtractIntentContentOptimized_ReadsRecentContentFirst(t *testing.T) {
	_, _ = testutil.NewTestContext(t) // Context-aware logging available
	t.Parallel()

	// Create a large transcript file to test performance optimization
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "large-transcript.jsonl")

	// Create transcript with older content and recent content
	var transcriptContent strings.Builder

	// Add many old entries (should be skipped with limit)
	for i := 0; i < 50; i++ {
		_, err := transcriptContent.WriteString(`{"type":"assistant","message":{"content":[` +
			`{"type":"text","text":"Old message ` + fmt.Sprintf("%d", i) + `"}]}}` + "\n")
		if err != nil {
			t.Fatalf("Failed to write to builder: %v", err)
		}
	}

	// Add recent relevant content that should be found
	_, err := transcriptContent.WriteString(`{"type":"assistant","message":{"content":[` +
		`{"type":"text","text":"I need to analyze the recent performance issue"},` +
		`{"type":"text","text":"Recent analysis shows performance problems"}]}}` + "\n")
	if err != nil {
		t.Fatalf("Failed to write to builder: %v", err)
	}
	_, err = transcriptContent.WriteString(`{"type":"assistant","message":{"content":[` +
		`{"type":"text","text":"Most recent message with important information"}]}}` + "\n")
	if err != nil {
		t.Fatalf("Failed to write to builder: %v", err)
	}

	if writeErr := os.WriteFile(transcriptPath, []byte(transcriptContent.String()), 0o600); writeErr != nil {
		t.Fatalf("Failed to write transcript file: %v", writeErr)
	}

	// Use optimized function with small line limit
	intent, err := ExtractIntentContentOptimized(context.Background(), transcriptPath, 5)
	if err != nil {
		t.Fatalf("ExtractIntentContentOptimized failed: %v", err)
	}

	// Should contain recent content
	if !strings.Contains(intent, "recent performance issue") {
		t.Errorf("Expected to find 'recent performance issue' in intent, got: %q", intent)
	}

	if !strings.Contains(intent, "Most recent message") {
		t.Errorf("Expected to find 'Most recent message' in intent, got: %q", intent)
	}

	// Should NOT contain very old messages (due to line limit)
	if strings.Contains(intent, "Old message 0") {
		t.Errorf("Should not contain old messages when using line limit, got: %q", intent)
	}
}

// BenchmarkExtractIntentContent benchmarks the regular intent extraction
func BenchmarkExtractIntentContent(b *testing.B) {
	// Create a large transcript file
	tmpDir := b.TempDir()
	transcriptPath := filepath.Join(tmpDir, "benchmark-transcript.jsonl")

	var transcriptContent strings.Builder
	for i := 0; i < 1000; i++ {
		_, err := transcriptContent.WriteString(`{"type":"assistant","message":{"content":[` +
			`{"type":"text","text":"Processing message ` + fmt.Sprintf("%d", i) + `"}]}}` + "\n")
		if err != nil {
			b.Fatalf("Failed to write to builder: %v", err)
		}
	}

	if err := os.WriteFile(transcriptPath, []byte(transcriptContent.String()), 0o600); err != nil {
		b.Fatalf("Failed to write transcript file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ExtractIntentContent(context.Background(), transcriptPath)
		if err != nil {
			b.Fatalf("ExtractIntentContent failed: %v", err)
		}
	}
}

// BenchmarkExtractIntentContentOptimized benchmarks the optimized intent extraction
func BenchmarkExtractIntentContentOptimized(b *testing.B) {
	// Create the same large transcript file
	tmpDir := b.TempDir()
	transcriptPath := filepath.Join(tmpDir, "benchmark-transcript.jsonl")

	var transcriptContent strings.Builder
	for i := 0; i < 1000; i++ {
		_, err := transcriptContent.WriteString(`{"type":"assistant","message":{"content":[` +
			`{"type":"text","text":"Processing message ` + fmt.Sprintf("%d", i) + `"}]}}` + "\n")
		if err != nil {
			b.Fatalf("Failed to write to builder: %v", err)
		}
	}

	if err := os.WriteFile(transcriptPath, []byte(transcriptContent.String()), 0o600); err != nil {
		b.Fatalf("Failed to write transcript file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ExtractIntentContentOptimized(context.Background(), transcriptPath, 50) // Only read recent 50 lines
		if err != nil {
			b.Fatalf("ExtractIntentContentOptimized failed: %v", err)
		}
	}
}

// Helper function for test
func containsPattern(text, pattern string) bool {
	return strings.Contains(text, pattern)
}

func TestExtractIntentByToolUseID_Success(t *testing.T) {
	_, _ = testutil.NewTestContext(t)
	t.Parallel()

	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "test-transcript-tool-use-id.jsonl")

	// Create test transcript with proper structure:
	// 1. Assistant intent message (uuid: parent-uuid)
	// 2. Assistant tool_use message (parentUuid: parent-uuid, content.id: tool-use-id)
	transcriptContent := `{"type":"user","message":{"role":"user","content":"Run some tests"},"uuid":"user1"}
{"type":"assistant","uuid":"parent-uuid","message":{"role":"assistant",` +
		`"content":[{"type":"text","text":"I'll help you run the tests. ` +
		`Let me check the current test configuration and run the appropriate test command."}]}}
{"type":"assistant","uuid":"tool-use-uuid","parentUuid":"parent-uuid",` +
		`"message":{"role":"assistant","content":[{"type":"tool_use",` +
		`"id":"toolu_01KTePc3uLq34eriLmSLbgnx","name":"Bash"}]}}
{"type":"system","toolUseID":"toolu_01KTePc3uLq34eriLmSLbgnx",` +
		`"content":"PreToolUse:Bash [bumpers hook] completed successfully"}`

	err := os.WriteFile(transcriptPath, []byte(transcriptContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test transcript: %v", err)
	}

	// Test extracting intent by tool use ID
	intent, err := ExtractIntentByToolUseID(context.Background(), transcriptPath, "toolu_01KTePc3uLq34eriLmSLbgnx")
	if err != nil {
		t.Fatalf("ExtractIntentByToolUseID failed: %v", err)
	}

	// Should return the parent message text content
	expected := "I'll help you run the tests. Let me check the current test " +
		"configuration and run the appropriate test command."
	if intent != expected {
		t.Errorf("Expected intent: %q, got: %q", expected, intent)
	}
}

func TestExtractIntentByToolUseID_LogsExtractedContent(t *testing.T) {
	ctx, getLogs := testutil.NewTestContext(t)
	t.Parallel()

	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "test-transcript-logging.jsonl")

	// Create test transcript
	transcriptContent := `{"type":"assistant","uuid":"parent-uuid","message":{"role":"assistant",` +
		`"content":[{"type":"text","text":"Testing logging behavior"}]}}
{"type":"assistant","uuid":"tool-use-uuid","parentUuid":"parent-uuid","message":{"role":"assistant",` +
		`"content":[{"type":"tool_use","id":"test-tool-id","name":"Bash"}]}}`

	err := os.WriteFile(transcriptPath, []byte(transcriptContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test transcript: %v", err)
	}

	// Extract intent using context-aware function - this should log the result
	intent, err := ExtractIntentByToolUseIDWithContext(ctx, transcriptPath, "test-tool-id")
	if err != nil {
		t.Fatalf("ExtractIntentByToolUseID failed: %v", err)
	}

	if intent != "Testing logging behavior" {
		t.Errorf("Expected 'Testing logging behavior', got: %q", intent)
	}

	// Check logs contain the extracted content
	logOutput := getLogs()
	if !strings.Contains(logOutput, "ExtractIntentByToolUseID extracted content from transcript") {
		t.Errorf("Expected log message not found in logs: %s", logOutput)
	}
	if !strings.Contains(logOutput, "Testing logging behavior") {
		t.Errorf("Expected extracted content not found in logs: %s", logOutput)
	}
}

func TestExtractIntentByToolUseID_NotFound(t *testing.T) {
	_, _ = testutil.NewTestContext(t)
	t.Parallel()

	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "test-transcript-no-match.jsonl")

	// Create transcript without the requested tool use ID
	transcriptContent := `{"type":"assistant","uuid":"parent-uuid",` +
		`"message":{"role":"assistant","content":[{"type":"text","text":"Some content"}]}}
{"type":"assistant","uuid":"tool-use-uuid","parentUuid":"parent-uuid",` +
		`"message":{"role":"assistant","content":[{"type":"tool_use","id":"different-id","name":"Bash"}]}}`

	err := os.WriteFile(transcriptPath, []byte(transcriptContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test transcript: %v", err)
	}

	// Should return empty string when tool_use_id not found
	intent, err := ExtractIntentByToolUseID(context.Background(), transcriptPath, "toolu_01KTePc3uLq34eriLmSLbgnx")
	if err != nil {
		t.Fatalf("ExtractIntentByToolUseID failed: %v", err)
	}

	if intent != "" {
		t.Errorf("Expected empty string when tool_use_id not found, got: %q", intent)
	}
}

func TestExtractIntentByToolUseID_MalformedJSON(t *testing.T) {
	_, _ = testutil.NewTestContext(t)
	t.Parallel()

	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "test-transcript-malformed.jsonl")

	// Mix valid and invalid JSON lines - should skip malformed lines
	transcriptContent := `{"type":"assistant","uuid":"parent-uuid",` +
		`"message":{"role":"assistant","content":[{"type":"text","text":"Valid intent message"}]}}
{invalid json line}
{"type":"assistant","uuid":"tool-use-uuid","parentUuid":"parent-uuid",` +
		`"message":{"role":"assistant","content":[{"type":"tool_use",` +
		`"id":"toolu_01KTePc3uLq34eriLmSLbgnx","name":"Bash"}]}}
not json at all`

	err := os.WriteFile(transcriptPath, []byte(transcriptContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test transcript: %v", err)
	}

	// Should handle malformed JSON gracefully and still find the valid intent
	intent, err := ExtractIntentByToolUseID(context.Background(), transcriptPath, "toolu_01KTePc3uLq34eriLmSLbgnx")
	if err != nil {
		t.Fatalf("ExtractIntentByToolUseID failed: %v", err)
	}

	expected := "Valid intent message"
	if intent != expected {
		t.Errorf("Expected intent: %q, got: %q", expected, intent)
	}
}

func TestExtractIntentContent_ImprovedErrorMessages(t *testing.T) {
	_, _ = testutil.NewTestContext(t) // Context-aware logging available
	t.Parallel()

	nonExistentPath := "/path/that/definitely/does/not/exist/transcript.jsonl"

	result, err := ExtractIntentContent(context.Background(), nonExistentPath)
	if err == nil {
		t.Fatal("Expected error for non-existent file, got nil")
	}
	if result != "" {
		t.Errorf("Expected empty result for failed file access, got: %s", result)
	}

	// Verify error message includes the file path
	if !strings.Contains(err.Error(), "failed to open transcript file") {
		t.Errorf("Expected descriptive error message, got: %v", err)
	}
	if !strings.Contains(err.Error(), nonExistentPath) {
		t.Errorf("Expected error message to include file path %s, got: %v", nonExistentPath, err)
	}
}

func TestExtractIntentContentOptimized_ImprovedErrorMessages(t *testing.T) {
	_, _ = testutil.NewTestContext(t) // Context-aware logging available
	t.Parallel()

	nonExistentPath := "/path/that/definitely/does/not/exist/transcript.jsonl"

	result, err := ExtractIntentContentOptimized(context.Background(), nonExistentPath, 50)
	if err == nil {
		t.Fatal("Expected error for non-existent file, got nil")
	}
	if result != "" {
		t.Errorf("Expected empty result for failed file access, got: %s", result)
	}

	// Verify error message includes the file path
	if !strings.Contains(err.Error(), "failed to open transcript file") {
		t.Errorf("Expected descriptive error message, got: %v", err)
	}
	if !strings.Contains(err.Error(), nonExistentPath) {
		t.Errorf("Expected error message to include file path %s, got: %v", nonExistentPath, err)
	}
}

func TestExtractIntentContent_HandlesLongLines(t *testing.T) {
	_, _ = testutil.NewTestContext(t)
	t.Parallel()

	tempDir := t.TempDir()
	transcriptPath := filepath.Join(tempDir, "long-line-transcript.jsonl")

	// Create a transcript with very long lines (over default buffer size)
	longText := strings.Repeat("This is a very long line of text that exceeds the "+
		"default bufio scanner buffer size. ", 1000)
	longEntry := fmt.Sprintf(`{"type":"assistant","message":{"content":[{"type":"text","text":%q}]}}`, longText)

	if err := os.WriteFile(transcriptPath, []byte(longEntry+"\n"), 0o600); err != nil {
		t.Fatalf("Failed to create test transcript: %v", err)
	}

	// This should not fail with "token too long" error
	_, err := ExtractIntentContent(context.Background(), transcriptPath)
	if err != nil && strings.Contains(err.Error(), "token too long") {
		t.Fatalf("ExtractIntentContent failed with token too long error: %v", err)
	}

	// Also test the optimized version
	_, err = ExtractIntentContentOptimized(context.Background(), transcriptPath, 50)
	if err != nil && strings.Contains(err.Error(), "token too long") {
		t.Fatalf("ExtractIntentContentOptimized failed with token too long error: %v", err)
	}
}

// TestFindRecentToolUseAndExtractIntent tests the new reliable intent extraction method
func TestFindRecentToolUseAndExtractIntent(t *testing.T) {
	_, _ = testutil.NewTestContext(t) // Context-aware logging available
	t.Parallel()

	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "test-transcript-recent-tool-use.jsonl")

	// Create realistic transcript with recent entries - system message includes toolUseID
	now := time.Now()
	recentTime := now.Format(time.RFC3339Nano)
	oldTime := now.Add(-2 * time.Minute).Format(time.RFC3339Nano) // Outside 1-minute window

	transcriptContent := fmt.Sprintf(
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text",`+
			`"text":"I'll help you run the tests."}]},"uuid":"intent1","timestamp":"%s"}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_123",`+
			`"name":"Bash","input":{"command":"just test"}}]},"uuid":"tool1","parentUuid":"intent1","timestamp":"%s"}
{"type":"system","message":{"toolUseID":"toolu_123","result":"Tests passed"},"timestamp":"%s"}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text",`+
			`"text":"Old intent from long ago"}]},"uuid":"old1","timestamp":"%s"}
`, recentTime, recentTime, recentTime, oldTime)

	err := os.WriteFile(transcriptPath, []byte(transcriptContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test transcript: %v", err)
	}

	// Test the new function
	intent, err := FindRecentToolUseAndExtractIntent(context.Background(), transcriptPath)
	if err != nil {
		t.Fatalf("FindRecentToolUseAndExtractIntent failed: %v", err)
	}

	// Should extract the intent from the tool_use parent message (correct behavior)
	if !containsPattern(intent, "I'll help you run the tests.") {
		t.Errorf("Expected intent to contain tool_use parent content, got: %s", intent)
	}

	// Should NOT contain the old intent that's not related to the tool_use
	if containsPattern(intent, "Old intent from long ago") {
		t.Errorf("Expected intent not to contain unrelated old content, got: %s", intent)
	}
}

// TestFindRecentToolUseAndExtractIntent_ShouldPrioritizeToolUseParentIntent tests the correct behavior
// where intent should be extracted from tool_use parent messages, not just the most recent text
func TestFindRecentToolUseAndExtractIntent_ShouldPrioritizeToolUseParentIntent(t *testing.T) {
	_, _ = testutil.NewTestContext(t)
	t.Parallel()

	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "test-transcript-tool-use-priority.jsonl")

	// Simulate the exact scenario from the bug report:
	// 1. Assistant text message with current intent: "Now let me update the Testing section:"
	// 2. Assistant tool_use message that references the intent message via parentUuid
	// 3. When PreToolUse hook fires, it should extract from step 1, not any older text
	transcriptContent := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text",` +
		`"text":"Previous intent that should be ignored"}]},"uuid":"old-intent"}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text",` +
		`"text":"Now let me update the Testing section:"}]},"uuid":"current-intent"}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_xyz",` +
		`"name":"Edit","input":{"file_path":"README.md"}}]},"uuid":"tool-use","parentUuid":"current-intent"}
{"type":"system","message":{"toolUseID":"toolu_xyz",` +
		`"content":"PreToolUse:Edit [bumpers hook] completed successfully"}}`

	err := os.WriteFile(transcriptPath, []byte(transcriptContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test transcript: %v", err)
	}

	// Test the function - it should extract the CURRENT intent, not the previous one
	intent, err := FindRecentToolUseAndExtractIntent(context.Background(), transcriptPath)
	if err != nil {
		t.Fatalf("FindRecentToolUseAndExtractIntent failed: %v", err)
	}

	// Should contain the current intent that's linked to the tool_use via parentUuid
	expected := "Now let me update the Testing section:"
	if !containsPattern(intent, expected) {
		t.Errorf("Expected intent to contain current tool_use parent intent %q, but got: %q", expected, intent)
	}

	// Should NOT contain the previous intent
	unwanted := "Previous intent that should be ignored"
	if containsPattern(intent, unwanted) {
		t.Errorf("Intent should not contain previous intent %q, but got: %q", unwanted, intent)
	}
}

// TestFindRecentToolUseAndExtractIntent_WithDifferentContent tests with different transcript content
func TestFindRecentToolUseAndExtractIntent_WithDifferentContent(t *testing.T) {
	_, _ = testutil.NewTestContext(t)
	t.Parallel()

	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "different-content.jsonl")

	// Create transcript with different content
	transcriptContent := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text",` +
		`"text":"Let me analyze your codebase first."}]},"uuid":"intent1"}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_456",` +
		`"name":"Read","input":{"file_path":"src/app.js"}}]},"uuid":"tool1","parentUuid":"intent1"}
{"type":"system","message":{"toolUseID":"toolu_456","result":"File read successfully"}}`

	err := os.WriteFile(transcriptPath, []byte(transcriptContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test transcript: %v", err)
	}

	// Test the new function
	intent, err := FindRecentToolUseAndExtractIntent(context.Background(), transcriptPath)
	if err != nil {
		t.Fatalf("FindRecentToolUseAndExtractIntent failed: %v", err)
	}

	// Should extract the correct intent
	if !containsPattern(intent, "Let me analyze your codebase first") {
		t.Errorf("Expected intent to contain 'analyze your codebase', got: %s", intent)
	}
}

func TestFindRecentToolUseAndExtractIntent_WithTextContent(t *testing.T) {
	_, _ = testutil.NewTestContext(t)
	t.Parallel()

	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "text-content.jsonl")

	// Create transcript with only text content (realistic for Claude Code)
	transcriptContent := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text",` +
		`"text":"Let me help you with that analysis."}]},"uuid":"recent"}`

	err := os.WriteFile(transcriptPath, []byte(transcriptContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test transcript: %v", err)
	}

	intent, err := FindRecentToolUseAndExtractIntent(context.Background(), transcriptPath)
	if err != nil {
		t.Fatalf("FindRecentToolUseAndExtractIntent failed: %v", err)
	}

	// Should extract text content
	if !containsPattern(intent, "Let me help you with that analysis") {
		t.Errorf("Expected intent to contain text content, got: %s", intent)
	}
}

func TestExtractIntentContent_WithThinkingBlocks(t *testing.T) {
	_, _ = testutil.NewTestContext(t) // Context-aware logging available
	t.Parallel()

	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "thinking-test.jsonl")

	// Real transcript data with thinking blocks from the actual conversation
	line1 := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"thinking",` +
		`"thinking":"The user is reporting that intent extraction isn't working properly. ` +
		`Let me investigate this issue.","signature":"test-signature"}]},` +
		`"uuid":"assistant1","timestamp":"2024-01-01T10:01:00Z"}`
	line2 := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text",` +
		`"text":"I'll investigate the intent extraction issue."}]},` +
		`"uuid":"assistant2","timestamp":"2024-01-01T10:02:00Z"}`
	transcriptContent := line1 + "\n" + line2

	err := os.WriteFile(transcriptPath, []byte(transcriptContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test transcript: %v", err)
	}

	intent, err := ExtractIntentContent(context.Background(), transcriptPath)
	if err != nil {
		t.Fatalf("ExtractIntentContent failed: %v", err)
	}

	// Should extract both thinking content and text content
	// This test will currently fail because extractContentByType doesn't handle thinking type
	if !containsPattern(intent, "intent extraction isn't working") {
		t.Errorf("Expected intent to contain thinking content 'intent extraction isn't working', got: %s", intent)
	}
	if !containsPattern(intent, "investigate the intent extraction") {
		t.Errorf("Expected intent to contain text content 'investigate the intent extraction', got: %s", intent)
	}
}
