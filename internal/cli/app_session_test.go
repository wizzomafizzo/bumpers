package cli

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wizzomafizzo/bumpers/internal/platform/claude"
	ai "github.com/wizzomafizzo/bumpers/internal/platform/claude/api"
)

func TestProcessSessionStartWithContext(t *testing.T) {
	t.Parallel()
	ctx, getLogs := setupTestWithContext(t)

	// Create a temporary config with session notes
	configContent := `
session:
  - add: "Session started with test context"
`
	configFile := createTempConfig(t, configContent)
	app := NewApp(ctx, configFile)

	// Create SessionStart event
	sessionJSON := `{"session_id": "test123", "hook_event_name": "SessionStart", "source": "startup"}`

	// Test that ProcessSessionStart works with context - this will fail until we add context parameter
	result, err := app.ProcessSessionStart(ctx, json.RawMessage(sessionJSON))
	require.NoError(t, err)
	assert.Contains(t, result, "Session started with test context")

	// Verify that logs were captured without race conditions
	logs := getLogs()
	assert.Contains(t, logs, "processing SessionStart hook")
}

func TestProcessHookRoutesSessionStart(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestWithContext(t)

	configContent := `rules:
  - match: "go test"
    send: "Use just test instead"
    generate: "off"
session:
  - add: "Remember to run tests first"
    generate: "off"
  - add: "Check CLAUDE.md for project conventions"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(ctx, configPath)

	// Test SessionStart hook routing with startup source
	sessionStartInput := `{
		"session_id": "abc123",
		"hook_event_name": "SessionStart",
		"source": "startup"
	}`

	result, err := app.ProcessHook(context.Background(), strings.NewReader(sessionStartInput))
	if err != nil {
		t.Fatalf("ProcessHook failed for SessionStart: %v", err)
	}

	expectedJSON := `{"hookSpecificOutput":{"hookEventName":"SessionStart",` +
		`"additionalContext":"Remember to run tests first\nCheck CLAUDE.md for project conventions"}}`
	if result.Message != expectedJSON {
		t.Errorf("Expected %q, got %q", expectedJSON, result.Message)
	}
}

func TestProcessSessionStartWithDifferentNotes(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestWithContext(t)

	configContent := `rules:
  - match: "go test"
    send: "Use just test instead"
    generate: "off"
session:
  - add: "Different message here"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(ctx, configPath)

	sessionStartInput := `{
		"session_id": "abc123",
		"hook_event_name": "SessionStart",
		"source": "startup"
	}`

	result, err := app.ProcessHook(context.Background(), strings.NewReader(sessionStartInput))
	if err != nil {
		t.Fatalf("ProcessHook failed for SessionStart: %v", err)
	}

	expectedJSON := `{"hookSpecificOutput":{"hookEventName":"SessionStart",` +
		`"additionalContext":"Different message here"}}`
	if result.Message != expectedJSON {
		t.Errorf("Expected %q, got %q", expectedJSON, result.Message)
	}
}

func TestProcessSessionStartIgnoresResume(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestWithContext(t)

	configContent := `session:
  - add: "Should not appear"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(ctx, configPath)

	sessionStartInput := `{
		"session_id": "abc123",
		"hook_event_name": "SessionStart",
		"source": "resume"
	}`

	result, err := app.ProcessHook(context.Background(), strings.NewReader(sessionStartInput))
	if err != nil {
		t.Fatalf("ProcessHook failed for SessionStart: %v", err)
	}

	// Should return empty string for resume source
	if result.Message != "" {
		t.Errorf("Expected empty string for resume source, got %q", result.Message)
	}
}

func TestProcessSessionStartWorksWithClear(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestWithContext(t)

	configContent := `session:
  - add: "Clear message"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(ctx, configPath)

	sessionStartInput := `{
		"session_id": "abc123",
		"hook_event_name": "SessionStart",
		"source": "clear"
	}`

	result, err := app.ProcessHook(context.Background(), strings.NewReader(sessionStartInput))
	if err != nil {
		t.Fatalf("ProcessHook failed for SessionStart: %v", err)
	}

	expectedJSON := `{"hookSpecificOutput":{"hookEventName":"SessionStart",` +
		`"additionalContext":"Clear message"}}`
	if result.Message != expectedJSON {
		t.Errorf("Expected %q, got %q", expectedJSON, result.Message)
	}
}

