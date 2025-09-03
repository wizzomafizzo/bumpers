package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProcessBuiltinCommand_InvalidDatabasePath(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Use an invalid database path that will cause connection failure
	invalidDBPath := "/invalid/path/that/does/not/exist/test.db"

	response, err := ProcessBuiltinCommand(ctx, "bumpers status", invalidDBPath, "test-project")

	require.Error(t, err)
	require.Nil(t, response)
	require.Contains(t, err.Error(), "failed to create database manager")
}

func TestProcessBuiltinCommand_EmptyProjectID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tempDir := t.TempDir()
	dbPath := tempDir + "/test.db"

	// Test with empty project ID - should not cause an error (valid use case)
	response, err := ProcessBuiltinCommand(ctx, "bumpers status", dbPath, "")

	require.NoError(t, err)
	require.NotNil(t, response)
	require.Contains(t, response.(string), "Rules are currently")
}
