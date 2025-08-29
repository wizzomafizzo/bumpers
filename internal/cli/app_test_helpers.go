package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	testutil "github.com/wizzomafizzo/bumpers/internal/testing"
)

// setupTestWithContext creates a context with logger for race-safe testing
func setupTestWithContext(t *testing.T) (ctx context.Context, getLogOutput func() string) {
	t.Helper()
	return testutil.NewTestContext(t)
}

// createTempConfig creates a temporary config file with the given content
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

// setupProjectStructure creates a temporary project structure for testing
func setupProjectStructure(t *testing.T, configFileName string) (projectDir, subDir string, cleanup func()) {
	t.Helper()

	tempDir := t.TempDir()
	projectDir = filepath.Join(tempDir, "my-project")
	subDir = filepath.Join(projectDir, "internal", "cli")

	err := os.MkdirAll(subDir, 0o750)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create go.mod file to mark project root
	goModPath := filepath.Join(projectDir, "go.mod")
	err = os.WriteFile(goModPath, []byte("module example.com/myproject\n"), 0o600)
	if err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create configuration file if specified
	if configFileName != "" {
		configPath := filepath.Join(projectDir, configFileName)
		configContent := `rules:
  - pattern: "echo *"
    action: allow
    reason: "Safe command for testing"
`
		err = os.WriteFile(configPath, []byte(configContent), 0o600)
		if err != nil {
			t.Fatalf("Failed to create config file: %v", err)
		}
	}

	cleanup = func() {
		// TempDir handles cleanup automatically
	}

	return projectDir, subDir, cleanup
}

// validateTestEnvironment checks that Initialize succeeds in test environment
func validateTestEnvironment(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("Expected Initialize to succeed in test environment, but got error: %v", err)
	}
}

// validateProductionEnvironment checks that Initialize fails appropriately in production
func validateProductionEnvironment(t *testing.T, err error, prodLikeDir string) {
	t.Helper()
	if err == nil {
		t.Error("Expected Initialize to fail when bumpers binary is missing in production environment")
		return
	}

	// Should contain helpful error message
	expectedPath := filepath.Join(prodLikeDir, "bin", "bumpers")
	if !strings.Contains(err.Error(), expectedPath) {
		t.Errorf("Error should mention the expected path %s, got: %v", expectedPath, err)
	}

	if !strings.Contains(err.Error(), "make build") {
		t.Errorf("Error should suggest running 'make build', got: %v", err)
	}
}
