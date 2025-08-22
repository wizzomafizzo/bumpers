package template

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wizzomafizzo/bumpers/internal/filesystem"
)

func TestReadFile_TextFile_ReturnsContent(t *testing.T) {
	t.Parallel()
	// Use the real filesystem for simplicity
	fs := filesystem.NewOSFileSystem()
	testContent := "Hello, World!"

	// Create a temporary test file in the project root
	err := os.WriteFile("../../test.txt", []byte(testContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	defer func() {
		_ = os.Remove("../../test.txt")
	}()

	// Test reading the file through template function
	result := readFile(fs, "test.txt")

	if result != testContent {
		t.Errorf("Expected %q, got %q", testContent, result)
	}
}

func TestReadFile_FileNotFound_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	fs := filesystem.NewMemoryFileSystem()

	// Test reading a file that doesn't exist
	result := readFile(fs, "nonexistent.txt")
	if result != "" {
		t.Errorf("Expected empty string for nonexistent file, got %q", result)
	}
}

func TestReadFile_BinaryFile_ReturnsBase64(t *testing.T) {
	t.Parallel()
	fs := filesystem.NewOSFileSystem()

	// Create a binary file (invalid UTF-8) in project root
	binaryContent := []byte{0xFF, 0xFE, 0x00, 0x01, 0x80, 0x90}
	err := os.WriteFile("../../binary.dat", binaryContent, 0o600)
	if err != nil {
		t.Fatalf("Failed to write binary test file: %v", err)
	}
	defer func() {
		_ = os.Remove("../../binary.dat")
	}()

	// Test reading the binary file
	result := readFile(fs, "binary.dat")

	// Should return base64 encoded content with data URI prefix
	expected := "data:application/octet-stream;base64,//4AAYCQ"
	if result != expected {
		t.Errorf("Expected base64 encoded content %q, got %q", expected, result)
	}
}

//nolint:paralleltest // changes working directory
func TestReadFile_DirectoryTraversal_WithRealFS_ReturnsEmpty(t *testing.T) {
	// This test uses the real filesystem to test security
	fs := filesystem.NewOSFileSystem()

	// Create a temporary directory to simulate project root
	tempDir, err := os.MkdirTemp("", "bumpers-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	// Create a test file inside the temp dir
	testFile := filepath.Join(tempDir, "safe.txt")
	err = os.WriteFile(testFile, []byte("safe content"), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a file outside the temp dir that should not be accessible
	outsideFile := filepath.Join(filepath.Dir(tempDir), "outside.txt")
	err = os.WriteFile(outsideFile, []byte("secret content"), 0o600)
	if err != nil {
		t.Fatalf("Failed to create outside file: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(outsideFile)
	}()

	// Change to the temp directory to simulate project context
	oldDir, _ := os.Getwd()
	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldDir)
	}()

	// Test directory traversal attempt
	result := readFile(fs, "../outside.txt")
	if result != "" {
		t.Errorf("Directory traversal should return empty string, got %q", result)
	}
}
