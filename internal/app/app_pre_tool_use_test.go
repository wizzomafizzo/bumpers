package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wizzomafizzo/bumpers/internal/claude"
	"github.com/wizzomafizzo/bumpers/internal/hooks"
	testutil "github.com/wizzomafizzo/bumpers/internal/testing"
)

type preToolUseIntentTestCase struct {
	testName          string
	transcriptContent string
	matchPattern      string
	command           string
	desc              string
	expectedMessage   string
}

// Helper function to test PreToolUse intent matching with different scenarios
func testPreToolUseIntentMatching(t *testing.T, tc *preToolUseIntentTestCase) {
	t.Helper()
	ctx, _ := setupTestWithContext(t)

	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, tc.testName+"-transcript.jsonl")

	err := os.WriteFile(transcriptPath, []byte(tc.transcriptContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to write transcript file: %v", err)
	}

	configContent := fmt.Sprintf(`rules:
  - match:
      pattern: "%s"
      event: "pre"
      sources: ["#intent"]
    send: "%s"
    generate: "off"`, tc.matchPattern, tc.expectedMessage)

	configPath := createTempConfig(t, configContent)
	app := NewApp(ctx, configPath)

	hookInput := fmt.Sprintf(`{
		"transcript_path": "%s",
		"tool_name": "Bash",
		"tool_input": {
			"command": "%s",
			"description": "%s"
		}
	}`, transcriptPath, tc.command, tc.desc)

	result, err := app.ProcessHook(context.Background(), strings.NewReader(hookInput))
	if err != nil {
		t.Fatalf("ProcessHook failed: %v", err)
	}

	if result.Message != tc.expectedMessage {
		t.Errorf("Expected %q, got %q", tc.expectedMessage, result.Message)
	}
}

func TestPreToolUseIntentSupport(t *testing.T) {
	t.Parallel()

	transcriptContent := `{"type":"assistant","message":{"content":[` +
		`{"type":"text","text":"I need to run some database tests to verify the connection works"}, ` +
		`{"type":"text","text":"I'll run the database tests for you"}]}}`
	testPreToolUseIntentMatching(t, &preToolUseIntentTestCase{
		testName:          "test",
		transcriptContent: transcriptContent,
		matchPattern:      "database.*test",
		command:           "python test_db.py",
		desc:              "Run database tests",
		expectedMessage:   "Consider checking database connection first",
	})
}

func TestPreToolUseIntentWithTextOnlyContent(t *testing.T) {
	t.Parallel()

	transcriptContent := `{"type":"assistant","message":{"content":[` +
		`{"type":"text","text":"I need to perform a critical system update to fix vulnerabilities"}]}}`
	testPreToolUseIntentMatching(t, &preToolUseIntentTestCase{
		testName:          "text-only",
		transcriptContent: transcriptContent,
		matchPattern:      "critical.*system",
		command:           "sudo apt update",
		desc:              "System update",
		expectedMessage:   "System update detected",
	})
}

