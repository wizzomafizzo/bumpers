//go:build integration

package filesystem

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/wizzomafizzo/bumpers/internal/testing"
)

// Test the basic contract we need from our filesystem abstraction
func TestMemoryFileSystemBasicOperations(t *testing.T) {
	testutil.InitTestLogger(t)
	t.Parallel()

	fs := NewMemoryFileSystem()

	// Test WriteFile and ReadFile - the core operations needed
	testContent := []byte("test content")
	err := fs.WriteFile("test.txt", testContent, 0o644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	content, err := fs.ReadFile("test.txt")
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if !bytes.Equal(content, testContent) {
		t.Errorf("Expected content %q, got %q", string(testContent), string(content))
	}
}

// Test OSFileSystem - production filesystem implementation
func TestOSFileSystemBasicOperations(t *testing.T) {
	t.Parallel()

	fs := NewOSFileSystem()
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")

	// Test WriteFile and ReadFile - matching MemoryFileSystem behavior
	testContent := []byte("test content")
	err := fs.WriteFile(testFile, testContent, 0o644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	content, err := fs.ReadFile(testFile)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if !bytes.Equal(content, testContent) {
		t.Errorf("Expected content %q, got %q", string(testContent), string(content))
	}

	// Test Stat - needed by production code
	info, err := fs.Stat(testFile)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	if info.Size() != int64(len(testContent)) {
		t.Errorf("Expected file size %d, got %d", len(testContent), info.Size())
	}

	// Test non-existent file
	_, err = fs.ReadFile(filepath.Join(tempDir, "nonexistent.txt"))
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
	if !os.IsNotExist(err) {
		t.Errorf("Expected os.ErrNotExist, got %v", err)
	}
}

// Test MkdirAll functionality for both implementations
func TestFileSystemMkdirAll(t *testing.T) {
	testutil.InitTestLogger(t)
	t.Parallel()

	t.Run("MemoryFileSystem", func(t *testing.T) {
		t.Parallel()
		testMemoryFileSystemMkdirAll(t)
	})

	t.Run("OSFileSystem", func(t *testing.T) {
		t.Parallel()
		testOSFileSystemMkdirAll(t)
	})
}

func testMemoryFileSystemMkdirAll(t *testing.T) {
	t.Helper()
	fs := NewMemoryFileSystem()

	err := fs.MkdirAll("path/to/nested/dir", 0o750)
	if err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	testFile := "path/to/nested/dir/test.txt"
	testContent := []byte("test in nested dir")
	err = fs.WriteFile(testFile, testContent, 0o644)
	if err != nil {
		t.Fatalf("WriteFile in created directory failed: %v", err)
	}

	content, err := fs.ReadFile(testFile)
	if err != nil {
		t.Fatalf("ReadFile from created directory failed: %v", err)
	}

	if !bytes.Equal(content, testContent) {
		t.Errorf("Expected content %q, got %q", string(testContent), string(content))
	}
}

func testOSFileSystemMkdirAll(t *testing.T) {
	t.Helper()
	fs := NewOSFileSystem()
	tempDir := t.TempDir()

	nestedPath := filepath.Join(tempDir, "path", "to", "nested", "dir")
	err := fs.MkdirAll(nestedPath, 0o750)
	if err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	info, err := fs.Stat(nestedPath)
	if err != nil {
		t.Fatalf("Stat on created directory failed: %v", err)
	}

	if !info.IsDir() {
		t.Error("Expected created path to be a directory")
	}

	testFile := filepath.Join(nestedPath, "test.txt")
	testContent := []byte("test in nested dir")
	err = fs.WriteFile(testFile, testContent, 0o644)
	if err != nil {
		t.Fatalf("WriteFile in created directory failed: %v", err)
	}

	content, err := fs.ReadFile(testFile)
	if err != nil {
		t.Fatalf("ReadFile from created directory failed: %v", err)
	}

	if !bytes.Equal(content, testContent) {
		t.Errorf("Expected content %q, got %q", string(testContent), string(content))
	}
}

// TestFileSystemContract runs contract tests against all implementations to ensure consistent behavior
func TestFileSystemContract(t *testing.T) {
	testutil.InitTestLogger(t)
	t.Parallel()

	t.Run("MemoryFileSystem", func(t *testing.T) {
		t.Parallel()
		fs := NewMemoryFileSystem()

		// Test WriteFile and ReadFile contract - basic operations should work identically
		testContent := []byte("contract test content")

		err := fs.WriteFile("contract_test.txt", testContent, 0o644)
		if err != nil {
			t.Fatalf("WriteFile should succeed, got: %v", err)
		}

		content, err := fs.ReadFile("contract_test.txt")
		if err != nil {
			t.Fatalf("ReadFile should succeed after WriteFile, got: %v", err)
		}

		if !bytes.Equal(content, testContent) {
			t.Errorf("ReadFile content mismatch: got %q, want %q", string(content), string(testContent))
		}
	})

	t.Run("OSFileSystem", func(t *testing.T) {
		t.Parallel()
		fs := NewOSFileSystem()
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "contract_test.txt")

		// Same contract test as MemoryFileSystem - behavior should be identical
		testContent := []byte("contract test content")

		err := fs.WriteFile(testFile, testContent, 0o644)
		if err != nil {
			t.Fatalf("WriteFile should succeed, got: %v", err)
		}

		content, err := fs.ReadFile(testFile)
		if err != nil {
			t.Fatalf("ReadFile should succeed after WriteFile, got: %v", err)
		}

		if !bytes.Equal(content, testContent) {
			t.Errorf("ReadFile content mismatch: got %q, want %q", string(content), string(testContent))
		}
	})
}
