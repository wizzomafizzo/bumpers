//nolint:tagliatelle // JSON tags must match Claude API format
package claude

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

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
	return "", &NotFoundError{
		AttemptedPaths: attemptedPaths,
	}
}

// ExecuteWithInput runs Claude with the given input and arguments using pipes like the working SDK
func (l *Launcher) ExecuteWithInput(input string) ([]byte, error) {
	claudePath, err := l.GetClaudePath()
	if err != nil {
		return nil, fmt.Errorf("failed to locate Claude binary: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmdArgs := []string{
		"--print",
		"--output-format", "json",
		"--model", "sonnet",
		"--max-turns", "3",
		input,
	}

	cmd := exec.CommandContext(ctx, claudePath, cmdArgs...) //nolint:gosec // claudePath is validated via GetClaudePath
	cmd.Env = append(os.Environ(), "BUMPERS_SKIP=1")

	log.Debug().
		Str("claude_path", claudePath).
		Strs("args", cmdArgs).
		Int("input_length", len(input)).
		Msg("executing Claude Code command")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute Claude Code command: %w", err)
	}

	return output, nil
}

// ServerToolUse represents server tool usage statistics
type ServerToolUse struct {
	WebSearchRequests int `json:"web_search_requests"` //nolint:tagliatelle // matches Claude API format
}

// Usage represents token usage and service information
type Usage struct {
	ServiceTier              string        `json:"service_tier"`
	InputTokens              int           `json:"input_tokens"`
	CacheCreationInputTokens int           `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int           `json:"cache_read_input_tokens"`
	OutputTokens             int           `json:"output_tokens"`
	ServerToolUse            ServerToolUse `json:"server_tool_use"`
}

// CLIResponse represents the parsed JSON output from Claude Code execution
type CLIResponse struct {
	Type              string   `json:"type"`
	Subtype           string   `json:"subtype"`
	Result            string   `json:"result"`
	SessionID         string   `json:"session_id"`
	UUID              string   `json:"uuid"`
	PermissionDenials []string `json:"permission_denials"`
	Usage             Usage    `json:"usage"`
	DurationMs        int      `json:"duration_ms"`
	DurationAPIMs     int      `json:"duration_api_ms"`
	NumTurns          int      `json:"num_turns"`
	TotalCostUsd      float64  `json:"total_cost_usd"`
	IsError           bool     `json:"is_error"`
}

// GenerateMessage uses Claude to generate an AI response for the given prompt
func (l *Launcher) GenerateMessage(prompt string) (string, error) {
	output, err := l.ExecuteWithInput(prompt)
	if err != nil {
		return "", err
	}

	var response CLIResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return "", fmt.Errorf("failed to parse claude code response: %w", err)
	}

	if response.IsError {
		return "", fmt.Errorf("claude code returned error response: %s", response.Result)
	}

	return response.Result, nil
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

// NotFoundError provides detailed information when Claude cannot be located
type NotFoundError struct {
	AttemptedPaths []string
}

func (e *NotFoundError) Error() string {
	var msg strings.Builder
	_, _ = msg.WriteString("Claude Code binary not found. Attempted locations:\n")

	for _, path := range e.AttemptedPaths {
		_, _ = msg.WriteString(fmt.Sprintf("  - %s\n", path))
	}

	_, _ = msg.WriteString("\nTo resolve this issue:\n")
	_, _ = msg.WriteString("  1. Install Claude Code from https://claude.ai/code\n")
	_, _ = msg.WriteString("  2. Ensure Claude is in your PATH\n")
	_, _ = msg.WriteString("  3. Or specify the path in bumpers.yml: claude_binary: \"/path/to/claude\"\n")

	return msg.String()
}

// IsClaudeNotFoundError checks if the error is a Claude not found error
func IsClaudeNotFoundError(err error) bool {
	notFoundError := &NotFoundError{}
	ok := errors.As(err, &notFoundError)
	return ok
}
