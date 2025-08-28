package transcript

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wizzomafizzo/bumpers/internal/testing"
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
		t.Errorf("Expected reasoning content to contain thinking patterns, got: %s", reasoning)
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

func TestExtractIntentContent_WithThinkingBlocks(t *testing.T) {
	_, _ = testutil.NewTestContext(t) // Context-aware logging available
	t.Parallel()

	// Create transcript with thinking blocks (new format)
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "test-transcript-thinking.jsonl")

	// Realistic transcript content with thinking blocks and text content
	transcriptContent := `{"type":"user","message":{"role":"user","content":"Run some tests"},"uuid":"user1"}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"thinking",` +
		`"thinking":"I need to analyze what tests to run and how to execute them properly. ` +
		`Let me check the available test commands."},` +
		`{"type":"text","text":"I'll run the tests for you."}]},"uuid":"assistant1"}
{"type":"assistant","message":{"role":"assistant","content":[` +
		`{"type":"text","text":"The tests completed successfully."}]},"uuid":"assistant2"}`

	err := os.WriteFile(transcriptPath, []byte(transcriptContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test transcript: %v", err)
	}

	// Extract intent content
	intent, err := ExtractIntentContent(transcriptPath)
	if err != nil {
		t.Fatalf("ExtractIntentContent failed: %v", err)
	}

	// Should contain both thinking and text content
	if !containsPattern(intent, "I need to analyze what tests to run") {
		t.Errorf("Expected intent to contain thinking content, got: %s", intent)
	}
	if !containsPattern(intent, "I'll run the tests for you") {
		t.Errorf("Expected intent to contain text content, got: %s", intent)
	}
	if !containsPattern(intent, "The tests completed successfully") {
		t.Errorf("Expected intent to contain second text content, got: %s", intent)
	}
}

// TestExtractIntentContent_WithTextOnly tests extraction when only text blocks are present
func TestExtractIntentContent_WithTextOnly(t *testing.T) {
	_, _ = testutil.NewTestContext(t) // Context-aware logging available
	t.Parallel()

	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "text-only-transcript.jsonl")

	// Content with only text blocks, no thinking blocks
	transcriptContent := `{"type":"user","message":{"role":"user","content":"Show me the status"},"uuid":"user1",` +
		`"timestamp":"2024-01-01T10:00:00Z"}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text",` +
		`"text":"I'll check the current status for you."}]},"uuid":"assistant1","timestamp":"2024-01-01T10:00:05Z"}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text",` +
		`"text":"The system is running normally with no issues detected."}]},"uuid":"assistant2",` +
		`"timestamp":"2024-01-01T10:00:10Z"}`

	err := os.WriteFile(transcriptPath, []byte(transcriptContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create text-only transcript: %v", err)
	}

	intent, err := ExtractIntentContent(transcriptPath)
	if err != nil {
		t.Fatalf("ExtractIntentContent failed: %v", err)
	}

	// Should contain text content from both assistant messages
	if !containsPattern(intent, "I'll check the current status") {
		t.Errorf("Expected intent to contain first text content, got: %s", intent)
	}
	if !containsPattern(intent, "The system is running normally") {
		t.Errorf("Expected intent to contain second text content, got: %s", intent)
	}

	// Should not contain user content
	if containsPattern(intent, "Show me the status") {
		t.Errorf("Intent should not contain user content, got: %s", intent)
	}
}

// TestExtractIntentContent_EmptyFile tests handling of empty transcript files
func TestExtractIntentContent_EmptyFile(t *testing.T) {
	_, _ = testutil.NewTestContext(t) // Context-aware logging available
	t.Parallel()

	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "empty-transcript.jsonl")

	// Create empty file
	err := os.WriteFile(transcriptPath, []byte(""), 0o600)
	if err != nil {
		t.Fatalf("Failed to create empty transcript: %v", err)
	}

	intent, err := ExtractIntentContent(transcriptPath)
	if err != nil {
		t.Fatalf("ExtractIntentContent should handle empty file gracefully, got: %v", err)
	}

	if intent != "" {
		t.Errorf("Expected empty intent from empty file, got: %s", intent)
	}
}

