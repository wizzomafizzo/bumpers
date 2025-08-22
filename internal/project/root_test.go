package project

import (
	"os"
	"path/filepath"
	"testing"
)

//nolint:paralleltest // changes working directory and environment variables
func TestFindRoot_FallbackToCwd(t *testing.T) {
	// Unset CLAUDE_PROJECT_DIR
	originalEnv := os.Getenv("CLAUDE_PROJECT_DIR")
	defer func() {
		if originalEnv == "" {
			if err := os.Unsetenv("CLAUDE_PROJECT_DIR"); err != nil {
				t.Errorf("Failed to unset CLAUDE_PROJECT_DIR: %v", err)
			}
		} else {
			if err := os.Setenv("CLAUDE_PROJECT_DIR", originalEnv); err != nil {
				t.Errorf("Failed to restore CLAUDE_PROJECT_DIR: %v", err)
			}
		}
	}()
	if err := os.Unsetenv("CLAUDE_PROJECT_DIR"); err != nil {
		t.Fatalf("Failed to unset CLAUDE_PROJECT_DIR: %v", err)
	}

	// Create a directory with no project markers
	tempDir := t.TempDir()
	originalCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		if cdErr := os.Chdir(originalCwd); cdErr != nil {
			t.Errorf("Failed to restore original directory: %v", cdErr)
		}
	}()

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Test that FindRoot falls back to current directory
	root, err := FindRoot()
	if err != nil {
		t.Fatalf("FindRoot failed: %v", err)
	}

	expected, _ := filepath.Abs(tempDir)
	actual, _ := filepath.Abs(root)

	if actual != expected {
		t.Errorf("Expected root %s, got %s", expected, actual)
	}
}

// setupTestProjectWithMarker creates a test project with a specific marker file
func setupTestProjectWithMarker(t *testing.T, markerName, markerContent string) (projectDir string, cleanup func()) {
	t.Helper()

	// Unset CLAUDE_PROJECT_DIR to test marker detection
	originalEnv := os.Getenv("CLAUDE_PROJECT_DIR")
	if err := os.Unsetenv("CLAUDE_PROJECT_DIR"); err != nil {
		t.Fatalf("Failed to unset CLAUDE_PROJECT_DIR: %v", err)
	}

	// Create a temporary directory structure
	tempDir := t.TempDir()
	projectDir = filepath.Join(tempDir, "my-project")
	subDir := filepath.Join(projectDir, "cmd", "myapp")

	err := os.MkdirAll(subDir, 0o750)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create marker file to mark project root
	markerPath := filepath.Join(projectDir, markerName)
	err = os.WriteFile(markerPath, []byte(markerContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create marker file: %v", err)
	}

	// Setup cleanup function
	originalCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	cleanup = func() {
		if cdErr := os.Chdir(originalCwd); cdErr != nil {
			t.Errorf("Failed to restore original directory: %v", cdErr)
		}
		// Restore environment variable
		if originalEnv == "" {
			if unsetErr := os.Unsetenv("CLAUDE_PROJECT_DIR"); unsetErr != nil {
				t.Errorf("Failed to unset CLAUDE_PROJECT_DIR: %v", unsetErr)
			}
		} else {
			if setErr := os.Setenv("CLAUDE_PROJECT_DIR", originalEnv); setErr != nil {
				t.Errorf("Failed to restore CLAUDE_PROJECT_DIR: %v", setErr)
			}
		}
	}

	// Change to subdirectory for testing
	err = os.Chdir(subDir)
	if err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	return projectDir, cleanup
}

//nolint:paralleltest // changes working directory via setupTestProjectWithMarker
func TestFindRoot_WithGoMod(t *testing.T) {
	projectDir, cleanup := setupTestProjectWithMarker(t, "go.mod", "module example.com/myproject\n")
	defer cleanup()

	// Test that FindRoot finds the project root from subdirectory
	root, err := FindRoot()
	if err != nil {
		t.Fatalf("FindRoot failed: %v", err)
	}

	expected, err := filepath.Abs(projectDir)
	if err != nil {
		t.Fatalf("Failed to get absolute path for expected: %v", err)
	}
	actual, err := filepath.Abs(root)
	if err != nil {
		t.Fatalf("Failed to get absolute path for actual: %v", err)
	}

	if actual != expected {
		t.Errorf("Expected root %s, got %s", expected, actual)
	}
}

//nolint:paralleltest // changes working directory via setupTestProjectWithMarker
func TestFindRoot_WithPackageJSON(t *testing.T) {
	projectDir, cleanup := setupTestProjectWithMarker(t, "package.json", `{"name": "my-project", "version": "1.0.0"}`)
	defer cleanup()

	// Test that FindRoot finds the project root from subdirectory
	root, err := FindRoot()
	if err != nil {
		t.Fatalf("FindRoot failed: %v", err)
	}

	expected, err := filepath.Abs(projectDir)
	if err != nil {
		t.Fatalf("Failed to get absolute path for expected: %v", err)
	}
	actual, err := filepath.Abs(root)
	if err != nil {
		t.Fatalf("Failed to get absolute path for actual: %v", err)
	}

	if actual != expected {
		t.Errorf("Expected root %s, got %s", expected, actual)
	}
}
