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
	"github.com/wizzomafizzo/bumpers/internal/platform/claude/api"
	"github.com/wizzomafizzo/bumpers/internal/platform/storage"
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

	app, cachePath, tempDir := setupSessionCacheTest(t)
	populateSessionCache(t, cachePath, tempDir)

	sessionStartInput := `{
		"session_id": "abc123",
		"hook_event_name": "SessionStart",
		"source": "startup"
	}`

	// Process session start should clear session cache
	_, err := app.ProcessSessionStart(ctx, json.RawMessage(sessionStartInput))
	if err != nil {
		t.Fatalf("ProcessSessionStart failed: %v", err)
	}

	verifySessionCacheCleared(t, cachePath, tempDir)
}

func setupSessionCacheTest(t *testing.T) (app *App, cachePath, tempDir string) {
	t.Helper()
	ctx, _ := setupTestWithContext(t)
	tempDir = t.TempDir()

	// Set XDG_DATA_HOME to use temp directory for cache - this works with t.Parallel()
	dataHome := filepath.Join(tempDir, ".local", "share")
	t.Setenv("XDG_DATA_HOME", dataHome)

	configPath := createTempConfig(t, `session:
  - add: "Session started"`)
	app = NewApp(ctx, configPath)
	app.projectRoot = tempDir

	// Recreate SessionManager with correct project root
	app.sessionManager = NewSessionManager(configPath, tempDir, nil)

	// Get the actual cache path that the app will use
	storageManager := storage.New(afero.NewOsFs())
	var err error
	cachePath, err = storageManager.GetCachePath()
	if err != nil {
		t.Fatalf("Failed to get cache path: %v", err)
	}

	return
}

func populateSessionCache(t *testing.T, cachePath, tempDir string) {
	t.Helper()
	cache, err := ai.NewCacheWithProject(cachePath, tempDir)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	// Close cache after populating to allow ProcessSessionStart to access it
	defer func() {
		if closeErr := cache.Close(); closeErr != nil {
			t.Logf("Failed to close cache: %v", closeErr)
		}
	}()

	expiry := time.Now().Add(24 * time.Hour)
	sessionEntry := &ai.CacheEntry{
		GeneratedMessage: "Generated session message",
		OriginalMessage:  "Original message",
		Timestamp:        time.Now(),
		ExpiresAt:        &expiry,
	}

	err = cache.Put("test-session-key", sessionEntry)
	if err != nil {
		t.Fatalf("Failed to put session entry: %v", err)
	}

	// Verify entry exists
	retrieved, err := cache.Get("test-session-key")
	if err != nil || retrieved == nil {
		t.Fatal("Session entry should exist before ProcessSessionStart")
	}
}

func verifySessionCacheCleared(t *testing.T, cachePath, tempDir string) {
	t.Helper()
	cache, err := ai.NewCacheWithProject(cachePath, tempDir)
	if err != nil {
		t.Fatalf("Failed to reopen cache: %v", err)
	}
	t.Cleanup(func() { _ = cache.Close() })

	retrieved, err := cache.Get("test-session-key")
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
	app := NewApp(ctx, configPath)

	// Set up mock launcher
	mockLauncher := claude.SetupMockLauncherWithDefaults()
	mockLauncher.SetResponseForPattern("", "Enhanced session message from AI")
	app.SetMockLauncher(mockLauncher)

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
