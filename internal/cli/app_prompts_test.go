package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wizzomafizzo/bumpers/internal/infrastructure/constants"
	"github.com/wizzomafizzo/bumpers/internal/platform/claude"
)

func TestProcessUserPromptWithContext(t *testing.T) {
	t.Parallel()
	ctx, getLogs := setupTestWithContext(t)

	// Create a temporary config with commands
	configContent := `
commands:
  - name: "test"
    send: "Test command executed"
`
	configFile := createTempConfig(t, configContent)
	app := NewApp(ctx, configFile)

	// Create UserPromptSubmit event with command
	promptJSON := `{"prompt": "` + constants.CommandPrefix + `test"}`

	// Test that ProcessUserPrompt works with context - this will fail until we add context parameter
	result, err := app.ProcessUserPrompt(ctx, json.RawMessage(promptJSON))
	require.NoError(t, err)
	assert.Contains(t, result, "Test command executed")

	// Verify that logs were captured without race conditions
	logs := getLogs()
	assert.Contains(t, logs, "processing UserPromptSubmit with prompt")
}

func TestProcessUserPrompt(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestWithContext(t)

	configContent := fmt.Sprintf(`rules:
  - match: "go test"
    send: "Use just test instead"
    generate: "off"
commands:
  - name: "help"
    send: "Available commands:\\n%shelp - Show this help\\n%sstatus - Show project status"
    generate: "off"
  - name: "status"
    send: "Project Status: All systems operational"
    generate: "off"`, constants.CommandPrefix, constants.CommandPrefix)

	configPath := createTempConfig(t, configContent)
	app := NewApp(ctx, configPath)

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name: fmt.Sprintf("Help command (%shelp)", constants.CommandPrefix),
			input: fmt.Sprintf(`{
				"prompt": "%shelp"
			}`, constants.CommandPrefix),
			want: fmt.Sprintf(
				`{"hookSpecificOutput":{"hookEventName":"UserPromptSubmit",`+
					`"additionalContext":"Available commands:\\n%shelp - Show this help\\n`+
					`%sstatus - Show project status"}}`,
				constants.CommandPrefix,
				constants.CommandPrefix),
			wantErr: false,
		},
		{
			name: fmt.Sprintf("Status command (%sstatus)", constants.CommandPrefix),
			input: fmt.Sprintf(`{
				"prompt": "%sstatus"
			}`, constants.CommandPrefix),
			want: `{"hookSpecificOutput":{"hookEventName":"UserPromptSubmit",` +
				`"additionalContext":"Project Status: All systems operational"}}`,
			wantErr: false,
		},
		{
			name: "Non-command prompt",
			input: `{
				"prompt": "regular question"
			}`,
			want:    "",
			wantErr: false,
		},
		{
			name: fmt.Sprintf("Invalid command index (%s5)", constants.CommandPrefix),
			input: fmt.Sprintf(`{
				"prompt": "%s5"
			}`, constants.CommandPrefix),
			want:    "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := app.ProcessUserPrompt(context.Background(), json.RawMessage(tt.input))

			if (err != nil) != tt.wantErr {
				t.Errorf("ProcessUserPrompt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result != tt.want {
				t.Errorf("ProcessUserPrompt() = %q, want %q", result, tt.want)
			}
		})
	}
}

func TestProcessUserPromptValidationResult(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestWithContext(t)

	// Create temporary config with named commands
	configContent := `
commands:
  - name: "test"
    send: "Test command message"
    generate: "off"
`

	configPath := createTempConfig(t, configContent)
	app := NewApp(ctx, configPath)

	// Test that named command prompts work
	promptJSON := `{"prompt": "` + constants.CommandPrefix + `test"}`
	result, err := app.ProcessUserPrompt(context.Background(), json.RawMessage(promptJSON))
	if err != nil {
		t.Fatalf("ProcessUserPrompt failed: %v", err)
	}

	expectedOutput := `{"hookSpecificOutput":{"hookEventName":"UserPromptSubmit",` +
		`"additionalContext":"Test command message"}}`
	if result != expectedOutput {
		t.Errorf("Expected hookSpecificOutput format for named command %q, got %q", expectedOutput, result)
	}
}

