package config

import (
	"testing"
)

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

	config, err := LoadFromYAML([]byte(yamlContent))
	if err != nil {
		t.Fatalf("Expected no error loading config with commands, got %v", err)
	}

	// Test rules are still loaded
	if len(config.Rules) != 1 {
		t.Fatalf("Expected 1 rule, got %d", len(config.Rules))
	}

	// Test commands are loaded
	if len(config.Commands) != 2 {
		t.Fatalf("Expected 2 commands, got %d", len(config.Commands))
	}

	expectedMessages := []string{
		"Available commands:\\n!help - Show this help\\n!status - Show project status",
		"Project Status: All systems operational",
	}

	for i, cmd := range config.Commands {
		if cmd.Send != expectedMessages[i] {
			t.Errorf("Expected command %d message %q, got %q", i, expectedMessages[i], cmd.Send)
		}
	}
}

// Test validation allows empty rules if commands are present
func TestConfigValidationWithCommands(t *testing.T) {
	t.Parallel()

	yamlContent := `commands:
  - name: "help"
    send: "Help command response"`

	config, err := LoadFromYAML([]byte(yamlContent))
	if err != nil {
		t.Fatalf("Expected no error loading config with commands only, got %v", err)
	}

	if len(config.Commands) != 1 {
		t.Fatalf("Expected 1 command, got %d", len(config.Commands))
	}

	if len(config.Rules) != 0 {
		t.Fatalf("Expected 0 rules, got %d", len(config.Rules))
	}
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
	if generate.Mode != "off" {
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
	if generate.Mode != "once" {
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
	if generate.Mode != "always" {
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