// TestExtractIntentContent_NonExistentFile tests graceful error handling for missing files
func TestExtractIntentContent_NonExistentFile(t *testing.T) {
	_, _ = testutil.NewTestContext(t) // Context-aware logging available
	t.Parallel()

	nonExistentPath := "/tmp/non-existent-transcript.jsonl"

	_, err := ExtractIntentContent(nonExistentPath)
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
{"type":"assistant","message":{"role":"assistant","content":[{"type":"thinking",` +
		`"thinking":"This should still be extracted despite the bad line above."}]},"uuid":"assistant2"}
not even close to json
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Final valid response"}]}` +
		`,"uuid":"assistant3"}`

	err := os.WriteFile(transcriptPath, []byte(transcriptContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create malformed transcript: %v", err)
	}

	intent, err := ExtractIntentContent(transcriptPath)
	if err != nil {
		t.Fatalf("ExtractIntentContent should gracefully handle malformed JSON, got: %v", err)
	}

	// Should extract content from valid lines while skipping invalid ones
	if !containsPattern(intent, "Valid assistant response") {
		t.Errorf("Expected intent to contain valid assistant text, got: %s", intent)
	}
	if !containsPattern(intent, "This should still be extracted") {
		t.Errorf("Expected intent to contain thinking after malformed line, got: %s", intent)
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

	// Realistic transcript with mixed thinking and text, tool calls, longer reasoning chains
	transcriptContent := `{"type":"user","message":{"role":"user",` +
		`"content":"I need to implement a new feature for user authentication"},` +
		`"uuid":"user1","timestamp":"2024-01-01T10:00:00Z"}
` + `{"type":"assistant","message":{"role":"assistant","content":[{"type":"thinking",` +
		`"thinking":"The user wants to implement authentication. I should consider the current ` +
		`architecture, security best practices, and available authentication methods. Let me think ` +
		`about JWT tokens, session management, password hashing, and database schema changes that ` +
		`might be needed."},` +
		`{"type":"text","text":"I'll help you implement user authentication. Let me start by ` +
		`examining your current setup to recommend the best approach."}]},"uuid":"assistant1",` +
		`"timestamp":"2024-01-01T10:00:05Z"}
` + `{"type":"tool_call","tool":"Read","parameters":{"file_path":"src/auth/config.js"},` +
		`"uuid":"tool1","timestamp":"2024-01-01T10:00:10Z"}
` + `{"type":"tool_response","tool_response":{"content":"// Current auth config\nmodule.exports = ` +
		`{\n  secret: process.env.JWT_SECRET,\n  expiry: '24h'\n};"},"uuid":"tool_response1",` +
		`"timestamp":"2024-01-01T10:00:12Z"}
` + `{"type":"assistant","message":{"role":"assistant","content":[{"type":"thinking",` +
		`"thinking":"I can see they already have some JWT configuration set up. This is good - ` +
		`I can build on their existing setup. I should recommend implementing proper password ` +
		`hashing with bcrypt, setting up user registration and login endpoints, and ensuring ` +
		`proper middleware for route protection."},` +
		`{"type":"text","text":"Great! I can see you already have JWT configuration in place. ` +
		`Let's build on that foundation."},` +
		`{"type":"text","text":"I'll help you implement: 1) User registration with password ` +
		`hashing, 2) Login endpoint with JWT generation, 3) Protected route middleware."}]},` +
		`"uuid":"assistant2","timestamp":"2024-01-01T10:00:15Z"}
` + `{"type":"user","message":{"role":"user","content":"That sounds perfect. Can you start ` +
		`with the registration endpoint?"},"uuid":"user2","timestamp":"2024-01-01T10:00:20Z"}
` + `{"type":"assistant","message":{"role":"assistant","content":[{"type":"thinking",` +
		`"thinking":"For the registration endpoint, I need to create a route that accepts ` +
		`username/email and password, validates the input, hashes the password with bcrypt, ` +
		`stores the user in the database, and returns appropriate success/error responses. ` +
		`I should also check for existing users to prevent duplicates."},` +
		`{"type":"text","text":"I'll create a registration endpoint with proper validation ` +
		`and security measures."}]},"uuid":"assistant3","timestamp":"2024-01-01T10:00:25Z"}`

	err := os.WriteFile(transcriptPath, []byte(transcriptContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create mixed content transcript: %v", err)
	}

	intent, err := ExtractIntentContent(transcriptPath)
	if err != nil {
		t.Fatalf("ExtractIntentContent failed: %v", err)
	}

	// Should extract thinking blocks
	if !containsPattern(intent, "I should consider the current architecture") {
		t.Errorf("Expected intent to contain architectural thinking, got: %s", intent)
	}
	if !containsPattern(intent, "I can see they already have some JWT") {
		t.Errorf("Expected intent to contain analysis thinking, got: %s", intent)
	}
	if !containsPattern(intent, "For the registration endpoint") {
		t.Errorf("Expected intent to contain implementation thinking, got: %s", intent)
	}

	// Should extract text blocks
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
		`{"type":"thinking","thinking":"I need to analyze the recent performance issue"},` +
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
	intent, err := ExtractIntentContentOptimized(transcriptPath, 5)
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
			`{"type":"thinking","thinking":"Analysis step ` + fmt.Sprintf("%d", i) + `"},` +
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
		_, err := ExtractIntentContent(transcriptPath)
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
			`{"type":"thinking","thinking":"Analysis step ` + fmt.Sprintf("%d", i) + `"},` +
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
		_, err := ExtractIntentContentOptimized(transcriptPath, 50) // Only read recent 50 lines
		if err != nil {
			b.Fatalf("ExtractIntentContentOptimized failed: %v", err)
		}
	}
}

// Helper function for test
func containsPattern(text, pattern string) bool {
	return strings.Contains(text, pattern)
}

func TestExtractIntentContent_ImprovedErrorMessages(t *testing.T) {
	_, _ = testutil.NewTestContext(t) // Context-aware logging available
	t.Parallel()

	nonExistentPath := "/path/that/definitely/does/not/exist/transcript.jsonl"

	result, err := ExtractIntentContent(nonExistentPath)
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

	result, err := ExtractIntentContentOptimized(nonExistentPath, 50)
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
