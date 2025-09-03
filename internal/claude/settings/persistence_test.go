package settings

import (
	"encoding/json"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSettingsFilename = "settings.json"

func TestLoadFromFileWithFS_Success(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	filename := testSettingsFilename

	// Create test settings JSON
	testSettings := &Settings{
		OutputStyle: "Explanatory",
		Model:       "claude-3-5-sonnet-20241022",
	}
	data, err := json.MarshalIndent(testSettings, "", "  ")
	require.NoError(t, err)

	// Write to memory filesystem
	err = afero.WriteFile(fs, filename, data, 0o600)
	require.NoError(t, err)

	// Test loading
	result, err := LoadFromFileWithFS(fs, filename)
	require.NoError(t, err)
	assert.Equal(t, "Explanatory", result.OutputStyle)
	assert.Equal(t, "claude-3-5-sonnet-20241022", result.Model)
}

func TestLoadFromFileWithFS_FileNotFound(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	filename := "nonexistent.json"

	// Test loading non-existent file
	result, err := LoadFromFileWithFS(fs, filename)
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read settings file")
}

func TestLoadFromFileWithFS_InvalidJSON(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	filename := "invalid.json"

	// Write invalid JSON
	err := afero.WriteFile(fs, filename, []byte("invalid json content"), 0o600)
	require.NoError(t, err)

	// Test loading invalid JSON
	result, err := LoadFromFileWithFS(fs, filename)
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse settings JSON")
}

func TestSaveToFileWithFS_Success(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	filename := testSettingsFilename

	settings := &Settings{
		OutputStyle: "Concise",
		Model:       "claude-3-5-sonnet-20241022",
	}

	// Test saving
	err := SaveToFileWithFS(fs, settings, filename)
	require.NoError(t, err)

	// Verify file was created and has correct content
	exists, err := afero.Exists(fs, filename)
	require.NoError(t, err)
	assert.True(t, exists)

	// Read back and verify
	result, err := LoadFromFileWithFS(fs, filename)
	require.NoError(t, err)
	assert.Equal(t, settings.OutputStyle, result.OutputStyle)
	assert.Equal(t, settings.Model, result.Model)
}

func TestCreateBackupWithFS_Success(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	filename := testSettingsFilename

	// Create original file
	originalContent := []byte(`{"outputStyle": "Explanatory"}`)
	err := afero.WriteFile(fs, filename, originalContent, 0o600)
	require.NoError(t, err)

	// Test creating backup
	backupPath, err := CreateBackupWithFS(fs, filename)
	require.NoError(t, err)
	assert.Equal(t, filename+".bak", backupPath)

	// Verify backup file exists and has same content
	backupContent, err := afero.ReadFile(fs, backupPath)
	require.NoError(t, err)
	assert.Equal(t, originalContent, backupContent)
}

func TestHasBackupWithFS(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	filename := testSettingsFilename

	// Initially no backup should exist
	assert.False(t, HasBackupWithFS(fs, filename))

	// Create backup file
	backupPath := GetBackupPath(filename)
	err := afero.WriteFile(fs, backupPath, []byte("test"), 0o600)
	require.NoError(t, err)

	// Now backup should exist
	assert.True(t, HasBackupWithFS(fs, filename))
}

func TestRestoreFromBackupWithFS(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	backupPath := "settings.json.bak"
	targetPath := "settings.json"

	backupContent := []byte(`{"outputStyle": "Original"}`)

	// Create backup file
	err := afero.WriteFile(fs, backupPath, backupContent, 0o600)
	require.NoError(t, err)

	// Create different target file
	err = afero.WriteFile(fs, targetPath, []byte(`{"outputStyle": "Modified"}`), 0o600)
	require.NoError(t, err)

	// Restore from backup
	err = RestoreFromBackupWithFS(fs, backupPath, targetPath)
	require.NoError(t, err)

	// Verify target file has backup content
	targetContent, err := afero.ReadFile(fs, targetPath)
	require.NoError(t, err)
	assert.Equal(t, backupContent, targetContent)
}
