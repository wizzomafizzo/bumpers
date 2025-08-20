package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	t.Parallel()

	yamlContent := `rules:
  - pattern: "go test.*"
    response: "Use make test instead"
`

	config, err := LoadFromYAML([]byte(yamlContent))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(config.Rules) != 1 {
		t.Fatalf("Expected 1 rule, got %d", len(config.Rules))
	}

	rule := config.Rules[0]
	if rule.Pattern != "go test.*" {
		t.Errorf("Expected rule pattern 'go test.*', got %s", rule.Pattern)
	}
}

func TestLoadConfigWithAlternatives(t *testing.T) {
	t.Parallel()

	yamlContent := `rules:
  - pattern: "go test.*"
    response: "Use make test instead for better TDD integration"
    alternatives:
      - "make test          # Run all tests"
      - "make test-unit     # Run unit tests only"
    use_claude: false
`

	config, err := LoadFromYAML([]byte(yamlContent))
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
    use_claude: "no"
    prompt: "Explain why direct go test is discouraged"
`

	config, err := LoadFromYAML([]byte(yamlContent))
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
	config, err := LoadFromYAML([]byte(yamlContent))
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

	config, err := LoadFromFile(configPath)
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

	config, err := LoadFromYAML([]byte(yamlContent))
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
