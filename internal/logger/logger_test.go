package logger

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wizzomafizzo/bumpers/internal/testutil"
)

func TestEnsureLogDir_Success(t *testing.T) {
	t.Parallel()
	testutil.InitTestLogger(t)

	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "logs")

	err := ensureLogDir(logDir)

	require.NoError(t, err)

	// Verify directory was created
	info, err := os.Stat(logDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestCreateLumberjackLogger_Configuration(t *testing.T) {
	t.Parallel()
	testutil.InitTestLogger(t)

	logFile := "/tmp/test.log"

	lj := createLumberjackLogger(logFile)

	assert.Equal(t, logFile, lj.Filename)
	assert.Equal(t, maxLogSizeMB, lj.MaxSize)
	assert.Equal(t, maxLogBackups, lj.MaxBackups)
	assert.Equal(t, maxLogAgeDays, lj.MaxAge)
}