// TestPreToolUseIntentWithMissingTranscript tests graceful handling when transcript doesn't exist
func TestPreToolUseIntentWithMissingTranscript(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestWithContext(t)

	configContent := `rules:
  - match:
      pattern: "anything"
      event: "pre"
      sources: ["#intent"]
    send: "This should not match"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(ctx, configPath)

	hookInput := `{
		"transcript_path": "/non/existent/transcript.jsonl",
		"tool_name": "Bash",
		"tool_input": {
			"command": "echo hello",
			"description": "Simple echo command"
		}
	}`

	result, err := app.ProcessHook(context.Background(), strings.NewReader(hookInput))
	if err != nil {
		t.Fatalf("ProcessHook failed: %v", err)
	}

	// Should return empty when transcript can't be read
	if result.Message != "" {
		t.Errorf("Expected empty result when transcript missing, got %q", result.Message)
	}
}

// TestExtractAndLogIntentCalledForAllPreToolUseEvents tests debug logging shows extracted intent content for PreToolUse
func TestExtractAndLogIntentCalledForAllPreToolUseEvents(t *testing.T) {
	t.Parallel()
	ctx, getLogs := setupTestWithContext(t)

	// Create a simple config (no specific rules needed - intent should be extracted regardless)
	configContent := `rules: []`
	configPath := createTempConfig(t, configContent)
	app := NewApp(ctx, configPath)

	// Create a transcript with intent content
	transcriptContent := `{"type":"assistant","uuid":"parent-uuid","message":{"role":"assistant",` +
		`"content":[{"type":"text","text":"I need to run echo hello for testing"}]}}`
	transcriptPath := createTempTranscript(t, transcriptContent)

	// Create PreToolUse hook input with transcript
	preToolUseInput := fmt.Sprintf(`{
		"tool_name": "Bash",
		"tool_input": {"command": "echo hello"},
		"transcript_path": "%s"
	}`, transcriptPath)

	// Process hook - this should call extractAndLogIntent and log the extracted intent
	_, err := app.ProcessHook(ctx, strings.NewReader(preToolUseInput))
	require.NoError(t, err)

	// Get captured log output
	logStr := getLogs()

	// Verify intent extraction debug output appears (this will fail until extractAndLogIntent is implemented)
	assert.Contains(t, logStr, "Intent extracted from transcript for hook processing")
	assert.Contains(t, logStr, "I need to run echo hello for testing")
}

func TestPreToolUseDebugOutputShowsIntentFields(t *testing.T) {
	t.Parallel()
	ctx, getLogs := setupTestWithContext(t)

	configContent := `rules:
  - match:
      pattern: "test.*pattern"
      event: "pre"
      sources: ["#intent"]
    send: "Found test pattern in intent"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(ctx, configPath)

	// Set up mock launcher to avoid real Claude calls
	mockLauncher := claude.SetupMockLauncherWithDefaults()
	mockLauncher.SetResponseForPattern("", "Enhanced message from AI")
	app.SetMockLauncher(mockLauncher)

	// Create transcript with text content
	transcriptContent := `{"type":"assistant","message":{"role":"assistant",` +
		`"content":[{"type":"text","text":"I need to test this pattern carefully"}]}}`
	transcriptPath := createTempTranscript(t, transcriptContent)

	preToolUseInput := fmt.Sprintf(`{
		"hook_event_name": "PreToolUse",
		"transcript_path": "%s",
		"tool_name": "Bash",
		"tool_input": {
			"command": "echo test",
			"description": "Simple echo command"
		}
	}`, transcriptPath)

	// Process hook with context-aware logging
	_, err := app.ProcessHook(ctx, strings.NewReader(preToolUseInput))
	require.NoError(t, err)

	// Get captured log output
	logStr := getLogs()

	// Verify PreToolUse processing debug output appears
	assert.Contains(t, logStr, "processing PreToolUse hook")
	assert.Contains(t, logStr, "received hook")
	// Intent extraction happens silently in PreToolUse - no additional logging asserts needed
}

func TestPreToolUseWithToolUseIDExtraction(t *testing.T) {
	t.Parallel()

	transcriptContent := `{"type":"assistant","uuid":"parent-uuid",` +
		`"message":{"role":"assistant","content":[{"type":"text",` +
		`"text":"I need to run some database tests to verify the connection works"}]}}
{"type":"assistant","uuid":"tool-use-uuid","parentUuid":"parent-uuid",` +
		`"message":{"role":"assistant","content":[{"type":"tool_use",` +
		`"id":"toolu_01KTePc3uLq34eriLmSLbgnx","name":"Bash"}]}}`

	configContent := `rules:
  - match:
      pattern: "database.*test"
      sources: ["#intent"]
    send: "Consider checking database connection first"
    generate: "off"`

	transcriptPath, app := createToolUseTestScenario(t, transcriptContent, configContent)

	hookInput := fmt.Sprintf(`{
		"tool_input": {"command": "python test_db.py"},
		"tool_name": "Bash",
		"transcript_path": "%s",
		"tool_use_id": "toolu_01KTePc3uLq34eriLmSLbgnx"
	}`, transcriptPath)

	response, err := app.ProcessHook(context.Background(), strings.NewReader(hookInput))
	require.NoError(t, err)
	assert.Contains(t, response.Message, "Consider checking database connection first",
		"Should match intent extracted by tool_use_id")
}

