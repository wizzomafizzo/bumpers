package claude

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/wizzomafizzo/bumpers/internal/config"
)

// Launcher handles Claude binary discovery and execution
type Launcher struct {
	config *config.Config
}

// NewLauncher creates a new Claude launcher with the given configuration
func NewLauncher(cfg *config.Config) *Launcher {
	return &Launcher{
		config: cfg,
	}
}

// Common Claude installation locations to check as fallback
var commonLocations = []string{
	"/opt/homebrew/bin/claude", // macOS Homebrew
	"/usr/local/bin/claude",    // Unix standard location
}

// GetClaudePath returns the path to the Claude binary
func (*Launcher) GetClaudePath() (string, error) {
	attemptedPaths := make([]string, 0, 4) // Pre-allocate with estimated capacity

	// 1. Check local Claude installation
	homeDir, err := os.UserHomeDir()
	if err == nil {
		localPath := filepath.Join(homeDir, ".claude", "local", "claude")
		attemptedPaths = append(attemptedPaths, fmt.Sprintf("local: %s", localPath))

		if err := validateBinary(localPath); err == nil {
			return localPath, nil
		}
	}

	// 3. Check system PATH
	if pathBinary, err := exec.LookPath("claude"); err == nil {
		attemptedPaths = append(attemptedPaths, fmt.Sprintf("PATH: %s", pathBinary))

		if err := validateBinary(pathBinary); err == nil {
			return pathBinary, nil
		}
	} else {
		attemptedPaths = append(attemptedPaths, "PATH: not found")
	}

	// 4. Check common installation locations
	for _, location := range commonLocations {
		attemptedPaths = append(attemptedPaths, fmt.Sprintf("common: %s", location))

		if err := validateBinary(location); err == nil {
			return location, nil
		}
	}

	// Claude not found anywhere
	return "", &ClaudeNotFoundError{
		AttemptedPaths: attemptedPaths,
	}
}

// Execute runs Claude with the given arguments and returns the output
func (l *Launcher) Execute(args ...string) ([]byte, error) {
	claudePath, err := l.GetClaudePath()
	if err != nil {
		return nil, fmt.Errorf("failed to locate Claude binary: %w", err)
	}

	// #nosec G204 -- claudePath is validated before use
	cmd := exec.CommandContext(context.Background(), claudePath, args...)

	// Log the exact command being executed
	log.Debug().
		Str("claude_path", claudePath).
		Strs("args", args).
		Msg("Executing Claude command")

	output, err := cmd.CombinedOutput() // Use CombinedOutput to capture stderr too
	if err != nil {
		// Log detailed error information
		log.Error().
			Str("claude_path", claudePath).
			Strs("args", args).
			Str("output", string(output)).
			Err(err).
			Msg("Claude command failed")
		return nil, fmt.Errorf("failed to execute Claude: %w", err)
	}

	log.Debug().
		Str("claude_path", claudePath).
		Int("output_length", len(output)).
		Msg("Claude command succeeded")

	return output, nil
}

// GenerateMessage uses Claude to generate an AI response for the given prompt
func (l *Launcher) GenerateMessage(prompt string) (string, error) {
	output, err := l.Execute("-p", prompt)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// validateBinary checks if the given path is a valid, executable file
func validateBinary(path string) error {
	// Check if file exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("file does not exist")
		}
		return fmt.Errorf("cannot stat file: %w", err)
	}

	// Check if it's a regular file (not a directory)
	if !info.Mode().IsRegular() {
		return errors.New("not a regular file")
	}

	// Check if it's executable
	if info.Mode().Perm()&0o111 == 0 {
		return errors.New("file is not executable")
	}

	return nil
}

// ClaudeNotFoundError provides detailed information when Claude cannot be located
type ClaudeNotFoundError struct {
	AttemptedPaths []string
}

func (e *ClaudeNotFoundError) Error() string {
	var msg strings.Builder
	_, _ = msg.WriteString("Claude binary not found. Attempted locations:\n")

	for _, path := range e.AttemptedPaths {
		_, _ = msg.WriteString(fmt.Sprintf("  - %s\n", path))
	}

	_, _ = msg.WriteString("\nTo resolve this issue:\n")
	_, _ = msg.WriteString("  1. Install Claude Code from https://claude.ai/code\n")
	_, _ = msg.WriteString("  2. Ensure Claude is in your PATH\n")
	_, _ = msg.WriteString("  3. Or specify the path in bumpers.yml: claude_binary: \"/path/to/claude\"\n")

	return msg.String()
}

// IsClaudeNotFoundError returns true if the error is a ClaudeNotFoundError
func IsClaudeNotFoundError(err error) bool {
	var cnfErr *ClaudeNotFoundError
	return errors.As(err, &cnfErr)
}
