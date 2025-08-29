package template

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/wizzomafizzo/bumpers/internal/infrastructure/project"
)

func TestReadFile_TextFile_ReturnsContent(t *testing.T) {
	t.Parallel()
	// Use the real filesystem for simplicity
	fs := afero.NewOsFs()
	testContent := "Hello, World!"

	// Get project root and create test file there
	projectRoot, err := project.FindRoot()
	if err != nil {
		t.Fatalf("Failed to find project root: %v", err)
	}
	testFile := filepath.Join(projectRoot, "test.txt")

	err = os.WriteFile(testFile, []byte(testContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Remove(testFile)
	})

	// Test reading the file through template function
	result := readFile(fs, "test.txt")

	if result != testContent {
		t.Errorf("Expected %q, got %q", testContent, result)
	}
}

func TestReadFile_FileNotFound_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()

	// Test reading a file that doesn't exist
	result := readFile(fs, "nonexistent.txt")
	if result != "" {
		t.Errorf("Expected empty string for nonexistent file, got %q", result)
	}
}

func TestReadFile_BinaryFile_ReturnsBase64(t *testing.T) {
	t.Parallel()
	fs := afero.NewOsFs()

	// Get project root and create binary file there
	projectRoot, err := project.FindRoot()
	if err != nil {
		t.Fatalf("Failed to find project root: %v", err)
	}

	// Create a binary file (invalid UTF-8) in project root
	binaryContent := []byte{0xFF, 0xFE, 0x00, 0x01, 0x80, 0x90}
	binaryFile := filepath.Join(projectRoot, "binary.dat")
	err = os.WriteFile(binaryFile, binaryContent, 0o600)
	if err != nil {
		t.Fatalf("Failed to write binary test file: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Remove(binaryFile)
	})

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
	fs := afero.NewOsFs()

	// Create a temporary directory to simulate project root
	tempDir, err := os.MkdirTemp("", "bumpers-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(tempDir)
	})

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
	t.Cleanup(func() {
		_ = os.RemoveAll(outsideFile)
	})

	// Change to the temp directory to simulate project context
	oldDir, _ := os.Getwd()
	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldDir)
	})

	// Test directory traversal attempt
	result := readFile(fs, "../outside.txt")
	if result != "" {
		t.Errorf("Directory traversal should return empty string, got %q", result)
	}
}

func setupMemoryFS(_ *testing.T) (fs afero.Fs, cleanup func()) {
	return afero.NewMemMapFs(), func() {}
}

func setupFileExistsFS(t *testing.T) (fs afero.Fs, cleanup func()) {
	fs = afero.NewOsFs()
	projectRoot, err := project.FindRoot()
	if err != nil {
		t.Fatalf("Failed to find project root: %v", err)
	}
	testFile := filepath.Join(projectRoot, "test-exists.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0o600)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	cleanup = func() {
		_ = os.Remove(testFile)
	}
	return fs, cleanup
}

func setupDirectoryExistsFS(t *testing.T) (fs afero.Fs, cleanup func()) {
	fs = afero.NewOsFs()
	projectRoot, err := project.FindRoot()
	if err != nil {
		t.Fatalf("Failed to find project root: %v", err)
	}
	testDir := filepath.Join(projectRoot, "test-dir")
	err = os.Mkdir(testDir, 0o750)
	if err != nil && !os.IsExist(err) {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	cleanup = func() {
		_ = os.Remove(testDir)
	}
	return fs, cleanup
}

func TestTestPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setupFS  func(t *testing.T) (afero.Fs, func())
		path     string
		expected bool
	}{
		{
			name:     "file not found returns false",
			setupFS:  setupMemoryFS,
			path:     "nonexistent.txt",
			expected: false,
		},
		{
			name:     "file exists returns true",
			setupFS:  setupFileExistsFS,
			path:     "test-exists.txt",
			expected: true,
		},
		{
			name:     "directory exists returns true",
			setupFS:  setupDirectoryExistsFS,
			path:     "test-dir",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fs, cleanup := tt.setupFS(t)
			t.Cleanup(cleanup)

			result := testPath(fs, tt.path)
			if result != tt.expected {
				t.Errorf("testPath() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

//nolint:paralleltest // changes working directory
func TestTestPath_DirectoryTraversal_WithRealFS_ReturnsFalse(t *testing.T) {
	// This test uses the real filesystem to test security
	fs := afero.NewOsFs()

	// Create a temporary directory to simulate project root
	tempDir, err := os.MkdirTemp("", "bumpers-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(tempDir)
	})

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
	t.Cleanup(func() {
		_ = os.RemoveAll(outsideFile)
	})

	// Change to the temp directory to simulate project context
	oldDir, _ := os.Getwd()
	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldDir)
	})

	// Test directory traversal attempt
	result := testPath(fs, "../outside.txt")
	if result {
		t.Error("Directory traversal should return false, got true")
	}
}

func TestArgc_WithNoArgs_ReturnsZero(t *testing.T) {
	t.Parallel()

	ctx := &CommandContext{
		Name: "test",
		Args: "",
		Argv: []string{"test"},
	}

	result := argc(ctx)
	if result != 0 {
		t.Errorf("Expected argc to return 0, got %d", result)
	}
}

func TestArgc_WithArgs_ReturnsCorrectCount(t *testing.T) {
	t.Parallel()

	ctx := &CommandContext{
		Name: "test",
		Args: "arg1 arg2 arg3",
		Argv: []string{"test", "arg1", "arg2", "arg3"},
	}

	result := argc(ctx)
	if result != 3 {
		t.Errorf("Expected argc to return 3, got %d", result)
	}
}

func TestArgc_WithNilContext_ReturnsZero(t *testing.T) {
	t.Parallel()

	result := argc(nil)
	if result != 0 {
		t.Errorf("Expected argc to return 0 for nil context, got %d", result)
	}
}

func TestArgv_IndexZero_ReturnsCommandName(t *testing.T) {
	t.Parallel()

	ctx := &CommandContext{
		Name: "test",
		Args: "arg1 arg2",
		Argv: []string{"test", "arg1", "arg2"},
	}

	result := argv(ctx, 0)
	if result != "test" {
		t.Errorf("Expected argv(0) to return 'test', got %q", result)
	}
}

func TestArgv_ValidIndex_ReturnsArgument(t *testing.T) {
	t.Parallel()

	ctx := &CommandContext{
		Name: "test",
		Args: "foo \"bar baz\" qux",
		Argv: []string{"test", "foo", "bar baz", "qux"},
	}

	result := argv(ctx, 1)
	if result != "foo" {
		t.Errorf("Expected argv(1) to return 'foo', got %q", result)
	}
}

func TestArgv_OutOfBoundsIndex_ReturnsEmpty(t *testing.T) {
	t.Parallel()

	ctx := &CommandContext{
		Name: "test",
		Args: "arg1",
		Argv: []string{"test", "arg1"},
	}

	result := argv(ctx, 5)
	if result != "" {
		t.Errorf("Expected argv(5) to return empty string, got %q", result)
	}
}

func TestArgv_WithNilContext_ReturnsEmpty(t *testing.T) {
	t.Parallel()

	result := argv(nil, 0)
	if result != "" {
		t.Errorf("Expected argv to return empty string for nil context, got %q", result)
	}
}