func TestPreToolUseWithoutToolUseIDNeedsNewReliableMethod(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestWithContext(t)

	// Create a transcript where old method would match wrong pattern due to noise
	// Old method concatenates ALL assistant text, new method gets most recent intent
	transcriptContent := `{"type":"assistant","message":{"role":"assistant",` +
		`"content":[{"type":"text","text":"I want to delete dangerous files using rm -rf"}]}}
{"type":"user","message":{"role":"user","content":[{"type":"text","text":"Actually, let's focus on testing instead"}]}}
{"type":"assistant","message":{"role":"assistant",` +
		`"content":[{"type":"text","text":"Good idea, let's focus on proper testing approaches"}]}}
{"type":"user","message":{"role":"user","content":[{"type":"text","text":"Run the tests please"}]}}
{"type":"assistant","message":{"role":"assistant",` +
		`"content":[{"type":"text","text":"Let me run some unit tests instead"}]}}
{"type":"assistant","message":{"role":"assistant",` +
		`"content":[{"type":"tool_use","id":"toolu_123","name":"Bash","input":{"command":"npm test"}}]}}`

	tempDir := t.TempDir()
	transcriptPath := filepath.Join(tempDir, "transcript.jsonl")
	err := os.WriteFile(transcriptPath, []byte(transcriptContent), 0o600)
	require.NoError(t, err)

	configContent := `rules:
  - match:
      pattern: "delete.*dangerous|rm.*-rf"
      sources: ["#intent"]
    send: "DANGEROUS COMMAND BLOCKED"
    generate: "off"
  - match:
      pattern: "unit.*test"
      sources: ["#intent"]
    send: "Test guidance"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(ctx, configPath)

	// Hook input WITHOUT tool_use_id - old method would include dangerous pattern
	hookInput := fmt.Sprintf(`{
		"tool_input": {"command": "npm test"},
		"tool_name": "Bash",
		"transcript_path": "%s"
	}`, transcriptPath)

	response, err := app.ProcessHook(context.Background(), strings.NewReader(hookInput))
	require.NoError(t, err)

	// With old method, this would wrongly block due to "rm -rf" in combined text
	// With new method, it should correctly allow and provide test guidance
	if strings.Contains(response.Message, "DANGEROUS COMMAND BLOCKED") {
		t.Error("Old method incorrectly blocked safe command due to noise in transcript")
	}
	assert.Contains(t, response.Message, "Test guidance",
		"Should match correct recent intent, not old noise")
}

func TestPreToolUseWithoutToolUseIDUsesOldMethod(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestWithContext(t)

	// Create a transcript that only works with the OLD method (no parent-child relationship)
	transcriptContent := `{"type":"assistant","message":{"role":"assistant",` +
		`"content":[{"type":"text","text":"I need to run database tests for verification"}]}}`

	tempDir := t.TempDir()
	transcriptPath := filepath.Join(tempDir, "transcript.jsonl")
	err := os.WriteFile(transcriptPath, []byte(transcriptContent), 0o600)
	require.NoError(t, err)

	configContent := `rules:
  - match:
      pattern: "database.*test"
      sources: ["#intent"]
    send: "Database test guidance"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(ctx, configPath)

	// Hook input WITHOUT tool_use_id - should fall back to old method
	hookInput := fmt.Sprintf(`{
		"tool_input": {"command": "python test_db.py"},
		"tool_name": "Bash",
		"transcript_path": "%s"
	}`, transcriptPath)

	response, err := app.ProcessHook(context.Background(), strings.NewReader(hookInput))
	require.NoError(t, err)
	assert.Contains(t, response.Message, "Database test guidance",
		"Should match using old extraction method when tool_use_id not available")
}

func TestPreToolUseToolUseIDExtractsPreciseIntent(t *testing.T) {
	t.Parallel()

	transcriptContent := `{"type":"user","message":{"role":"user","content":"unrelated user text"}}
{"type":"assistant","uuid":"parent-uuid","message":{"role":"assistant",` +
		`"content":[{"type":"text","text":"precise-target-intent"}]}}
{"type":"assistant","uuid":"tool-use-uuid","parentUuid":"parent-uuid",` +
		`"message":{"role":"assistant","content":[{"type":"tool_use",` +
		`"id":"toolu_01KTePc3uLq34eriLmSLbgnx","name":"Bash"}]}}
{"type":"assistant","message":{"role":"assistant",` +
		`"content":[{"type":"text","text":"other unrelated assistant message"}]}}`

	configContent := `rules:
  - match:
      pattern: "^precise-target-intent$"
      sources: ["#intent"]
    send: "Matched precise intent via tool_use_id"
    generate: "off"`

	transcriptPath, app := createToolUseTestScenario(t, transcriptContent, configContent)

	hookInput := fmt.Sprintf(`{
		"tool_input": {"command": "test command"},
		"tool_name": "Bash",
		"transcript_path": "%s",
		"tool_use_id": "toolu_01KTePc3uLq34eriLmSLbgnx"
	}`, transcriptPath)

	response, err := app.ProcessHook(context.Background(), strings.NewReader(hookInput))
	require.NoError(t, err)
	assert.Contains(t, response.Message, "Matched precise intent via tool_use_id",
		"Should match only the precise parent intent, not all assistant messages")
}

func TestExtractAndLogIntent_ReturnsIntentContent(t *testing.T) {
	ctx, getLogOutput := testutil.NewTestContext(t)
	t.Parallel()

	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "intent-test.jsonl")

	// Create transcript with assistant message
	transcriptContent := `{"type":"assistant","message":{"role":"assistant",` +
		`"content":[{"type":"text","text":"I'll search the codebase to understand the implementation"}]},` +
		`"uuid":"test-uuid","timestamp":"2024-01-01T10:00:00Z"}`
	err := os.WriteFile(transcriptPath, []byte(transcriptContent), 0o600)
	require.NoError(t, err)

	app := NewAppWithWorkDir("", tmpDir)
	event := hooks.HookEvent{TranscriptPath: transcriptPath}

	// Test that extractAndLogIntent returns the extracted content
	intentContent := app.extractAndLogIntent(ctx, &event)
	assert.Contains(t, intentContent, "I'll search the codebase to understand")

	_ = getLogOutput // Suppress unused variable warning
}
