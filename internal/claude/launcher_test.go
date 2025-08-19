package claude

import (
	"os"
	"path/filepath"
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

func TestGetClaudePathWithConfigOverride(t *testing.T) {
	t.Parallel()
	// Test that config override is attempted, but if it fails, fallback works
	cfg := &config.Config{
		ClaudeBinary: "/definitely/nonexistent/path/to/claude",
	}
	launcher := NewLauncher(cfg)

	path, err := launcher.GetClaudePath()

	// This might succeed if Claude is installed in fallback locations
	// OR fail with detailed error - both are valid outcomes
	if err == nil {
		t.Logf("Config override failed but fallback found Claude at: %s", path)
	} else {
		// Error should be detailed, showing the config path was tried first
		t.Logf("Expected detailed error when Claude not found: %v", err)
		// Verify this is the detailed error type
		if IsClaudeNotFoundError(err) {
			t.Log("Got detailed ClaudeNotFoundError (good)")
		} else {
			t.Error("Should get ClaudeNotFoundError with details")
		}
	}
}

func TestGetClaudePathWithValidConfig(t *testing.T) {
	t.Parallel()
	// Create a temporary file to act as a valid Claude binary
	tempDir := t.TempDir()
	claudePath := filepath.Join(tempDir, "claude")

	// Create the file and make it executable
	file, err := os.Create(claudePath) // #nosec G304 -- using safe temp directory path
	if err != nil {
		t.Fatalf("Failed to create temp Claude binary: %v", err)
	}
	if closeErr := file.Close(); closeErr != nil {
		t.Fatalf("Failed to close temp Claude binary: %v", closeErr)
	}

	err = os.Chmod(claudePath, 0o755) // #nosec G302 -- executable permission needed for test binary
	if err != nil {
		t.Fatalf("Failed to make temp Claude binary executable: %v", err)
	}

	// Test that config override works with valid path
	cfg := &config.Config{
		ClaudeBinary: claudePath,
	}
	launcher := NewLauncher(cfg)

	path, err := launcher.GetClaudePath()
	if err != nil {
		t.Errorf("Expected success with valid config path, got error: %v", err)
	}

	if path != claudePath {
		t.Errorf("Expected path %s, got %s", claudePath, path)
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
