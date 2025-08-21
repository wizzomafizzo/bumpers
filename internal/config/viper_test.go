package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadViperJSON(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test.json")

	jsonContent := `{
  "rules": [
    {
      "pattern": "go test.*",
      "response": "Use make test instead"
    }
  ]
}`

	err := os.WriteFile(configFile, []byte(jsonContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := Load(configFile)
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
