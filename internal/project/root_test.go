package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wizzomafizzo/bumpers/internal/testutil"
)

func TestCheckClaudeProjectDir_EmptyEnv(t *testing.T) {
	testutil.InitTestLogger(t)

	// Ensure CLAUDE_PROJECT_DIR is not set
	t.Setenv("CLAUDE_PROJECT_DIR", "")

	path, found := checkClaudeProjectDir()

	assert.False(t, found)
	assert.Empty(t, path)
}

func TestCheckClaudeProjectDir_ValidDirectory(t *testing.T) {
	testutil.InitTestLogger(t)

	tempDir := t.TempDir()
	t.Setenv("CLAUDE_PROJECT_DIR", tempDir)

	path, found := checkClaudeProjectDir()

	assert.True(t, found)
	assert.Contains(t, path, tempDir) // Use Contains since path might be absolute
}

func TestCheckClaudeProjectDir_NonexistentDirectory(t *testing.T) {
	testutil.InitTestLogger(t)

	t.Setenv("CLAUDE_PROJECT_DIR", "/nonexistent/path/that/does/not/exist")

	path, found := checkClaudeProjectDir()

	assert.False(t, found)
	assert.Empty(t, path)
}

func TestHasProjectMarker_WithGitMarker(t *testing.T) {
	t.Parallel()
	testutil.InitTestLogger(t)

	tempDir := t.TempDir()

	// Create .git file
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, ".git"), []byte("gitdir: .git"), 0o600))

	result := hasProjectMarker(tempDir, []string{".git"})

	assert.True(t, result)
}

func TestHasProjectMarker_NoMarkers(t *testing.T) {
	t.Parallel()
	testutil.InitTestLogger(t)

	tempDir := t.TempDir()

	result := hasProjectMarker(tempDir, []string{".git", "go.mod", "package.json"})

	assert.False(t, result)
}
