package claude

import (
	"testing"

	"github.com/wizzomafizzo/bumpers/internal/config"
)

func TestNewLauncher(t *testing.T) {
	t.Parallel()
	// Test that NewLauncher creates a launcher instance
	cfg := &config.Config{}
	launcher := NewLauncher(cfg)

	if launcher == nil {
		t.Error("NewLauncher should return a non-nil launcher")
		return
	}

	if launcher.config != cfg {
		t.Error("Launcher should store the provided config")
	}
}

func TestGetClaudePath(t *testing.T) {
	t.Parallel()
	// Test that GetClaudePath works with fallback chain
	cfg := &config.Config{}
	launcher := NewLauncher(cfg)

	path, err := launcher.GetClaudePath()

	// This might succeed if Claude is installed, or fail with detailed error
	if err == nil {
		t.Logf("Found Claude at: %s", path)
		// Verify it's a valid path
		if path == "" {
			t.Error("Path should not be empty when error is nil")
		}
	} else {
		t.Logf("Claude not found (detailed error): %v", err)
		// Should be a detailed error
		if IsClaudeNotFoundError(err) {
			t.Log("Got detailed ClaudeNotFoundError (good)")
		}
	}
}

func TestGetClaudePathBasic(t *testing.T) {
	t.Parallel()
	// Test that launcher works with basic config
	cfg := &config.Config{}
	launcher := NewLauncher(cfg)

	// This may find Claude or return error - both are valid outcomes
	path, err := launcher.GetClaudePath()
	if err == nil {
		t.Logf("Found Claude at: %s", path)
	} else {
		if IsClaudeNotFoundError(err) {
			t.Logf("Got expected Claude not found error: %v", err)
		} else {
			t.Errorf("Expected ClaudeNotFoundError, got: %v", err)
		}
	}
}

func TestGetClaudePathFallbackChain(t *testing.T) {
	t.Parallel()
	// Test that the fallback chain is attempted when no config is provided
	cfg := &config.Config{} // No config override
	launcher := NewLauncher(cfg)

	path, err := launcher.GetClaudePath()

	// This might find Claude or return detailed error - both valid
	if err == nil {
		t.Logf("Found Claude via fallback chain at: %s", path)
	} else {
		// Error should show attempted locations, not just be generic
		if IsClaudeNotFoundError(err) {
			t.Logf("Got detailed error (good): %v", err)
		} else {
			t.Error("Should get ClaudeNotFoundError with attempted locations")
		}
	}
}

func TestExecute(t *testing.T) {
	t.Parallel()
	// Test the Execute method
	cfg := &config.Config{}
	launcher := NewLauncher(cfg)

	// Try to execute claude --version (should work if Claude is available)
	output, err := launcher.Execute("--version")

	if err == nil {
		// If successful, output should not be empty
		if len(output) == 0 {
			t.Error("Expected non-empty output from claude --version, got empty")
		} else {
			t.Logf("Claude execute successful, output: %s", string(output))
		}
	} else {
		t.Logf("Claude execute failed (expected if Claude not available): %v", err)
	}
}
