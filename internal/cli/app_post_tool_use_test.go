package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wizzomafizzo/bumpers/internal/platform/claude"
)

// TestProcessPostToolUseWithContext tests that ProcessPostToolUse can accept context for race-safe logging
func TestProcessPostToolUseWithContext(t *testing.T) {
	t.Parallel()
	ctx, getLogs := setupTestWithContext(t)

	// Create a temporary config with post-tool rules
	configContent := `
rules:
  - match:
      pattern: "test content"
      event: "post"
    send: "Post tool use message"
`
	configFile := createTempConfig(t, configContent)
	app := NewApp(ctx, configFile)

	// Create PostToolUse event
	postToolJSON := `{
		"tool_name": "TestTool",
		"tool_response": "test content",
		"transcript_path": ""
	}`

	// Test that ProcessPostToolUse works with context - this will fail until we add context parameter
	result, err := app.ProcessPostToolUse(ctx, json.RawMessage(postToolJSON))
	require.NoError(t, err)
	assert.Contains(t, result, "Post tool use message")

	// Verify that logs were captured without race conditions
	logs := getLogs()
	assert.Contains(t, logs, "processing PostToolUse hook")
}

// TestProcessHookRoutesPostToolUse tests that PostToolUse hooks are properly routed
func TestProcessHookRoutesPostToolUse(t *testing.T) {
	t.Parallel()

	configContent := `rules:
  # Post-tool-use rule matching output
  - match:
      pattern: "error|failed"
      event: "post"
      sources: ["tool_response"]
    send: "Command failed - check the output"
  # Post-tool-use rule matching reasoning
  - match:
      pattern: "not related to my changes"
      event: "post"
      sources: ["#intent"]
    send: "AI claiming unrelated - please verify"`

	configPath := createTempConfig(t, configContent)
	workDir := t.TempDir()
	fs := afero.NewMemMapFs()
	app := NewAppWithFileSystem(configPath, workDir, fs)

	// Use static testdata transcript
	testdataDir := "../../testdata"
	transcriptPath := filepath.Join(testdataDir, "transcript-not-related.jsonl")
	absPath, err := filepath.Abs(transcriptPath)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}
	transcriptPath = absPath

	// Test PostToolUse hook with reasoning match
	postToolUseInput := `{
		"session_id": "abc123",
		"transcript_path": "` + transcriptPath + `",
		"cwd": "/test/directory",
		"hook_event_name": "PostToolUse",
		"tool_name": "Bash",
		"tool_input": {
			"command": "npm test"
		},
		"tool_response": {
			"success": true,
			"output": "All tests passed"
		}
	}`

	result, err := app.ProcessHook(context.Background(), strings.NewReader(postToolUseInput))
	if err != nil {
		t.Fatalf("ProcessHook failed for PostToolUse: %v", err)
	}

	// Should match the reasoning rule and return the message
	expectedMessage := "AI claiming unrelated - please verify"
	if result.Message != expectedMessage {
		t.Errorf("Expected %q, got %q", expectedMessage, result.Message)
	}
}

// TestPostToolUseWithDifferentTranscript tests that PostToolUse reads actual transcript content
func TestPostToolUseWithDifferentTranscript(t *testing.T) {
	t.Parallel()

	configContent := `rules:
  - match:
      pattern: "permission denied"
      event: "post"
      sources: ["#intent"]
    send: "File permission error detected"`

	configPath := createTempConfig(t, configContent)
	workDir := t.TempDir()
	fs := afero.NewMemMapFs()
	app := NewAppWithFileSystem(configPath, workDir, fs)

	// Use static testdata transcript with permission denied content
	testdataDir := "../../testdata"
	transcriptPath := filepath.Join(testdataDir, "transcript-permission-denied.jsonl")
	absPath, err := filepath.Abs(transcriptPath)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}
	transcriptPath = absPath

	postToolUseInput := `{
		"session_id": "test456",
		"transcript_path": "` + transcriptPath + `",
		"cwd": "/test/dir",
		"hook_event_name": "PostToolUse",
		"tool_name": "Bash",
		"tool_input": {"command": "ls /root"},
		"tool_response": {"error": true}
	}`

	result, err := app.ProcessHook(context.Background(), strings.NewReader(postToolUseInput))
	if err != nil {
		t.Fatalf("ProcessHook failed: %v", err)
	}

	// This should match "permission denied" pattern, not return the hardcoded stub message
	expectedMessage := "File permission error detected"
	if result.Message != expectedMessage {
		t.Errorf("Expected %q (from transcript matching), got %q", expectedMessage, result.Message)
	}

	// Verify it's NOT returning the hardcoded stub message
	stubMessage := "AI claiming unrelated - please verify"
	if result.Message == stubMessage {
		t.Error("Got hardcoded stub message - PostToolUse handler not reading transcript properly")
	}
}