func TestProcessUserPromptWithCommandGeneration(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestWithContext(t)

	configContent := `commands:
  - name: "help"
    send: "Basic help message"
    generate: "always"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(ctx, configPath)

	// Set up mock launcher
	mockLauncher := claude.SetupMockLauncherWithDefaults()
	mockLauncher.SetResponseForPattern("", "Enhanced help message from AI")
	app.SetMockLauncher(mockLauncher)

	promptJSON := `{"prompt": "` + constants.CommandPrefix + `help"}`
	result, err := app.ProcessUserPrompt(context.Background(), json.RawMessage(promptJSON))
	if err != nil {
		t.Fatalf("ProcessUserPrompt failed: %v", err)
	}

	expectedOutput := `{"hookSpecificOutput":{"hookEventName":"UserPromptSubmit",` +
		`"additionalContext":"Enhanced help message from AI"}}`
	if result != expectedOutput {
		t.Errorf("Expected AI-generated command output %q, got %q", expectedOutput, result)
	}

	// Verify the mock was called with the right prompt
	if mockLauncher.GetCallCount() == 0 {
		t.Error("Expected mock launcher to be called for AI generation")
	}
}

func TestProcessUserPromptWithTemplate(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestWithContext(t)

	configContent := `commands:
  - name: "hello"
    send: "Hello {{.Name}}!"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(ctx, configPath)

	promptJSON := `{"prompt": "` + constants.CommandPrefix + `hello"}`
	result, err := app.ProcessUserPrompt(ctx, json.RawMessage(promptJSON))
	if err != nil {
		t.Fatalf("ProcessUserPrompt failed: %v", err)
	}

	expectedOutput := `{"hookSpecificOutput":{"hookEventName":"UserPromptSubmit",` +
		`"additionalContext":"Hello hello!"}}`
	if result != expectedOutput {
		t.Errorf("Expected templated message, got: %q", result)
	}
}

func TestProcessUserPromptWithTodayVariable(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestWithContext(t)

	configContent := `commands:
  - name: "hello"
    send: "Hello {{.Name}} on {{.Today}}!"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(ctx, configPath)

	promptJSON := `{"prompt": "` + constants.CommandPrefix + `hello"}`
	result, err := app.ProcessUserPrompt(ctx, json.RawMessage(promptJSON))
	if err != nil {
		t.Fatalf("ProcessUserPrompt failed: %v", err)
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
	expectedMessage := "Hello hello on " + expectedDate + "!"
	if additionalContext != expectedMessage {
		t.Errorf("Expected %q, got %q", expectedMessage, additionalContext)
	}
}

func TestProcessUserPromptWithCommandArguments(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestWithContext(t)

	configContent := `commands:
  - name: "test"
    send: "Command: {{.Name}}, Args: {{argc}}, First: {{argv 1}}"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(ctx, configPath)

	promptJSON := `{"prompt": "` + constants.CommandPrefix + `test arg1 arg2"}`
	result, err := app.ProcessUserPrompt(ctx, json.RawMessage(promptJSON))
	if err != nil {
		t.Fatalf("ProcessUserPrompt failed: %v", err)
	}

	// Parse the response to get the additionalContext
	var response map[string]any
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	hookOutput, ok := response["hookSpecificOutput"].(map[string]any)
	if !ok {
		t.Fatal("Response missing hookSpecificOutput")
	}

	additionalContext, ok := hookOutput["additionalContext"].(string)
	if !ok {
		t.Fatal("Response missing additionalContext")
	}

	expected := "Command: test, Args: 2, First: arg1"
	if additionalContext != expected {
		t.Errorf("Expected %q, got %q", expected, additionalContext)
	}
}

func TestProcessUserPromptWithNoArguments(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestWithContext(t)

	configContent := `commands:
  - name: "test"
    send: "Command: {{.Name}}, Args: {{argc}}"
    generate: "off"`

	configPath := createTempConfig(t, configContent)
	app := NewApp(ctx, configPath)

	promptJSON := `{"prompt": "` + constants.CommandPrefix + `test"}`
	result, err := app.ProcessUserPrompt(ctx, json.RawMessage(promptJSON))
	if err != nil {
		t.Fatalf("ProcessUserPrompt failed: %v", err)
	}

	// Parse the response
	var response map[string]any
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	hookOutput, ok := response["hookSpecificOutput"].(map[string]any)
	if !ok {
		t.Fatalf("Expected hookSpecificOutput to be map[string]any, got %T", response["hookSpecificOutput"])
	}
	additionalContext, ok := hookOutput["additionalContext"].(string)
	if !ok {
		t.Fatalf("Expected additionalContext to be string, got %T", hookOutput["additionalContext"])
	}

	expected := "Command: test, Args: 0"
	if additionalContext != expected {
		t.Errorf("Expected %q, got %q", expected, additionalContext)
	}
}
