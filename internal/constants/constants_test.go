package constants

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDatabaseFilename(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "bumpers.db", DatabaseFilename)
}

func TestCommandPrefix(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "$", CommandPrefix)
}

func TestClaudeDir(t *testing.T) {
	t.Parallel()
	assert.Equal(t, ".claude", ClaudeDir)
}

func TestAppSubDir(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "bumpers", AppSubDir)
}

func TestLogFilename(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "bumpers.log", LogFilename)
}

func TestSettingsFilename(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "settings.local.json", SettingsFilename)
}
