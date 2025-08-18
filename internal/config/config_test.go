package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	t.Parallel()

	yamlContent := `rules:
  - name: "test-rule"
    pattern: "go test.*"
    action: "deny"
    message: "Use make test instead"
`

	config, err := LoadFromYAML([]byte(yamlContent))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(config.Rules) != 1 {
		t.Fatalf("Expected 1 rule, got %d", len(config.Rules))
	}

	rule := config.Rules[0]
	if rule.Name != "test-rule" {
		t.Errorf("Expected rule name 'test-rule', got %s", rule.Name)
	}
}

func TestLoadConfigWithAlternatives(t *testing.T) {
	t.Parallel()

	yamlContent := `rules:
  - name: "block-go-test"
    pattern: "go test.*"
    action: "deny"
    message: "Use make test instead for better TDD integration"
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

func TestRuleActionConstants(t *testing.T) {
	t.Parallel()

	if ActionAllow != "allow" {
		t.Errorf("Expected ActionAllow to be 'allow', got %s", ActionAllow)
	}

	if ActionDeny != "deny" {
		t.Errorf("Expected ActionDeny to be 'deny', got %s", ActionDeny)
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	t.Parallel()

	// Create temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.yaml")
	configContent := `rules:
  - name: "block-go-test"
    pattern: "go test.*"
    action: "deny"
    message: "Use make test instead for better TDD integration"
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
	if rule.Name != "block-go-test" {
		t.Errorf("Expected first rule name 'block-go-test', got %s", rule.Name)
	}
}
