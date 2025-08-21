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

	testConfigLoading(t, configFile, "go test.*")
}
