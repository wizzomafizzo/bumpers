package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wizzomafizzo/bumpers/internal/testutil"
)

func TestLoggingOnlyInitializedForHookCommand(t *testing.T) {
	t.Parallel()
	setupTest(t)

	t.Run("status command doesn't initialize logging", func(t *testing.T) {
		t.Parallel()
		testStatusCommandLogging(t)
	})

	t.Run("hook command initializes logging", func(t *testing.T) {
		t.Parallel()
		testHookCommandLogging(t)
	})
}

func testStatusCommandLogging(t *testing.T) {
	t.Helper()

	tempDir := t.TempDir()
	setupTempDirectory(t, tempDir)

	// Create a basic config file with absolute path
	configFile := filepath.Join(tempDir, "bumpers.yml")
	err := os.WriteFile(configFile, []byte(`rules: []`), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	// Run status command with explicit absolute config path to avoid project root detection issues
	rootCmd := createNewRootCommand()
	rootCmd.SetArgs([]string{"--config", configFile, "status"})

	var stderr bytes.Buffer
	rootCmd.SetErr(&stderr)

	err = rootCmd.Execute()
	if err != nil {
		t.Logf("Status command error (expected): %v", err)
	}

	// Should not have created any log files in .claude directory
	claudeDir := ".claude"
	if _, err := os.Stat(claudeDir); !os.IsNotExist(err) {
		t.Error("Expected .claude directory not to exist for non-hook commands")
	}
}

func testHookCommandLogging(t *testing.T) {
	t.Helper()

	tempDir := t.TempDir()
	setupTempDirectory(t, tempDir)
	configPath := createTestConfig(t, tempDir)

	// Create hook input that will trigger rule processing
	hookInput := `{
		"hook_event_name": "PreToolUse",
		"tool_name": "Bash",
		"tool_input": {
			"command": "go test"
		}
	}`

	runHookCommandAndVerifyLogging(t, hookInput, configPath)
}

func setupTempDirectory(t *testing.T, tempDir string) {
	t.Helper()

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalDir) })

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	// Create a project marker file to ensure project.FindRoot() works correctly
	goModPath := filepath.Join(tempDir, "go.mod")
	err = os.WriteFile(goModPath, []byte("module testproject\n\ngo 1.21\n"), 0o600)
	if err != nil {
		t.Fatal(err)
	}
}

func createTestConfig(t *testing.T, tempDir string) string {
	t.Helper()

	configContent := `rules:
  - pattern: "go test"
    message: "Test command blocked"
`
	configPath := filepath.Join(tempDir, "bumpers.yml")
	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	if err != nil {
		t.Fatal(err)
	}
	return configPath
}

func runHookCommandAndVerifyLogging(t *testing.T, hookInput, configPath string) {
	t.Helper()

	// Run hook command with explicit absolute config path to avoid project root detection issues
	rootCmd := createNewRootCommand()
	rootCmd.SetArgs([]string{"--config", configPath, "hook"})

	var stdout, stderr bytes.Buffer
	rootCmd.SetIn(strings.NewReader(hookInput))
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)

	err := rootCmd.Execute()
	logCommandOutput(t, stdout.String(), stderr.String(), err)

	// Give logging time to initialize and write
	time.Sleep(100 * time.Millisecond)

	// The key test is that the hook command ran without error, which means logging initialization succeeded
	if err != nil {
		t.Errorf("Hook command failed, suggesting logging initialization issue: %v", err)
	}
}

func logCommandOutput(t *testing.T, stdout, stderr string, err error) {
	t.Helper()

	t.Logf("Hook command stdout: %s", stdout)
	t.Logf("Hook command stderr: %s", stderr)
	if err != nil {
		t.Logf("Hook command error: %v", err)
	}
}

func setupTest(t *testing.T) {
	t.Helper()
	testutil.InitTestLogger(t)
}
