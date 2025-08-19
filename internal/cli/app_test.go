package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// createTempConfig creates a temporary config file for testing
func createTempConfig(t *testing.T, content string) string {
	t.Helper()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.yaml")
	err := os.WriteFile(configPath, []byte(content), 0o600)
	if err != nil {
		t.Fatalf("Failed to create temp config: %v", err)
	}
	return configPath
}

func TestProcessHook(t *testing.T) {
	t.Parallel()

	configContent := `rules:
  - name: "block-go-test"
    pattern: "go test"
    action: "deny"
    message: "Use make test instead for better TDD integration"
    alternatives:
      - "make test          # Run all tests"
      - "make test-unit     # Run unit tests only"
    use_claude: false`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	hookInput := `{
		"command": "go test ./...",
		"args": ["go", "test", "./..."],
		"cwd": "/path/to/project"
	}`

	response, err := app.ProcessHook(strings.NewReader(hookInput))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should return a response since "go test" matches a rule
	if response == "" {
		t.Error("Expected non-empty response for blocked command")
	}

	if !strings.Contains(response, "make test") {
		t.Error("Response should suggest make test alternative")
	}
}

func TestProcessHookAllowed(t *testing.T) {
	t.Parallel()

	configContent := `rules:
  - name: "block-go-test"
    pattern: "go test"
    action: "deny"
    message: "Use make test instead for better TDD integration"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	hookInput := `{
		"command": "make test",
		"args": ["make", "test"],
		"cwd": "/path/to/project"
	}`

	response, err := app.ProcessHook(strings.NewReader(hookInput))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should return empty response since "make test" doesn't match any deny rule
	if response != "" {
		t.Errorf("Expected empty response for allowed command, got %s", response)
	}
}

func TestProcessHookDangerousCommand(t *testing.T) {
	t.Parallel()

	configContent := `rules:
  - name: "dangerous-rm"
    pattern: "rm -rf /*"
    action: "deny"
    message: "⚠️  Dangerous rm command detected"
    alternatives:
      - "Be more specific with your rm command"
      - "Use a safer alternative like moving to trash"
    use_claude: true
    claude_prompt: "Explain why this rm command is dangerous and suggest safer alternatives"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	hookInput := `{
		"command": "rm -rf /tmp",
		"args": ["rm", "-rf", "/tmp"],
		"cwd": "/path/to/project"
	}`

	response, err := app.ProcessHook(strings.NewReader(hookInput))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should return a response since dangerous rm command matches a rule
	if response == "" {
		t.Error("Expected non-empty response for dangerous command")
	}
}

func TestProcessHookPatternMatching(t *testing.T) {
	t.Parallel()

	configContent := `rules:
  - name: "block-go-test"
    pattern: "go test"
    action: "deny"
    message: "Use make test instead for better TDD integration"
    alternatives:
      - "make test          # Run all tests"
      - "make test-unit     # Run unit tests only"
    use_claude: false`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	hookInput := `{
		"command": "go test -v ./pkg/...",
		"args": ["go", "test", "-v", "./pkg/..."],
		"cwd": "/path/to/project"
	}`

	response, err := app.ProcessHook(strings.NewReader(hookInput))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should return a response since this matches "go test.*" pattern
	if response == "" {
		t.Error("Expected non-empty response for go test pattern match")
	}

	if !strings.Contains(response, "make test") {
		t.Error("Response should suggest make test alternative")
	}
}

func TestConfigurationIsUsed(t *testing.T) {
	t.Parallel()

	// This test ensures we're actually using the config file by checking for
	// a specific message from the config rather than hardcoded responses
	configContent := `rules:
  - name: "block-go-test"
    pattern: "go test"
    action: "deny"
    message: "Use make test instead for better TDD integration"
    use_claude: false`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	hookInput := `{
		"command": "go test ./...",
		"args": ["go", "test", "./..."],
		"cwd": "/path/to/project"
	}`

	response, err := app.ProcessHook(strings.NewReader(hookInput))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should contain the exact message from config file
	if !strings.Contains(response, "better TDD integration") {
		t.Error("Response should contain message from config file")
	}
}

func TestTestCommand(t *testing.T) {
	t.Parallel()

	configContent := `rules:
  - name: "block-go-test"
    pattern: "go test"
    action: "deny"
    message: "Use make test instead for better TDD integration"
    use_claude: false`

	configPath := createTempConfig(t, configContent)
	app := NewApp(configPath)

	result, err := app.TestCommand("go test ./...")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !strings.Contains(result, "blocked") {
		t.Error("Result should indicate command is blocked")
	}
}