// TestPostToolUseRuleNotMatching tests that PostToolUse returns empty when no rules match
func TestPostToolUseRuleNotMatching(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestWithContext(t)

	configContent := `rules:
  - match:
      pattern: "file not found"
      event: "post"
      sources: ["#intent"]
    send: "Check the file path"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(ctx, configPath)

	// Use transcript that won't match the "file not found" pattern
	testdataDir := "../../testdata"
	transcriptPath := filepath.Join(testdataDir, "transcript-no-match.jsonl")
	absPath, err := filepath.Abs(transcriptPath)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	postToolUseInput := `{
		"session_id": "test789",
		"transcript_path": "` + absPath + `",
		"cwd": "/test/dir",
		"hook_event_name": "PostToolUse",
		"tool_name": "Bash",
		"tool_input": {"command": "ls"},
		"tool_response": {"success": true}
	}`

	result, err := app.ProcessHook(context.Background(), strings.NewReader(postToolUseInput))
	if err != nil {
		t.Fatalf("ProcessHook failed: %v", err)
	}

	// Should return empty string when no rules match
	if result.Message != "" {
		t.Errorf("Expected empty result when no rules match, got %q", result.Message)
	}
}

// TestPostToolUseWithCustomPattern tests rule system integration with custom patterns
func TestPostToolUseWithCustomPattern(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestWithContext(t)

	// Create a config with a custom pattern that doesn't match hardcoded patterns
	configContent := `rules:
  - match:
      pattern: "timeout.*occurred"
      event: "post"
      sources: ["#intent"]
    send: "Operation timed out - check network connection"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(ctx, configPath)

	// Create a temporary transcript file with the custom pattern
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "custom-transcript.jsonl")
	transcriptContent := `{"type":"assistant","message":{"content":[{"type":"text",` +
		`"text":"The command failed because a timeout occurred while connecting to the server"}]}}`

	if err := os.WriteFile(transcriptPath, []byte(transcriptContent), 0o600); err != nil {
		t.Fatalf("Failed to write transcript file: %v", err)
	}

	postToolUseInput := fmt.Sprintf(`{
		"session_id": "test123",
		"transcript_path": "%s",
		"cwd": "/test/dir",
		"hook_event_name": "PostToolUse",
		"tool_name": "Bash",
		"tool_input": {"command": "curl -m 5 example.com"},
		"tool_response": {"success": false}
	}`, transcriptPath)

	result, err := app.ProcessHook(context.Background(), strings.NewReader(postToolUseInput))
	if err != nil {
		t.Fatalf("ProcessHook failed: %v", err)
	}

	// Should match the custom pattern and return the configured message
	expectedMessage := "Operation timed out - check network connection"
	if result.Message != expectedMessage {
		t.Errorf("Expected %q, got %q (rule system integration needed)",
			expectedMessage, result.Message)
	}
}

