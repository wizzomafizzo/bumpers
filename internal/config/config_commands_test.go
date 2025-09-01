package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	generateModeOff    = "off"
	generateModeOnce   = "once"
	generateModeAlways = "always"
)

// Helper function to load config from YAML and handle errors consistently
func loadConfigFromYAML(t *testing.T, yamlContent string) *Config {
	t.Helper()
	config, err := LoadFromYAML([]byte(yamlContent))
	require.NoError(t, err, "Should load config from YAML without error")
	return config
}

// Test the Commands feature
func TestConfigWithCommands(t *testing.T) {
	t.Parallel()

	yamlContent := `rules:
  - match:
      pattern: "go test.*"
    send: "Use just test instead"
commands:
  - name: "help"
    send: "Available commands:\\n!help - Show this help\\n!status - Show project status"
  - name: "status"
    send: "Project Status: All systems operational"`

	config := loadConfigFromYAML(t, yamlContent)

	// Test rules are still loaded
	require.Len(t, config.Rules, 1, "Should have exactly 1 rule")

	// Test commands are loaded
	require.Len(t, config.Commands, 2, "Should have exactly 2 commands")

	expectedMessages := []string{
		"Available commands:\\n!help - Show this help\\n!status - Show project status",
		"Project Status: All systems operational",
	}

	for i, cmd := range config.Commands {
		require.Equal(t, expectedMessages[i], cmd.Send, "Command %d should have expected message", i)
	}
}

// Test validation allows empty rules if commands are present
func TestConfigValidationWithCommands(t *testing.T) {
	t.Parallel()

	yamlContent := `commands:
  - name: "help"
    send: "Help command response"`

	config := loadConfigFromYAML(t, yamlContent)

	require.Len(t, config.Commands, 1, "Should have exactly 1 command")
	require.Empty(t, config.Rules, "Should have no rules")
}

// Test Command Generate field defaults to session
func TestCommandGenerateFieldDefaultToOff(t *testing.T) {
	t.Parallel()

	yamlContent := `commands:
  - name: "help"
    send: "Help message"`

	config, err := LoadFromYAML([]byte(yamlContent))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	command := config.Commands[0]
	generate := command.GetGenerate()
	if generate.Mode != generateModeOff {
		t.Errorf("Expected Command Generate.Mode to be 'off', got %s", generate.Mode)
	}
}

// Test Command Generate shortform
func TestCommandGenerateShortform(t *testing.T) {
	t.Parallel()

	yamlContent := `commands:
  - name: "help"
    send: "Help message"
    generate: "once"`

	config, err := LoadFromYAML([]byte(yamlContent))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	command := config.Commands[0]
	generate := command.GetGenerate()
	if generate.Mode != generateModeOnce {
		t.Errorf("Expected Command Generate.Mode to be 'once', got %s", generate.Mode)
	}
}

// Test Command Generate full form
func TestCommandGenerateFullForm(t *testing.T) {
	t.Parallel()

	yamlContent := `commands:
  - name: "help"
    send: "Help message"
    generate:
      mode: "always"
      prompt: "Custom prompt"`

	config, err := LoadFromYAML([]byte(yamlContent))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	command := config.Commands[0]
	generate := command.GetGenerate()
	if generate.Mode != generateModeAlways {
		t.Errorf("Expected Command Generate.Mode to be 'always', got %s", generate.Mode)
	}
	if generate.Prompt != "Custom prompt" {
		t.Errorf("Expected Command Generate.Prompt to be 'Custom prompt', got %s", generate.Prompt)
	}
}

// Test Session Generate field defaults to session
func TestSessionGenerateFieldDefaultToOff(t *testing.T) {
	t.Parallel()

	yamlContent := `session:
  - add: "Session note"`

	config, err := LoadFromYAML([]byte(yamlContent))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	session := config.Session[0]
	generate := session.GetGenerate()
	if generate.Mode != "off" {
		t.Errorf("Expected Session Generate.Mode to be 'off', got %s", generate.Mode)
	}
}

