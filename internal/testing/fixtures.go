package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// LoadTestdataFile loads a file from the testdata directory
func LoadTestdataFile(t *testing.T, relativePath string) []byte {
	t.Helper()

	// Find project root by looking for go.mod
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	projectRoot := wd
	for {
		if _, statErr := os.Stat(filepath.Join(projectRoot, "go.mod")); statErr == nil {
			break
		}
		parent := filepath.Dir(projectRoot)
		if parent == projectRoot {
			t.Fatal("Could not find project root (go.mod)")
		}
		projectRoot = parent
	}

	fullPath := filepath.Join(projectRoot, "testdata", relativePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("Failed to load testdata file %s: %v", relativePath, err)
	}

	return content
}

// LoadTestdataString loads a file from testdata as a string
func LoadTestdataString(t *testing.T, relativePath string) string {
	t.Helper()
	return string(LoadTestdataFile(t, relativePath))
}

// GetTestdataPath returns the full path to a testdata file
func GetTestdataPath(t *testing.T, relativePath string) string {
	t.Helper()

	// Find project root by looking for go.mod
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	projectRoot := wd
	for {
		if _, statErr := os.Stat(filepath.Join(projectRoot, "go.mod")); statErr == nil {
			break
		}
		parent := filepath.Dir(projectRoot)
		if parent == projectRoot {
			t.Fatal("Could not find project root (go.mod)")
		}
		projectRoot = parent
	}

	return filepath.Join(projectRoot, "testdata", relativePath)
}

// WriteTestdataFile writes content to a testdata file (useful for updating golden files)
func WriteTestdataFile(t *testing.T, relativePath string, content []byte) {
	t.Helper()

	fullPath := GetTestdataPath(t, relativePath)
	dir := filepath.Dir(fullPath)

	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("Failed to create testdata directory %s: %v", dir, err)
	}

	if err := os.WriteFile(fullPath, content, 0o600); err != nil {
		t.Fatalf("Failed to write testdata file %s: %v", relativePath, err)
	}
}
