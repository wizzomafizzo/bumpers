package claude

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wizzomafizzo/bumpers/internal/config"
	"github.com/wizzomafizzo/bumpers/internal/testutil"
)

func TestIsClaudeNotFoundError(t *testing.T) {
	testutil.InitTestLogger(t)
	t.Parallel()

	// Test with NotFoundError
	notFoundErr := &NotFoundError{AttemptedPaths: []string{"test"}}
	if !IsClaudeNotFoundError(notFoundErr) {
		t.Error("IsClaudeNotFoundError should return true for NotFoundError")
	}

	// Test with other error types
	otherErr := errors.New("some other error")
	if IsClaudeNotFoundError(otherErr) {
		t.Error("IsClaudeNotFoundError should return false for non-NotFoundError")
	}

	// Test with nil
	if IsClaudeNotFoundError(nil) {
		t.Error("IsClaudeNotFoundError should return false for nil")
	}
}

func TestNewLauncher(t *testing.T) {
	testutil.InitTestLogger(t)
	t.Parallel()

	cfg := &config.Config{
		Rules: []config.Rule{
			{Match: "^go test", Send: "Use just test instead"},
		},
	}

	launcher := NewLauncher(cfg)

	assert.NotNil(t, launcher, "NewLauncher should return non-nil launcher")
	assert.Equal(t, cfg, launcher.config, "Launcher should store provided config")
}

func TestValidateBinary(t *testing.T) {
	testutil.InitTestLogger(t)
	t.Parallel()

	tests := []struct {
		setupFunc   func(t *testing.T) string
		name        string
		errContains string
		wantErr     bool
	}{
		{
			name: "valid executable file",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				execPath := filepath.Join(tmpDir, "test-executable")

				// Create executable file
				//nolint:gosec // test file needs execute permissions
				err := os.WriteFile(execPath, []byte("#!/bin/sh\necho test"), 0o755)
				require.NoError(t, err)

				return execPath
			},
			wantErr: false,
		},
		{
			name: "non-existent file",
			setupFunc: func(_ *testing.T) string {
				return "/path/that/does/not/exist"
			},
			wantErr:     true,
			errContains: "file does not exist",
		},
		{
			name: "directory instead of file",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				return tmpDir
			},
			wantErr:     true,
			errContains: "not a regular file",
		},
		{
			name: "non-executable file",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				nonExecPath := filepath.Join(tmpDir, "non-executable")

				// Create non-executable file
				err := os.WriteFile(nonExecPath, []byte("test content"), 0o600)
				require.NoError(t, err)

				return nonExecPath
			},
			wantErr:     true,
			errContains: "file is not executable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := tt.setupFunc(t)

			err := validateBinary(path)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNotFoundError_Error(t *testing.T) {
	testutil.InitTestLogger(t)
	t.Parallel()

	tests := []struct {
		name           string
		attemptedPaths []string
		wantContains   []string
	}{
		{
			name:           "single path",
			attemptedPaths: []string{"local: /home/user/.claude/local/claude"},
			wantContains: []string{
				"Claude Code binary not found",
				"local: /home/user/.claude/local/claude",
				"Install Claude Code from https://claude.ai/code",
				"Ensure Claude is in your PATH",
			},
		},
		{
			name: "multiple paths",
			attemptedPaths: []string{
				"local: /home/user/.claude/local/claude",
				"PATH: not found",
				"common: /usr/local/bin/claude",
			},
			wantContains: []string{
				"Claude Code binary not found",
				"local: /home/user/.claude/local/claude",
				"PATH: not found",
				"common: /usr/local/bin/claude",
			},
		},
		{
			name:           "empty paths",
			attemptedPaths: []string{},
			wantContains: []string{
				"Claude Code binary not found",
				"Install Claude Code from https://claude.ai/code",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := &NotFoundError{AttemptedPaths: tt.attemptedPaths}

			errMsg := err.Error()

			for _, want := range tt.wantContains {
				assert.Contains(t, errMsg, want, "Error message should contain expected text")
			}
		})
	}
}

func getCLIResponseTestCases() []struct {
	name        string
	jsonOutput  string
	wantResult  string
	errContains string
	wantErr     bool
} {
	return []struct {
		name        string
		jsonOutput  string
		wantResult  string
		errContains string
		wantErr     bool
	}{
		{
			name: "successful response",
			jsonOutput: `{
				"type": "conversation",
				"subtype": "response",
				"result": "Hello, this is Claude!",
				"session_id": "test-session",
				"uuid": "test-uuid",
				"usage": {"input_tokens": 10, "output_tokens": 5},
				"duration_ms": 1000,
				"num_turns": 1,
				"is_error": false
			}`,
			wantResult: "Hello, this is Claude!",
			wantErr:    false,
		},
		{
			name: "error response",
			jsonOutput: `{
				"type": "error",
				"subtype": "api_error",
				"result": "API rate limit exceeded",
				"session_id": "test-session",
				"uuid": "test-uuid",
				"usage": {"input_tokens": 0, "output_tokens": 0},
				"duration_ms": 100,
				"num_turns": 0,
				"is_error": true
			}`,
			wantResult:  "",
			wantErr:     true,
			errContains: "API rate limit exceeded",
		},
		{
			name:        "invalid JSON",
			jsonOutput:  `{"invalid": json}`,
			wantResult:  "",
			wantErr:     true,
			errContains: "invalid character",
		},
		{
			name:        "empty response",
			jsonOutput:  "",
			wantResult:  "",
			wantErr:     true,
			errContains: "unexpected end of JSON input",
		},
	}
}

func TestCLIResponse_JSONParsing(t *testing.T) {
	testutil.InitTestLogger(t)
	t.Parallel()

	tests := getCLIResponseTestCases()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var response CLIResponse
			err := json.Unmarshal([]byte(tt.jsonOutput), &response)

			if strings.Contains(tt.name, "invalid JSON") || strings.Contains(tt.name, "empty response") {
				assert.Error(t, err, "Should fail to parse invalid JSON")
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err, "Should parse valid JSON")

			if response.IsError {
				assert.True(t, tt.wantErr, "Should expect error for error response")
				expectedErrMsg := strings.Replace(tt.errContains, "claude code returned error response: ", "", 1)
				assert.Contains(t, response.Result, expectedErrMsg)
			} else {
				assert.False(t, tt.wantErr, "Should not expect error for success response")
				assert.Equal(t, tt.wantResult, response.Result)
			}
		})
	}
}

// Example demonstrates how to create a Claude launcher with configuration
func ExampleNewLauncher() {
	// Create a basic configuration
	cfg := &config.Config{
		Rules: []config.Rule{
			{Match: "^go test", Send: "Use just test instead"},
		},
	}

	launcher := NewLauncher(cfg)

	// Check that launcher was created successfully
	if launcher != nil {
		_, _ = fmt.Println("Claude launcher created successfully")
	}

	// Output:
	// Claude launcher created successfully
}
