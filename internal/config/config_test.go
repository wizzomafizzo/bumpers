package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to test config loading with basic rule validation
func testConfigLoading(t *testing.T, configFile, expectedPattern string) {
	t.Helper()

	config, err := Load(configFile)
	require.NoError(t, err, "Config loading should not error")

	require.Len(t, config.Rules, 1, "Should have exactly 1 rule")

	rule := config.Rules[0]
	assert.Equal(t, expectedPattern, rule.Match, "Rule pattern should match expected")
}

func TestLoadConfig(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test.yaml")

	yamlContent := `rules:
  - match: "go test.*"
    send: "Use just test instead"
`

	err := os.WriteFile(configFile, []byte(yamlContent), 0o600)
	require.NoError(t, err, "Should be able to write test config file")

	testConfigLoading(t, configFile, "go test.*")
}

func TestLoadConfigWithAlternatives(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test.yaml")

	yamlContent := `rules:
  - match: "go test.*"
    send: "Use just test instead for better TDD integration"
`

	err := os.WriteFile(configFile, []byte(yamlContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := Load(configFile)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	rule := config.Rules[0]
	if rule.Send == "" {
		t.Error("Expected Add to be set, got empty string")
	}
}

func TestLoadConfigWithNewStructure(t *testing.T) {
	t.Parallel()

	yamlContent := `rules:
  - match: "go test*"
    send: |
      Use just test instead for better TDD integration
      
      Try one of these alternatives:
      • just test          # Run all tests
      • just test-unit     # Run unit tests only
    generate:
      mode: "always"
      prompt: "Explain why direct go test is discouraged"
`

	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test.yaml")

	err := os.WriteFile(configFile, []byte(yamlContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := Load(configFile)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	rule := config.Rules[0]
	assert.NotEmpty(t, rule.Send, "Expected message to be populated")

	generate := rule.GetGenerate()
	assert.Equal(t, "always", generate.Mode, "Expected Generate.Mode to be 'always'")
	assert.Equal(t, "Explain why direct go test is discouraged", generate.Prompt,
		"Expected Generate.Prompt to be set correctly")
}

func TestSimplifiedSchemaHasNoActionConstants(t *testing.T) {
	t.Parallel()

	// This test will fail until action constants are removed
	// It verifies the constants don't exist by trying to use them
	// We can't directly test for their non-existence, but we can test
	// that the simplified schema works without them

	yamlContent := `rules:
  - match: "test*"
    send: "Test response"
`
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test.yaml")

	err := os.WriteFile(configFile, []byte(yamlContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := Load(configFile)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify that rules work without action constants
	rule := config.Rules[0]
	if rule.Match != "test*" {
		t.Errorf("Expected pattern 'test*', got %s", rule.Match)
	}
	if rule.Send != "Test response" {
		t.Errorf("Expected message 'Test response', got %s", rule.Send)
	}

	// This test implicitly shows that action constants are not needed
	// since the rule works without specifying any action
}

func TestLoadConfigFromFile(t *testing.T) {
	t.Parallel()

	// Create temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.yaml")
	configContent := `rules:
  - match: "go test.*"
    send: "Use just test instead for better TDD integration"
    generate:
      mode: "session"
      prompt: "Explain why direct go test is discouraged"`

	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create temp config: %v", err)
	}

	config, err := Load(configPath)
	if err != nil {
		t.Fatalf("Expected no error loading config file, got %v", err)
	}

	if len(config.Rules) == 0 {
		t.Fatal("Expected at least one rule from config file")
	}

	// Check first rule
	rule := config.Rules[0]
	if rule.Match != "go test.*" {
		t.Errorf("Expected first rule pattern 'go test.*', got %s", rule.Match)
	}
}

func TestGenerateFieldAsString(t *testing.T) {
	t.Parallel()

	yamlContent := `rules:
  - match: "go test"
    send: "Use just test instead"
    generate: "session"
`

	config, err := LoadFromYAML([]byte(yamlContent))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	rule := config.Rules[0]
	generate := rule.GetGenerate()
	if generate.Mode != "session" {
		t.Errorf("Expected Generate.Mode to be 'session', got %s", generate.Mode)
	}
}

func TestGenerateFieldDefaultToSession(t *testing.T) {
	t.Parallel()

	yamlContent := `rules:
  - match: "go test"
    send: "Use just test instead"
`

	config, err := LoadFromYAML([]byte(yamlContent))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	rule := config.Rules[0]
	generate := rule.GetGenerate()
	if generate.Mode != "session" {
		t.Errorf("Expected Generate.Mode to default to 'session', got %s", generate.Mode)
	}
}

func TestLoadConfigSimplifiedSchema(t *testing.T) {
	t.Parallel()

	yamlContent := `rules:
  - match: "go test*"
    send: |
      Use just test instead for better TDD integration
      
      Try one of these alternatives:
      • just test          # Run all tests
      • just test-unit     # Run unit tests only
  - match: "rm -rf /*"
    send: "Dangerous rm command detected! Be more specific with your rm command."
    generate:
      mode: "always"
      prompt: "Explain why this rm command is dangerous and suggest safer alternatives"
`

	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test.yaml")

	err := os.WriteFile(configFile, []byte(yamlContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := Load(configFile)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(config.Rules) != 2 {
		t.Fatalf("Expected 2 rules, got %d", len(config.Rules))
	}

	// Test first rule (go test)
	rule1 := config.Rules[0]
	if rule1.Match != "go test*" {
		t.Errorf("Expected pattern 'go test*', got %s", rule1.Match)
	}
	if rule1.Send == "" {
		t.Error("Expected message to be populated")
	}
	generate1 := rule1.GetGenerate()
	if generate1.Mode != "session" {
		t.Error("Expected Generate.Mode to default to 'session'")
	}
	// In simplified schema, these fields don't exist at all

	// Test second rule (dangerous rm)
	rule2 := config.Rules[1]
	if rule2.Match != "rm -rf /*" {
		t.Errorf("Expected pattern 'rm -rf /*', got %s", rule2.Match)
	}
	generate2 := rule2.GetGenerate()
	if generate2.Mode != "always" {
		t.Errorf("Expected Generate.Mode to be 'always', got %s", generate2.Mode)
	}
	if generate2.Prompt == "" {
		t.Error("Expected Generate.Prompt to be populated when Generate is set")
	}
	// In simplified schema, these fields don't exist at all
}

// Test config validation
func TestConfigValidation(t *testing.T) {
	t.Parallel()

	t.Run("ValidConfigs", testValidConfigs)
	t.Run("InvalidConfigs", testInvalidConfigs)
}

func testValidConfigs(t *testing.T) {
	t.Parallel()
	tests := []configTestCase{
		{
			name: "valid config",
			yamlContent: `rules:
  - match: "go test.*"
    send: "Use just test instead"`,
			expectError: false,
		},
		{
			name: "valid generate with prompt",
			yamlContent: `rules:
  - match: "test.*"
    generate:
      mode: "always"
      prompt: "Test prompt"`,
			expectError: false,
		},
		{
			name: "rule with message only",
			yamlContent: `rules:
  - match: "test.*"
    send: "Use just test instead"`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			validateConfigTest(t, tt)
		})
	}
}

func testInvalidConfigs(t *testing.T) {
	t.Parallel()
	tests := []configTestCase{
		{
			name:          "empty rules",
			yamlContent:   `rules: []`,
			expectError:   true,
			errorContains: "must contain at least one rule",
		},
		{
			name:          "missing rules entirely",
			yamlContent:   `claude_binary: "/usr/bin/claude"`,
			expectError:   true,
			errorContains: "must contain at least one rule",
		},
		{
			name: "empty pattern",
			yamlContent: `rules:
  - match: ""
    send: "Empty pattern"`,
			expectError:   true,
			errorContains: "match field is required",
		},
		{
			name: "invalid regex pattern",
			yamlContent: `rules:
  - match: "[invalid"
    send: "Invalid regex"`,
			expectError:   true,
			errorContains: "invalid regex pattern",
		},
		{
			name: "rule with generate off and no send",
			yamlContent: `rules:
  - match: "test.*"
    generate: "off"`,
			expectError:   true,
			errorContains: "must provide either a message or generate configuration",
		},
		{
			name: "multiple rules with one invalid",
			yamlContent: `rules:
  - match: "valid.*"
    send: "Valid rule"
  - match: "[invalid"
    send: "Invalid rule"`,
			expectError:   true,
			errorContains: "rule 2 validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			validateConfigTest(t, tt)
		})
	}
}

type configTestCase struct {
	name          string
	yamlContent   string
	errorContains string
	expectError   bool
}

func validateConfigTest(t *testing.T, tt configTestCase) {
	_, err := LoadFromYAML([]byte(tt.yamlContent))

	if tt.expectError {
		if err == nil {
			t.Error("Expected error but got none")
			return
		}
		if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
			t.Errorf("Expected error to contain '%s', got: %s", tt.errorContains, err.Error())
		}
	} else if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

// Test basic configuration without logging
func TestConfigWithoutLogging(t *testing.T) {
	t.Parallel()

	yamlContent := `rules:
  - match: "go test.*"
    send: "Use just test instead"`

	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test.yaml")

	err := os.WriteFile(configFile, []byte(yamlContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := Load(configFile)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check that basic config loads successfully without logging
	if len(config.Rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(config.Rules))
	}
}

// TestPartialConfigLoading tests that valid rules can be loaded even when some rules are invalid
func TestPartialConfigLoading(t *testing.T) {
	t.Parallel()

	yamlContent := `rules:
  - match: "go test.*"
    send: "Use just test instead"
  - match: "[invalid regex"
    send: "This rule has invalid regex"
  - match: "rm -rf"
    send: "Dangerous command - use safer alternatives"
  - match: ""
    send: "Empty pattern rule"
`

	// This should eventually work with partial loading
	partialConfig, err := LoadPartial([]byte(yamlContent))
	if err != nil {
		t.Fatalf("Expected LoadPartial to succeed even with invalid rules, got %v", err)
	}

	// Should have 2 valid rules (go test and rm -rf)
	if len(partialConfig.Rules) != 2 {
		t.Errorf("Expected 2 valid rules, got %d", len(partialConfig.Rules))
	}

	// Should have 2 warnings (invalid regex and empty pattern)
	if len(partialConfig.ValidationWarnings) != 2 {
		t.Errorf("Expected 2 validation warnings, got %d", len(partialConfig.ValidationWarnings))
	}

	// Verify the valid rules are the correct ones
	expectedPatterns := []string{"go test.*", "rm -rf"}
	for i, rule := range partialConfig.Rules {
		if rule.Match != expectedPatterns[i] {
			t.Errorf("Expected rule %d pattern '%s', got '%s'", i, expectedPatterns[i], rule.Match)
		}
	}

	// Debug: print warnings
	for i, warning := range partialConfig.ValidationWarnings {
		t.Logf("Warning %d: Rule index %d, pattern '%s', error: %v",
			i, warning.RuleIndex, warning.Rule.Match, warning.Error)
	}
}

// Helper function for substring checking
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// checkRulePattern validates a specific rule pattern and message
func checkRulePattern(t *testing.T, rule *Rule, patternName, expectedPattern, expectedMessage string) bool {
	if strings.Contains(rule.Match, expectedPattern) {
		if !strings.Contains(rule.Send, expectedMessage) {
			t.Errorf("Expected %s rule to mention %s", patternName, expectedMessage)
		}
		return true
	}
	return false
}

// validateBasicDefaults checks basic configuration validation
func validateBasicDefaults(t *testing.T, config *Config) {
	if len(config.Rules) == 0 {
		t.Error("Expected default config to have rules")
	}
}

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	config := DefaultConfig()

	// Verify default config has expected rules
	if len(config.Rules) == 0 {
		t.Fatal("Expected default config to have rules")
	}

	// Check for expected rule patterns
	foundGoTest := false
	foundLint := false
	foundTmp := false

	for _, rule := range config.Rules {
		if checkRulePattern(t, &rule, "go test", "go test", "just test") {
			foundGoTest = true
		}
		if checkRulePattern(t, &rule, "lint", "gci|go vet", "just lint fix") {
			foundLint = true
		}
		if strings.Contains(rule.Match, "cd /tmp") {
			foundTmp = true
			if !strings.Contains(rule.Send, "tmp") {
				t.Errorf("Expected tmp rule to mention tmp, got: %s", rule.Send)
			}
		}
	}

	if !foundGoTest {
		t.Error("Expected default config to have go test rule")
	}
	if !foundLint {
		t.Error("Expected default config to have lint rule")
	}
	if !foundTmp {
		t.Error("Expected default config to have tmp rule")
	}

	validateBasicDefaults(t, config)

	// Check for example commands
	if len(config.Commands) == 0 {
		t.Error("Expected default config to have example commands")
	}
}

func TestDefaultConfigYAML(t *testing.T) {
	t.Parallel()

	yamlBytes, err := DefaultConfigYAML()
	if err != nil {
		t.Fatalf("DefaultConfigYAML failed: %v", err)
	}

	if len(yamlBytes) == 0 {
		t.Fatal("Expected YAML bytes, got empty")
	}

	// Verify the YAML contains expected patterns
	yamlStr := string(yamlBytes)
	expectedPatterns := []string{
		"rules:",
		"match: ^go test",
		"just test",
		"gci|go vet",
		"just lint fix",
		"cd /tmp",
		"tmp",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(yamlStr, pattern) {
			t.Errorf("Expected YAML to contain %q, got:\n%s", pattern, yamlStr)
		}
	}
}

// Test the Commands feature
func TestConfigWithCommands(t *testing.T) {
	t.Parallel()

	yamlContent := `rules:
  - match: "go test.*"
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
  - match: "go test"
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

func TestRuleWithToolsField(t *testing.T) {
	t.Parallel()

	yamlData := `
rules:
  - match: "^rm -rf"
    tool: "^(Bash|Task)$"
    send: "Dangerous command"
  - match: "password"
    tool: "^(Write|Edit)$"
    send: "No hardcoded secrets"
  - match: "test"
    send: "Bash only rule (no tool field)"
`

	config, err := LoadFromYAML([]byte(yamlData))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(config.Rules) != 3 {
		t.Fatalf("Expected 3 rules, got %d", len(config.Rules))
	}

	// First rule with tools field
	if config.Rules[0].Match != "^rm -rf" {
		t.Errorf("Expected pattern '^rm -rf', got '%s'", config.Rules[0].Match)
	}
	if config.Rules[0].Tool != "^(Bash|Task)$" {
		t.Errorf("Expected tools '^(Bash|Task)$', got '%s'", config.Rules[0].Tool)
	}

	// Second rule with tools field
	if config.Rules[1].Tool != "^(Write|Edit)$" {
		t.Errorf("Expected tools '^(Write|Edit)$', got '%s'", config.Rules[1].Tool)
	}

	// Third rule without tools field (should be empty string)
	if config.Rules[2].Tool != "" {
		t.Errorf("Expected empty tools field, got '%s'", config.Rules[2].Tool)
	}
}

func TestRuleValidationWithInvalidToolsRegex(t *testing.T) {
	t.Parallel()

	yamlData := `
rules:
  - match: "test"
    tool: "[invalid regex"
    send: "Test message"
`

	_, err := LoadFromYAML([]byte(yamlData))
	if err == nil {
		t.Fatal("Expected error for invalid tools regex, got nil")
	}

	expectedError := "invalid tools regex pattern"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain '%s', got '%s'", expectedError, err.Error())
	}
}

func TestRuleValidationWithGenerateField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		yamlData    string
		errorText   string
		expectError bool
	}{
		{
			name: "valid generate field - once",
			yamlData: `
rules:
  - match: "test"
    send: "Test message"
    generate:
      mode: "once"
`,
			expectError: false,
		},
		{
			name: "valid generate field - session",
			yamlData: `
rules:
  - match: "test"
    send: "Test message"
    generate:
      mode: "session"
`,
			expectError: false,
		},
		{
			name: "valid generate field - always",
			yamlData: `
rules:
  - match: "test"
    send: "Test message"
    generate:
      mode: "always"
`,
			expectError: false,
		},
		{
			name: "valid generate field - off",
			yamlData: `
rules:
  - match: "test"  
    send: "Test message"
    generate:
      mode: "off"
`,
			expectError: false,
		},
		{
			name: "invalid generate field",
			yamlData: `
rules:
  - match: "test"
    send: "Test message"
    generate:
      mode: "invalid"
`,
			expectError: true,
			errorText:   "invalid generate mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			testRuleGenerateValidation(t, tt)
		})
	}
}

// testRuleGenerateValidation is a helper function to reduce complexity
func testRuleGenerateValidation(t *testing.T, tt struct {
	name        string
	yamlData    string
	errorText   string
	expectError bool
},
) {
	_, err := LoadFromYAML([]byte(tt.yamlData))

	if tt.expectError {
		if err == nil {
			t.Fatalf("Expected error for %s, got nil", tt.name)
		}
		if !strings.Contains(err.Error(), tt.errorText) {
			t.Errorf("Expected error to contain '%s', got '%s'", tt.errorText, err.Error())
		}
		return
	}
	if err != nil {
		t.Errorf("Expected no error for %s, got %v", tt.name, err)
	}
}

// Benchmark tests for config parsing performance
func BenchmarkLoadConfig(b *testing.B) {
	tempDir := b.TempDir()
	configFile := filepath.Join(tempDir, "test.yaml")

	yamlContent := `rules:
  - match: "^go test"
    send: "Use just test instead"
  - match: "^rm -rf"
    send: "Dangerous command"
  - match: "^git push"
    send: "Review before pushing"
`

	err := os.WriteFile(configFile, []byte(yamlContent), 0o600)
	if err != nil {
		b.Fatalf("Failed to write config file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Load(configFile)
	}
}

// Fuzz test for config YAML parsing
func FuzzLoadPartial(f *testing.F) {
	// Add valid seed inputs
	f.Add(`rules:
  - match: "go test"
    send: "Use just test"`)
	f.Add(`rules:
  - match: "^rm -rf"
    tool: "Bash"
    send: "Dangerous"`)
	f.Add(`rules: []`)

	f.Fuzz(func(_ *testing.T, input string) {
		_, _ = LoadPartial([]byte(input))
		// No assertions - just ensuring no panics occur
	})
}

// Example demonstrates how to create and use a basic configuration
func ExampleLoadFromYAML() {
	yamlConfig := `rules:
  - match: "go test.*"
    send: "Use 'just test' instead for better TDD integration"
    generate:
      mode: "session"
      prompt: "Explain the benefits of using just test vs go test"`

	config, err := LoadFromYAML([]byte(yamlConfig))
	if err != nil {
		_, _ = fmt.Printf("Error: %v\n", err)
		return
	}

	_, _ = fmt.Printf("Loaded %d rules\n", len(config.Rules))
	_, _ = fmt.Printf("First rule pattern: %s\n", config.Rules[0].Match)

	// Output:
	// Loaded 1 rules
	// First rule pattern: go test.*
}

// Example shows how to work with the default configuration
func ExampleDefaultConfig() {
	config := DefaultConfig()

	_, _ = fmt.Printf("Default rules: %d\n", len(config.Rules))
	_, _ = fmt.Printf("Has commands: %v\n", len(config.Commands) > 0)
	_, _ = fmt.Printf("Has session notes: %v\n", len(config.Session) > 0)

	// Check for go test rule
	hasGoTestRule := false
	for i := range config.Rules {
		if strings.Contains(config.Rules[i].Match, "go test") {
			hasGoTestRule = true
			break
		}
	}
	_, _ = fmt.Printf("Has go test rule: %v\n", hasGoTestRule)

	// Output:
	// Default rules: 3
	// Has commands: true
	// Has session notes: true
	// Has go test rule: true
}

func getEventSourcesTestCases() []struct {
	name            string
	yamlContent     string
	expectedEvent   string
	expectedSources []string
} {
	return []struct {
		name            string
		yamlContent     string
		expectedEvent   string
		expectedSources []string
	}{
		{
			name: "post-tool reasoning matching",
			yamlContent: `rules:
  - match: "not related to my changes"
    send: "AI claiming unrelated"
    event: "post"
    sources: ["reasoning"]`,
			expectedEvent:   "post",
			expectedSources: []string{"reasoning"},
		},
		{
			name: "pre-tool command matching",
			yamlContent: `rules:
  - match: "^rm -rf"
    send: "Dangerous deletion"
    event: "pre"
    sources: ["command"]`,
			expectedEvent:   "pre",
			expectedSources: []string{"command"},
		},
		{
			name: "tool output matching",
			yamlContent: `rules:
  - match: "error|failed"
    send: "Command failed"
    event: "post" 
    sources: ["tool_output"]`,
			expectedEvent:   "post",
			expectedSources: []string{"tool_output"},
		},
		{
			name: "multiple field matching",
			yamlContent: `rules:
  - match: "password|secret"
    send: "Avoid secrets"
    event: "pre"
    sources: ["command", "content"]`,
			expectedEvent:   "pre",
			expectedSources: []string{"command", "content"},
		},
		{
			name: "defaults to pre event when omitted",
			yamlContent: `rules:
  - match: "dangerous"
    send: "Be careful"
    sources: ["command"]`,
			expectedEvent:   "pre",
			expectedSources: []string{"command"},
		},
		{
			name: "no default sources when omitted",
			yamlContent: `rules:
  - match: "unrelated"
    send: "AI deflection"
    event: "post"`,
			expectedEvent:   "post",
			expectedSources: nil,
		},
	}
}

func runEventSourcesConfigurationTest(t *testing.T, tc struct {
	name            string
	yamlContent     string
	expectedEvent   string
	expectedSources []string
},
) {
	config, err := LoadFromYAML([]byte(tc.yamlContent))
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	if len(config.Rules) != 1 {
		t.Fatalf("Expected 1 rule, got %d", len(config.Rules))
	}

	rule := config.Rules[0]
	rule.ValidateEventSources() // Apply defaults
	assert.Equal(t, tc.expectedEvent, rule.Event)
	assert.Equal(t, tc.expectedSources, rule.Sources)
}

// TestEventSourcesConfiguration tests the new Event and Sources configuration syntax
func TestEventSourcesConfiguration(t *testing.T) {
	t.Parallel()

	testCases := getEventSourcesTestCases()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			runEventSourcesConfigurationTest(t, tc)
		})
	}
}