// TestPostToolUseWithToolOutputMatching tests tool_response field matching
func TestPostToolUseWithToolOutputMatching(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestWithContext(t)

	configContent := `rules:
  - match:
      pattern: "error.*exit code"
      event: "post"
      sources: ["tool_response"]
    send: "Tool execution failed"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(ctx, configPath)

	// Create mock JSON with tool_response containing error
	jsonData := `{
		"session_id": "test123",
		"transcript_path": "",
		"hook_event_name": "PostToolUse",
		"tool_name": "Bash",
		"tool_response": "Command failed with error: exit code 1"
	}`

	result, err := app.ProcessPostToolUse(context.Background(), json.RawMessage(jsonData))
	if err != nil {
		t.Fatalf("ProcessPostToolUse failed: %v", err)
	}

	assert.Equal(t, "Tool execution failed", result)
}

// TestPostToolUseWithMultipleFieldMatching tests multiple field matching in a single rule
func TestPostToolUseWithMultipleFieldMatching(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestWithContext(t)

	configContent := `rules:
  - match:
      pattern: "timeout|permission denied"
      event: "post"
      sources: ["#intent", "tool_response"]
    send: "Operation issue detected"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(ctx, configPath)

	// Generate unique session IDs to avoid cross-test interference
	sessionID1 := fmt.Sprintf("test-%d-1", time.Now().UnixNano())
	sessionID2 := fmt.Sprintf("test-%d-2", time.Now().UnixNano())

	// Test with tool_response containing timeout
	jsonData1 := fmt.Sprintf(`{
		"session_id": "%s",
		"transcript_path": "",
		"hook_event_name": "PostToolUse",
		"tool_name": "Bash",
		"tool_response": "Connection timeout occurred"
	}`, sessionID1)

	result1, err := app.ProcessPostToolUse(context.Background(), json.RawMessage(jsonData1))
	if err != nil {
		t.Fatalf("ProcessPostToolUse failed: %v", err)
	}

	assert.Equal(t, "Operation issue detected", result1)

	// Test with reasoning content containing permission denied
	// Create our own temporary transcript file to avoid race conditions with shared files
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript-permission-denied.jsonl")

	// Content that matches the pattern "permission denied" in intent extraction
	transcriptContent := `{"type":"user","message":{"role":"user","content":"List the files in /root directory"},` +
		`"uuid":"user1","timestamp":"2024-01-01T10:00:00Z"}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text",` +
		`"text":"I'll try to list the files in the /root directory for you."}]},` +
		`"uuid":"assistant1","timestamp":"2024-01-01T10:01:00Z"}
{"type":"user","message":{"role":"user","content":[{"tool_use_id":"tool1","type":"tool_result",` +
		`"content":[{"type":"text","text":"ls: cannot access '/root': Permission denied"}]}]},` +
		`"uuid":"user2","timestamp":"2024-01-01T10:02:00Z"}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text",` +
		`"text":"I can see there's a permission denied error when trying to access the /root directory. ` +
		`This suggests the command was run without sufficient privileges."}]},` +
		`"uuid":"assistant2","timestamp":"2024-01-01T10:03:00Z"}`

	if writeErr := os.WriteFile(transcriptPath, []byte(transcriptContent), 0o600); writeErr != nil {
		t.Fatalf("Failed to write transcript file: %v", writeErr)
	}

	jsonData2 := fmt.Sprintf(`{
		"session_id": "%s",
		"transcript_path": "%s",
		"hook_event_name": "PostToolUse",
		"tool_name": "Bash",
		"tool_response": "Command executed successfully"
	}`, sessionID2, transcriptPath)

	result2, err := app.ProcessPostToolUse(context.Background(), json.RawMessage(jsonData2))
	if err != nil {
		t.Fatalf("ProcessPostToolUse failed: %v", err)
	}

	assert.Equal(t, "Operation issue detected", result2)
}

