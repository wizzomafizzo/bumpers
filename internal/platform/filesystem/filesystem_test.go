package filesystem

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wizzomafizzo/bumpers/internal/testing"
)

func TestNewMemoryFileSystem(t *testing.T) {
	testutil.InitTestLogger(t)
	t.Parallel()

	fs := NewMemoryFileSystem()

	assert.NotNil(t, fs, "NewMemoryFileSystem should return non-nil filesystem")
	assert.NotNil(t, fs.files, "MemoryFileSystem should have initialized files map")
	assert.Empty(t, fs.files, "New MemoryFileSystem should start with empty files map")
}

func TestMemoryFileSystem_WriteFile_ReadFile(t *testing.T) {
	testutil.InitTestLogger(t)
	t.Parallel()

	fs := NewMemoryFileSystem()
	testData := []byte("test file content")
	filename := "test.txt"

	// Write file
	err := fs.WriteFile(filename, testData, 0o644)
	require.NoError(t, err, "WriteFile should succeed")

	// Verify data was stored
	assert.Contains(t, fs.files, filename, "File should be stored in files map")

	// Read file back
	readData, err := fs.ReadFile(filename)
	require.NoError(t, err, "ReadFile should succeed")

	// Verify content matches
	assert.Equal(t, testData, readData, "Read data should match written data")

	// Verify data is independent (deep copy)
	testData[0] = 'X'
	readData2, err := fs.ReadFile(filename)
	require.NoError(t, err, "Second ReadFile should succeed")
	assert.NotEqual(t, testData[0], readData2[0], "Stored data should not be affected by modifying original")
}

func TestMemoryFileSystem_ReadFile_NonExistent(t *testing.T) {
	testutil.InitTestLogger(t)
	t.Parallel()

	fs := NewMemoryFileSystem()

	// Try to read non-existent file
	data, err := fs.ReadFile("non-existent.txt")

	assert.Nil(t, data, "ReadFile should return nil data for non-existent file")
	assert.ErrorIs(t, err, os.ErrNotExist, "ReadFile should return os.ErrNotExist for non-existent file")
}

func TestMemoryFileSystem_Stat(t *testing.T) {
	testutil.InitTestLogger(t)
	t.Parallel()

	fs := NewMemoryFileSystem()
	testData := []byte("test content for stat")
	filename := "stat-test.txt"

	// Write file first
	err := fs.WriteFile(filename, testData, 0o644)
	require.NoError(t, err, "WriteFile should succeed")

	// Get file info
	info, err := fs.Stat(filename)
	require.NoError(t, err, "Stat should succeed for existing file")

	assert.Equal(t, filename, info.Name(), "Stat should return correct filename")
	assert.Equal(t, int64(len(testData)), info.Size(), "Stat should return correct file size")
	assert.False(t, info.IsDir(), "File should not be reported as directory")

	// Test non-existent file
	_, err = fs.Stat("non-existent.txt")
	assert.ErrorIs(t, err, os.ErrNotExist, "Stat should return os.ErrNotExist for non-existent file")
}

func TestMemoryFileSystem_MkdirAll(t *testing.T) {
	testutil.InitTestLogger(t)
	t.Parallel()

	fs := NewMemoryFileSystem()

	// MkdirAll should always succeed for MemoryFileSystem (no-op)
	err := fs.MkdirAll("/some/deep/path", 0o755)
	require.NoError(t, err, "MkdirAll should succeed for MemoryFileSystem")

	// Should work with various paths
	err = fs.MkdirAll("", 0o755)
	require.NoError(t, err, "MkdirAll should handle empty path")

	err = fs.MkdirAll("single-dir", 0o755)
	assert.NoError(t, err, "MkdirAll should handle single directory")
}

func TestNewOSFileSystem(t *testing.T) {
	testutil.InitTestLogger(t)
	t.Parallel()

	fs := NewOSFileSystem()

	assert.NotNil(t, fs, "NewOSFileSystem should return non-nil filesystem")
}
