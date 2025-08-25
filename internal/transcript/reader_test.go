package transcript

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wizzomafizzo/bumpers/internal/testutil"
)

func TestExtractReasoningContent_WithRealisticTranscript(t *testing.T) {
	testutil.InitTestLogger(t)
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
	testutil.InitTestLogger(t)
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
	testutil.InitTestLogger(t)
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

// Helper function for test
func containsPattern(text, pattern string) bool {
	return strings.Contains(text, pattern)
}