// TestPostToolUseWithThinkingAndTextBlocks tests extraction of both thinking blocks and text content
func TestPostToolUseWithThinkingAndTextBlocks(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestWithContext(t)

	configContent := `rules:
  - match:
      pattern: "need to analyze.*performance"
      event: "post"
      sources: ["#intent"]
    send: "Performance analysis detected"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(ctx, configPath)

	// Create transcript with text content and proper tool_use structure
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "thinking-text-transcript.jsonl")
	transcriptContent := `{"type":"user","message":{"role":"user","content":"Check system performance"},` +
		`"uuid":"user1","timestamp":"2024-01-01T10:00:00Z"}
{"type":"assistant","message":{"role":"assistant","content":[` +
		`{"type":"text","text":"I need to analyze the system performance metrics to ` +
		`identify bottlenecks. I'll check the system performance for you."}]},"uuid":"assistant1",` +
		`"timestamp":"2024-01-01T10:01:00Z"}
{"type":"assistant","message":{"role":"assistant","content":[` +
		`{"type":"tool_use","id":"tool1","name":"Bash","input":{"command":"top"}}]},"uuid":"assistant2",` +
		`"parentUuid":"assistant1","timestamp":"2024-01-01T10:01:30Z"}
{"type":"user","message":{"role":"user","content":[{"tool_use_id":"tool1","type":"tool_result",` +
		`"content":[{"type":"text","text":"CPU: 85%, Memory: 70%, Disk: 40%"}]}]},"uuid":"user2",` +
		`"timestamp":"2024-01-01T10:02:00Z"}
{"type":"assistant","message":{"role":"assistant","content":[` +
		`{"type":"text","text":"The CPU usage is quite high at 85%, which could indicate performance issues. ` +
		`The system shows high CPU usage at 85%. This could be causing performance issues."}}],"uuid":"assistant3",` +
		`"timestamp":"2024-01-01T10:03:00Z"}`

	if err := os.WriteFile(transcriptPath, []byte(transcriptContent), 0o600); err != nil {
		t.Fatalf("Failed to write transcript file: %v", err)
	}

	postToolUseInput := fmt.Sprintf(`{
		"session_id": "test-thinking",
		"transcript_path": "%s",
		"cwd": "/test/dir",
		"hook_event_name": "PostToolUse",
		"tool_name": "Bash",
		"tool_input": {"command": "top"},
		"tool_response": {"success": true}
	}`, transcriptPath)

	result, err := app.ProcessHook(context.Background(), strings.NewReader(postToolUseInput))
	if err != nil {
		t.Fatalf("ProcessHook failed: %v", err)
	}

	// Should match pattern from text content
	expectedMessage := "Performance analysis detected"
	if result.Message != expectedMessage {
		t.Errorf("Expected %q (text content should be extracted), got %q", expectedMessage, result.Message)
	}
}

type postToolUseIntegrationTestCase struct {
	sessionID       string
	transcriptFile  string
	matchPattern    string
	command         string
	expectedMessage string
	success         bool
}

// Helper function for PostToolUse integration tests
func testPostToolUseIntegration(t *testing.T, tc *postToolUseIntegrationTestCase) {
	t.Helper()
	ctx, _ := setupTestWithContext(t)

	testdataDir := "../../testdata"
	transcriptPath := filepath.Join(testdataDir, tc.transcriptFile)
	absPath, err := filepath.Abs(transcriptPath)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	configContent := fmt.Sprintf(`rules:
  - match:
      pattern: "%s"
      event: "post"
      sources: ["#intent"]
    send: "%s"
    generate: "off"`, tc.matchPattern, tc.expectedMessage)

	configPath := createTempConfig(t, configContent)
	app := NewApp(ctx, configPath)

	postToolUseInput := fmt.Sprintf(`{
		"session_id": "%s",
		"transcript_path": "%s",
		"cwd": "/test/dir",
		"hook_event_name": "PostToolUse",
		"tool_name": "Bash",
		"tool_input": {"command": "%s"},
		"tool_response": {"success": %t}
	}`, tc.sessionID, absPath, tc.command, tc.success)

	result, err := app.ProcessHook(context.Background(), strings.NewReader(postToolUseInput))
	if err != nil {
		t.Fatalf("ProcessHook failed: %v", err)
	}

	if result.Message != tc.expectedMessage {
		t.Errorf("Expected %q, got %q", tc.expectedMessage, result.Message)
	}
}

func TestIntegrationThinkingAndTextExtraction(t *testing.T) {
	t.Parallel()

	testPostToolUseIntegration(t, &postToolUseIntegrationTestCase{
		sessionID:       "integration-test",
		transcriptFile:  "transcript-thinking-and-text.jsonl",
		matchPattern:    "transcript.*data",
		command:         "go test ./...",
		expectedMessage: "Transcript analysis detected",
		success:         false,
	})
}

func TestIntegrationPerformanceAnalysisDetection(t *testing.T) {
	t.Parallel()

	testPostToolUseIntegration(t, &postToolUseIntegrationTestCase{
		sessionID:       "perf-test",
		transcriptFile:  "transcript-performance-analysis.jsonl",
		matchPattern:    "immediate.*optimization",
		command:         "top",
		expectedMessage: "Performance optimization guidance triggered",
		success:         true,
	})
}

// TestPostToolUseStructuredToolResponseFields tests matching against specific fields in structured tool responses
func TestPostToolUseStructuredToolResponseFields(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestWithContext(t)

	// Test that rules can match against specific tool output fields by name
	configContent := `rules:
  - match:
      pattern: "JWT_SECRET"
      event: "post"
      sources: ["content"]  # Should match the "content" field in tool_response
    send: "Configuration file contains sensitive data"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(ctx, configPath)

	// Test matching against "content" field
	jsonWithContent := `{
		"session_id": "test123",
		"transcript_path": "",
		"hook_event_name": "PostToolUse", 
		"tool_name": "Read",
		"tool_response": {
			"content": "module.exports = { secret: process.env.JWT_SECRET }",
			"lines": 42
		}
	}`

	result1, err := app.ProcessPostToolUse(context.Background(), json.RawMessage(jsonWithContent))
	require.NoError(t, err)
	assert.Equal(t, "Configuration file contains sensitive data", result1)
}

// TestPostToolUseDebugOutputShowsIntentFields tests that debug logging shows extracted intent content
func TestPostToolUseDebugOutputShowsIntentFields(t *testing.T) {
	t.Parallel()
	ctx, getLogs := setupTestWithContext(t)

	configContent := `rules:
  - match:
      pattern: "test.*pattern"
      event: "post"
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

	postToolUseInput := fmt.Sprintf(`{
		"session_id": "debug-test",
		"transcript_path": "%s",
		"cwd": "/test/dir",
		"hook_event_name": "PostToolUse",
		"tool_name": "Bash",
		"tool_input": {"command": "echo test"},
		"tool_response": {"output": "Command output"}
	}`, transcriptPath)

	// Process hook with context-aware logging
	_, err := app.ProcessHook(ctx, strings.NewReader(postToolUseInput))
	require.NoError(t, err)

	// Get captured log output
	logStr := getLogs()

	// Verify intent extraction debug output appears
	assert.Contains(t, logStr, "FindRecentToolUseAndExtractIntent extracted content from transcript")
}

// TestPostToolUseWithToolUseIDExtraction tests tool_use_id based intent extraction for PostToolUse
func TestPostToolUseWithToolUseIDExtraction(t *testing.T) {
	t.Parallel()

	transcriptContent := `{"type":"assistant","message":{"role":"assistant",` +
		`"content":[{"type":"text","text":"general background intent"}]}}
{"type":"assistant","uuid":"parent-uuid","message":{"role":"assistant",` +
		`"content":[{"type":"text","text":"specific error handling needed for this operation"}]}}
{"type":"assistant","uuid":"tool-use-uuid","parentUuid":"parent-uuid",` +
		`"message":{"role":"assistant","content":[{"type":"tool_use",` +
		`"id":"toolu_01KTePc3uLq34eriLmSLbgnx","name":"Bash"}]}}`

	configContent := `rules:
  - match:
      pattern: "^specific error handling needed"
      sources: ["#intent"]
      event: "post"
    send: "Matched specific post-tool intent via tool_use_id"
    generate: "off"`

	transcriptPath, app := createToolUseTestScenario(t, transcriptContent, configContent)

	hookInput := fmt.Sprintf(`{
		"tool_name": "Bash",
		"transcript_path": "%s",
		"tool_use_id": "toolu_01KTePc3uLq34eriLmSLbgnx",
		"tool_response": "command output"
	}`, transcriptPath)

	response, err := app.ProcessPostToolUse(context.Background(), json.RawMessage(hookInput))
	require.NoError(t, err)
	assert.Contains(t, response, "Matched specific post-tool intent via tool_use_id",
		"Should match specific intent from tool_use_id extraction in PostToolUse")
}

// createTempTranscript creates a temporary transcript file for testing
func createTempTranscript(t *testing.T, content string) string {
	t.Helper()
	tempDir := t.TempDir()
	transcriptPath := filepath.Join(tempDir, "test-transcript.jsonl")
	err := os.WriteFile(transcriptPath, []byte(content), 0o600)
	if err != nil {
		t.Fatalf("Failed to create temp transcript: %v", err)
	}
	return transcriptPath
}

// Test helper for creating tool use transcript test scenarios
func createToolUseTestScenario(t *testing.T, transcriptContent, configContent string) (string, *App) {
	ctx, _ := setupTestWithContext(t)
	tempDir := t.TempDir()
	transcriptPath := filepath.Join(tempDir, "transcript.jsonl")
	err := os.WriteFile(transcriptPath, []byte(transcriptContent), 0o600)
	require.NoError(t, err)

	configPath := createTempConfig(t, configContent)
	app := NewApp(ctx, configPath)

	return transcriptPath, app
}
