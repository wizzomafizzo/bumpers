package cli

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const testProjectID = "test-project"

func TestIsBuiltinCommand_EnableCommand(t *testing.T) {
	t.Parallel()
	result := IsBuiltinCommand("bumpers enable")
	require.True(t, result)
}

func TestIsBuiltinCommand_NonBuiltinCommand(t *testing.T) {
	t.Parallel()
	result := IsBuiltinCommand("ls -la")
	require.False(t, result)
}

func TestProcessBuiltinCommand_Enable(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	response, err := ProcessBuiltinCommand(ctx, "bumpers enable", dbPath, testProjectID)
	require.NoError(t, err)
	require.NotNil(t, response)
}

func TestProcessBuiltinCommand_StatusReturnsString(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	response, err := ProcessBuiltinCommand(ctx, "bumpers status", dbPath, testProjectID)
	require.NoError(t, err)
	require.IsType(t, "", response, "status command should return a string")
}

func TestProcessBuiltinCommand_Skip(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	response, err := ProcessBuiltinCommand(ctx, "bumpers skip", dbPath, testProjectID)
	require.NoError(t, err)
	require.NotNil(t, response)
}

func TestProcessBuiltinCommand_DisableActuallyDisables(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	projectID := testProjectID

	// First, disable rules
	_, err := ProcessBuiltinCommand(ctx, "bumpers disable", dbPath, projectID)
	require.NoError(t, err)

	// Check status - should show disabled
	statusResponse, err := ProcessBuiltinCommand(ctx, "bumpers status", dbPath, projectID)
	require.NoError(t, err)

	statusStr, ok := statusResponse.(string)
	require.True(t, ok, "Expected status response to be string, got %T", statusResponse)

	if statusStr == "Rules are currently enabled" {
		t.Error("Expected status to show rules are disabled after 'bumpers disable'")
	}
}

func TestProcessBuiltinCommand_EnableAfterDisableShowsEnabled(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	projectID := testProjectID

	// First, disable rules
	_, err := ProcessBuiltinCommand(ctx, "bumpers disable", dbPath, projectID)
	require.NoError(t, err)

	// Then enable rules
	_, err = ProcessBuiltinCommand(ctx, "bumpers enable", dbPath, projectID)
	require.NoError(t, err)

	// Check status - should show enabled again
	statusResponse, err := ProcessBuiltinCommand(ctx, "bumpers status", dbPath, projectID)
	require.NoError(t, err)

	statusStr, ok := statusResponse.(string)
	require.True(t, ok, "Expected status response to be string, got %T", statusResponse)

	if !strings.Contains(statusStr, "enabled") {
		t.Errorf("Expected status to show rules are enabled after 'bumpers enable', got: %s", statusStr)
	}
}

func TestProcessBuiltinCommand_DifferentProjectsHaveIndependentState(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	project1 := "project-1"
	project2 := "project-2"

	// Disable rules for project 1
	_, err := ProcessBuiltinCommand(ctx, "bumpers disable", dbPath, project1)
	require.NoError(t, err)

	// Project 2 should still have rules enabled (default state)
	statusResponse, err := ProcessBuiltinCommand(ctx, "bumpers status", dbPath, project2)
	require.NoError(t, err)

	statusStr, ok := statusResponse.(string)
	require.True(t, ok, "Expected status response to be string, got %T", statusResponse)

	if !strings.Contains(statusStr, "enabled") {
		t.Errorf("Expected project 2 to have rules enabled when project 1 is disabled, got: %s", statusStr)
	}

	// Project 1 should still be disabled
	statusResponse1, err := ProcessBuiltinCommand(ctx, "bumpers status", dbPath, project1)
	require.NoError(t, err)

	statusStr1, ok := statusResponse1.(string)
	require.True(t, ok, "Expected status response to be string, got %T", statusResponse1)

	if !strings.Contains(statusStr1, "disabled") {
		t.Errorf("Expected project 1 to have rules disabled, got: %s", statusStr1)
	}
}

func TestProcessBuiltinCommand_StatePersistsAcrossProcessCalls(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	projectID := testProjectID

	// Disable rules
	_, err := ProcessBuiltinCommand(ctx, "bumpers disable", dbPath, projectID)
	require.NoError(t, err)

	// In a real scenario, this would be a separate process call
	// The state should persist because it's in the database, not in memory
	statusResponse, err := ProcessBuiltinCommand(ctx, "bumpers status", dbPath, projectID)
	require.NoError(t, err)

	statusStr, ok := statusResponse.(string)
	require.True(t, ok, "Expected status response to be string, got %T", statusResponse)

	if !strings.Contains(statusStr, "disabled") {
		t.Errorf("Expected disabled state to persist across process calls, got: %s", statusStr)
	}
}