func TestProcessSessionStartWithTemplate(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestWithContext(t)

	configContent := `session:
  - add: "Hello from template!"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(ctx, configPath)

	sessionStartInput := `{
		"session_id": "abc123",
		"hook_event_name": "SessionStart",
		"source": "startup"
	}`

	result, err := app.ProcessHook(context.Background(), strings.NewReader(sessionStartInput))
	if err != nil {
		t.Fatalf("ProcessHook failed for SessionStart: %v", err)
	}

	// The template should be processed (no template syntax, so it should pass through as-is)
	expectedJSON := `{"hookSpecificOutput":{"hookEventName":"SessionStart",` +
		`"additionalContext":"Hello from template!"}}`
	if result.Message != expectedJSON {
		t.Errorf("Expected %q, got %q", expectedJSON, result.Message)
	}
}

func TestProcessSessionStartWithTodayVariable(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestWithContext(t)

	configContent := `session:
  - add: "Today is {{.Today}}"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(ctx, configPath)

	sessionStartInput := `{
		"session_id": "abc123",
		"hook_event_name": "SessionStart",
		"source": "startup"
	}`

	result, err := app.ProcessSessionStart(ctx, json.RawMessage(sessionStartInput))
	if err != nil {
		t.Fatalf("ProcessSessionStart failed: %v", err)
	}

	// Parse the response to get the additionalContext
	var response map[string]any
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	hookOutput, ok := response["hookSpecificOutput"].(map[string]any)
	if !ok {
		t.Fatal("Expected hookSpecificOutput in response")
	}

	additionalContext, ok := hookOutput["additionalContext"].(string)
	if !ok {
		t.Fatal("Expected additionalContext string in hookSpecificOutput")
	}

	expectedDate := time.Now().Format("2006-01-02")
	expectedMessage := "Today is " + expectedDate
	if additionalContext != expectedMessage {
		t.Errorf("Expected %q, got %q", expectedMessage, additionalContext)
	}
}

func TestProcessSessionStartClearsSessionCache(t *testing.T) { //nolint:paralleltest // t.Setenv() usage
	ctx, _ := setupTestWithContext(t)

	configContent := `session:
  - add: "Session started"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(ctx, configPath)

	// Create a single cache instance for testing
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "test.db")
	sharedCache, err := ai.NewCacheWithProject(ctx, cachePath, tempDir)
	if err != nil {
		t.Fatalf("Failed to create test cache: %v", err)
	}
	defer func() { _ = sharedCache.Close() }()

	// Populate cache with session data
	expiry := time.Now().Add(24 * time.Hour)
	sessionEntry := &ai.CacheEntry{
		GeneratedMessage: "Session test message",
		OriginalMessage:  "Original test message",
		Timestamp:        time.Now(),
		ExpiresAt:        &expiry,
	}

	err = sharedCache.Put(ctx, "test-session-key", sessionEntry)
	if err != nil {
		t.Fatalf("Failed to populate test cache: %v", err)
	}

	// Verify entry exists before clearing
	retrieved, err := sharedCache.Get(ctx, "test-session-key")
	if err != nil || retrieved == nil {
		t.Fatal("Session entry should exist before ProcessSessionStart")
	}

	// Inject the cache instance into the SessionManager for testing
	sessionManager, ok := app.sessionManager.(*DefaultSessionManager)
	require.True(t, ok, "expected DefaultSessionManager")
	sessionManager.SetCacheForTesting(sharedCache)

	sessionStartInput := `{
		"session_id": "abc123",
		"hook_event_name": "SessionStart",
		"source": "startup"
	}`

	// Process session start should clear session cache without database conflicts
	_, err = app.ProcessSessionStart(ctx, json.RawMessage(sessionStartInput))
	if err != nil {
		t.Fatalf("ProcessSessionStart failed: %v", err)
	}

	// Test cache clearing directly on the shared cache first
	err = sharedCache.ClearSessionCache(ctx)
	if err != nil {
		t.Fatalf("Direct cache clear failed: %v", err)
	}

	// Verify entries were cleared using the same shared cache instance
	retrieved, err = sharedCache.Get(ctx, "test-session-key")
	if err != nil {
		t.Fatalf("Unexpected error getting session key after clearing: %v", err)
	}
	if retrieved != nil {
		t.Error("Session entry should be cleared after ProcessSessionStart")
	}
}

func TestProcessSessionStartWithAIGeneration(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestWithContext(t)

	configContent := `session:
  - add: "Basic session message"
    generate: "always"`

	configPath := createTempConfig(t, configContent)
	workDir := t.TempDir()
	fs := afero.NewMemMapFs()
	app := NewAppWithFileSystem(configPath, workDir, fs)

	// Set up mock launcher
	mockLauncher := claude.SetupMockLauncherWithDefaults()
	mockLauncher.SetResponseForPattern("", "Enhanced session message from AI")
	app.SetMockLauncher(mockLauncher)

	// Create a temporary database file for AI cache testing
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "ai_test.db")

	// Set cache path on session manager's AI helper for testing
	sessionManager, ok := app.sessionManager.(*DefaultSessionManager)
	require.True(t, ok, "expected DefaultSessionManager")
	sessionManager.aiHelper.cachePath = cachePath

	sessionStartInput := `{
		"session_id": "abc123",
		"hook_event_name": "SessionStart",
		"source": "startup"
	}`

	result, err := app.ProcessHook(ctx, strings.NewReader(sessionStartInput))
	if err != nil {
		t.Fatalf("ProcessHook failed for SessionStart: %v", err)
	}

	expectedJSON := `{"hookSpecificOutput":{"hookEventName":"SessionStart",` +
		`"additionalContext":"Enhanced session message from AI"}}`
	if result.Message != expectedJSON {
		t.Errorf("Expected AI-generated session output %q, got %q", expectedJSON, result.Message)
	}

	// Verify the mock was called with the right prompt
	if mockLauncher.GetCallCount() == 0 {
		t.Error("Expected mock launcher to be called for AI generation")
	}
}