// Test Session Generate shortform
func TestSessionGenerateShortform(t *testing.T) {
	t.Parallel()

	yamlContent := `session:
  - add: "Session note"
    generate: "once"`

	config, err := LoadFromYAML([]byte(yamlContent))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	session := config.Session[0]
	generate := session.GetGenerate()
	if generate.Mode != "once" {
		t.Errorf("Expected Session Generate.Mode to be 'once', got %s", generate.Mode)
	}
}

// Test Session Generate full form
func TestSessionGenerateFullForm(t *testing.T) {
	t.Parallel()

	yamlContent := `session:
  - add: "Session note"
    generate:
      mode: "always"
      prompt: "Custom prompt"`

	config, err := LoadFromYAML([]byte(yamlContent))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	session := config.Session[0]
	generate := session.GetGenerate()
	if generate.Mode != "always" {
		t.Errorf("Expected Session Generate.Mode to be 'always', got %s", generate.Mode)
	}
	if generate.Prompt != "Custom prompt" {
		t.Errorf("Expected Session Generate.Prompt to be 'Custom prompt', got %s", generate.Prompt)
	}
}

func TestConfigWithNotes(t *testing.T) {
	t.Parallel()

	yamlContent := `rules:
  - match:
      pattern: "go test"
    send: "Use just test instead"
session:
  - add: "Remember to run tests first"
  - add: "Check CLAUDE.md for project conventions"`

	config, err := LoadFromYAML([]byte(yamlContent))
	if err != nil {
		t.Fatalf("Expected no error loading config with notes, got %v", err)
	}

	if len(config.Session) != 2 {
		t.Fatalf("Expected 2 notes, got %d", len(config.Session))
	}

	expectedMessages := []string{
		"Remember to run tests first",
		"Check CLAUDE.md for project conventions",
	}

	for i, note := range config.Session {
		if note.Add != expectedMessages[i] {
			t.Errorf("Expected note %d message %q, got %q", i, expectedMessages[i], note.Add)
		}
	}
}

func TestConfigValidationWithNotesOnly(t *testing.T) {
	t.Parallel()

	yamlContent := `session:
  - add: "Just a note"`

	config, err := LoadFromYAML([]byte(yamlContent))
	if err != nil {
		t.Fatalf("Expected no error loading config with notes only, got %v", err)
	}

	if len(config.Session) != 1 {
		t.Fatalf("Expected 1 note, got %d", len(config.Session))
	}

	if len(config.Rules) != 0 {
		t.Fatalf("Expected 0 rules, got %d", len(config.Rules))
	}
}

func TestDefaultConfigIncludesNotes(t *testing.T) {
	t.Parallel()

	config := DefaultConfig()

	if len(config.Session) == 0 {
		t.Error("Expected default config to include example notes")
	}

	// Check that notes contain helpful messages
	hasUsefulNote := false
	for _, note := range config.Session {
		if note.Add != "" {
			hasUsefulNote = true
			break
		}
	}

	if !hasUsefulNote {
		t.Error("Expected at least one note with non-empty message")
	}
}

func TestDefaultConfigCommandsHaveNames(t *testing.T) {
	t.Parallel()

	config := DefaultConfig()

	if len(config.Commands) == 0 {
		t.Error("Expected default config to include commands")
	}

	expectedNames := []string{"help", "status", "docs"}

	if len(config.Commands) != len(expectedNames) {
		t.Fatalf("Expected %d commands, got %d", len(expectedNames), len(config.Commands))
	}

	for i, cmd := range config.Commands {
		if cmd.Name == "" {
			t.Errorf("Command %d is missing required Name field", i)
		}
		if cmd.Name != expectedNames[i] {
			t.Errorf("Expected command %d to have name %q, got %q", i, expectedNames[i], cmd.Name)
		}
	}
}
