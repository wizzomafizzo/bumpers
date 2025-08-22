package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Helper function to test config loading with basic rule validation
func testConfigLoading(t *testing.T, configFile, expectedPattern string) {
	t.Helper()

	config, err := Load(configFile)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(config.Rules) != 1 {
		t.Fatalf("Expected 1 rule, got %d", len(config.Rules))
	}

	rule := config.Rules[0]
	if rule.Pattern != expectedPattern {
		t.Errorf("Expected rule pattern '%s', got %s", expectedPattern, rule.Pattern)
	}
}

func TestLoadConfig(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test.yaml")

	yamlContent := `rules:
  - pattern: "go test.*"
    response: "Use make test instead"
`

	err := os.WriteFile(configFile, []byte(yamlContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	testConfigLoading(t, configFile, "go test.*")
}

func TestLoadConfigWithAlternatives(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test.yaml")

	yamlContent := `rules:
  - pattern: "go test.*"
    response: "Use make test instead for better TDD integration"
    alternatives:
      - "make test          # Run all tests"
      - "make test-unit     # Run unit tests only"
    use_claude: false
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
	if len(rule.Alternatives) != 2 {
		t.Fatalf("Expected 2 alternatives, got %d", len(rule.Alternatives))
	}

	if rule.UseClaude {
		t.Errorf("Expected UseClaude to be false, got %v", rule.UseClaude)
	}
}

func TestLoadConfigWithNewStructure(t *testing.T) {
	t.Parallel()

	yamlContent := `rules:
  - pattern: "go test*"
    response: |
      Use make test instead for better TDD integration
      
      Try one of these alternatives:
      • make test          # Run all tests
      • make test-unit     # Run unit tests only
    use_claude: false
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
	if rule.Response == "" {
		t.Error("Expected response to be populated")
	}

	// For now, UseClaude is still a bool, so this test checks the zero value
	if rule.UseClaude {
		t.Errorf("Expected UseClaude to be false (zero value), got %v", rule.UseClaude)
	}

	if rule.Prompt != "Explain why direct go test is discouraged" {
		t.Errorf("Expected prompt to be set correctly, got %s", rule.Prompt)
	}
}

func TestSimplifiedSchemaHasNoActionConstants(t *testing.T) {
	t.Parallel()

	// This test will fail until action constants are removed
	// It verifies the constants don't exist by trying to use them
	// We can't directly test for their non-existence, but we can test
	// that the simplified schema works without them

	yamlContent := `rules:
  - pattern: "test*"
    response: "Test response"
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
	if rule.Pattern != "test*" {
		t.Errorf("Expected pattern 'test*', got %s", rule.Pattern)
	}
	if rule.Response != "Test response" {
		t.Errorf("Expected response 'Test response', got %s", rule.Response)
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
  - pattern: "go test.*"
    response: "Use make test instead for better TDD integration"
    alternatives:
      - "make test          # Run all tests"
      - "make test-unit     # Run unit tests only"
    use_claude: false`

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
	if rule.Pattern != "go test.*" {
		t.Errorf("Expected first rule pattern 'go test.*', got %s", rule.Pattern)
	}
}

func TestLoadConfigSimplifiedSchema(t *testing.T) {
	t.Parallel()

	yamlContent := `rules:
  - pattern: "go test*"
    response: |
      Use make test instead for better TDD integration
      
      Try one of these alternatives:
      • make test          # Run all tests
      • make test-unit     # Run unit tests only
  - pattern: "rm -rf /*"
    response: "Dangerous rm command detected! Be more specific with your rm command."
    use_claude: true
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
	if rule1.Pattern != "go test*" {
		t.Errorf("Expected pattern 'go test*', got %s", rule1.Pattern)
	}
	if rule1.Response == "" {
		t.Error("Expected response to be populated")
	}
	if rule1.UseClaude {
		t.Error("Expected UseClaude to be false by default")
	}
	// In simplified schema, these fields don't exist at all

	// Test second rule (dangerous rm)
	rule2 := config.Rules[1]
	if rule2.Pattern != "rm -rf /*" {
		t.Errorf("Expected pattern 'rm -rf /*', got %s", rule2.Pattern)
	}
	if !rule2.UseClaude {
		t.Error("Expected UseClaude to be true")
	}
	if rule2.Prompt == "" {
		t.Error("Expected prompt to be populated when UseClaude is true")
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
  - pattern: "go test.*"
    response: "Use make test instead"`,
			expectError: false,
		},
		{
			name: "valid use_claude with prompt",
			yamlContent: `rules:
  - pattern: "test.*"
    use_claude: true
    prompt: "Test prompt"`,
			expectError: false,
		},
		{
			name: "rule with alternatives only",
			yamlContent: `rules:
  - pattern: "test.*"
    alternatives:
      - "make test"`,
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
  - pattern: ""
    response: "Empty pattern"`,
			expectError:   true,
			errorContains: "pattern field is required",
		},
		{
			name: "invalid regex pattern",
			yamlContent: `rules:
  - pattern: "[invalid"
    response: "Invalid regex"`,
			expectError:   true,
			errorContains: "invalid regex pattern",
		},
		{
			name: "use_claude without prompt",
			yamlContent: `rules:
  - pattern: "test.*"
    use_claude: true`,
			expectError:   true,
			errorContains: "prompt field is required when use_claude is true",
		},
		{
			name: "rule with no response mechanisms",
			yamlContent: `rules:
  - pattern: "test.*"`,
			expectError:   true,
			errorContains: "must provide either a response, alternatives, or use_claude configuration",
		},
		{
			name: "multiple rules with one invalid",
			yamlContent: `rules:
  - pattern: "valid.*"
    response: "Valid rule"
  - pattern: "[invalid"
    response: "Invalid rule"`,
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

// Test logging configuration
func TestConfigWithLogging(t *testing.T) {
	t.Parallel()

	yamlContent := `logging:
  level: debug
  path: /custom/log/path.log
  max_size: 5
  max_backups: 3
  max_age: 7
rules:
  - pattern: "go test.*"
    response: "Use make test instead"`

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

	// Check logging configuration
	if config.Logging.Level != "debug" {
		t.Errorf("Expected logging level 'debug', got %s", config.Logging.Level)
	}

	if config.Logging.Path != "/custom/log/path.log" {
		t.Errorf("Expected logging path '/custom/log/path.log', got %s", config.Logging.Path)
	}

	if config.Logging.MaxSize != 5 {
		t.Errorf("Expected max_size 5, got %d", config.Logging.MaxSize)
	}

	if config.Logging.MaxBackups != 3 {
		t.Errorf("Expected max_backups 3, got %d", config.Logging.MaxBackups)
	}

	if config.Logging.MaxAge != 7 {
		t.Errorf("Expected max_age 7, got %d", config.Logging.MaxAge)
	}
}

// Helper function for substring checking
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// checkRulePattern validates a specific rule pattern and response
func checkRulePattern(t *testing.T, rule *Rule, patternName, expectedPattern, expectedResponse string) bool {
	if strings.Contains(rule.Pattern, expectedPattern) {
		if !strings.Contains(rule.Response, expectedResponse) {
			t.Errorf("Expected %s rule to mention %s", patternName, expectedResponse)
		}
		return true
	}
	return false
}

// validateLoggingDefaults checks the default logging configuration
func validateLoggingDefaults(t *testing.T, config *Config) {
	if config.Logging.Level != "info" {
		t.Errorf("Expected default log level 'info', got %s", config.Logging.Level)
	}
	if config.Logging.MaxSize != 10 {
		t.Errorf("Expected default max size 10, got %d", config.Logging.MaxSize)
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
		if checkRulePattern(t, &rule, "go test", "go test", "make test") {
			foundGoTest = true
		}
		if checkRulePattern(t, &rule, "lint", "gci|go vet", "make lint-fix") {
			foundLint = true
		}
		if strings.Contains(rule.Pattern, "cd /tmp") {
			foundTmp = true
			if !strings.Contains(rule.Response, "tmp") {
				t.Errorf("Expected tmp rule to mention tmp, got: %s", rule.Response)
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

	validateLoggingDefaults(t, config)

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
		"pattern: ^go test",
		"make test",
		"gci|go vet",
		"make lint-fix",
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
  - pattern: "go test.*"
    response: "Use make test instead"
commands:
  - message: "Available commands:\\n!help - Show this help\\n!status - Show project status"
  - message: "Project Status: All systems operational"`

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
		if cmd.Message != expectedMessages[i] {
			t.Errorf("Expected command %d message %q, got %q", i, expectedMessages[i], cmd.Message)
		}
	}
}

// Test validation allows empty rules if commands are present
func TestConfigValidationWithCommands(t *testing.T) {
	t.Parallel()

	yamlContent := `commands:
  - message: "Help command response"`

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

func TestConfigWithNotes(t *testing.T) {
	t.Parallel()

	yamlContent := `rules:
  - pattern: "go test"
    response: "Use just test instead"
notes:
  - message: "Remember to run tests first"
  - message: "Check CLAUDE.md for project conventions"`

	config, err := LoadFromYAML([]byte(yamlContent))
	if err != nil {
		t.Fatalf("Expected no error loading config with notes, got %v", err)
	}

	if len(config.Notes) != 2 {
		t.Fatalf("Expected 2 notes, got %d", len(config.Notes))
	}

	expectedMessages := []string{
		"Remember to run tests first",
		"Check CLAUDE.md for project conventions",
	}

	for i, note := range config.Notes {
		if note.Message != expectedMessages[i] {
			t.Errorf("Expected note %d message %q, got %q", i, expectedMessages[i], note.Message)
		}
	}
}

func TestConfigValidationWithNotesOnly(t *testing.T) {
	t.Parallel()

	yamlContent := `notes:
  - message: "Just a note"`

	config, err := LoadFromYAML([]byte(yamlContent))
	if err != nil {
		t.Fatalf("Expected no error loading config with notes only, got %v", err)
	}

	if len(config.Notes) != 1 {
		t.Fatalf("Expected 1 note, got %d", len(config.Notes))
	}

	if len(config.Rules) != 0 {
		t.Fatalf("Expected 0 rules, got %d", len(config.Rules))
	}
}

func TestDefaultConfigIncludesNotes(t *testing.T) {
	t.Parallel()

	config := DefaultConfig()

	if len(config.Notes) == 0 {
		t.Error("Expected default config to include example notes")
	}

	// Check that notes contain helpful messages
	hasUsefulNote := false
	for _, note := range config.Notes {
		if note.Message != "" {
			hasUsefulNote = true
			break
		}
	}

	if !hasUsefulNote {
		t.Error("Expected at least one note with non-empty message")
	}
}
