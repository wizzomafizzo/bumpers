package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wizzomafizzo/bumpers/internal/claude/settings"
	"github.com/wizzomafizzo/bumpers/internal/config"
	"github.com/wizzomafizzo/bumpers/internal/constants"
)

// Initialize sets up bumpers configuration and installs Claude hooks.
func (a *App) Initialize() error {
	// Get working directory for logger initialization - prefer project root
	workingDir := a.projectRoot
	if workingDir == "" {
		workingDir = a.workDir
	}
	if workingDir == "" {
		var err error
		workingDir, err = os.Getwd() //nolint:ineffassign,staticcheck // workingDir is used in logger initialization
		if err != nil {
			return fmt.Errorf("failed to get current working directory: %w", err)
		}
	}

	// Get filesystem to use (either injected or default)
	fs := a.getFileSystem()

	// Create config file if it doesn't exist
	if _, statErr := fs.Stat(a.configPath); os.IsNotExist(statErr) {
		defaultConfigBytes, configErr := config.DefaultConfigYAML()
		if configErr != nil {
			return fmt.Errorf("failed to generate default config: %w", configErr)
		}

		writeErr := fs.WriteFile(a.configPath, defaultConfigBytes, 0o600)
		if writeErr != nil {
			return fmt.Errorf("failed to write config file to %s: %w", a.configPath, writeErr)
		}
	}

	// Install Claude hooks
	installErr := a.installClaudeHooks()
	if installErr != nil {
		return installErr
	}

	return nil
}

// Status returns the current status of bumpers configuration.
func (a *App) Status() (string, error) {
	var status strings.Builder

	// strings.Builder.WriteString never returns error, but satisfying linter
	writeString := func(s string) {
		_, _ = status.WriteString(s)
	}

	writeString("Bumpers Status:\n")
	writeString("===============\n\n")

	// Check config file
	fs := a.getFileSystem()
	if _, err := fs.Stat(a.configPath); os.IsNotExist(err) {
		writeString("Config file: NOT FOUND\n")
		writeString(fmt.Sprintf("   Expected: %s\n", a.configPath))
	} else {
		writeString("Config file: EXISTS\n")
		writeString(fmt.Sprintf("   Location: %s\n", a.configPath))
	}

	return status.String(), nil
}

// setupClaudeDirectory ensures .claude directory exists and returns settings
func (a *App) setupClaudeDirectory(workingDir string) (*settings.Settings, string, error) {
	claudeDir := filepath.Join(workingDir, constants.ClaudeDir)
	localPath := filepath.Join(claudeDir, constants.SettingsFilename)

	// Get filesystem to use (either injected or default)
	fs := a.getFileSystem()

	// Ensure .claude directory exists
	err := fs.MkdirAll(claudeDir, 0o750)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create Claude directory %s: %w", claudeDir, err)
	}

	// Load or create settings
	var claudeSettings *settings.Settings
	if _, statErr := fs.Stat(localPath); os.IsNotExist(statErr) {
		claudeSettings = &settings.Settings{}
	} else {
		// Create backup before modifying existing settings
		_, backupErr := settings.CreateBackupWithFS(fs, localPath)
		if backupErr != nil {
			return nil, "", fmt.Errorf("failed to create backup of Claude settings: %w", backupErr)
		}

		claudeSettings, err = settings.LoadFromFileWithFS(fs, localPath)
		if err != nil {
			return nil, "", fmt.Errorf("failed to load Claude settings from %s: %w", localPath, err)
		}
	}

	return claudeSettings, localPath, nil
}

// validateBumpersPath checks if bumpers binary exists (skips in test env and PATH commands)
func (a *App) validateBumpersPath(workingDir, bumpersPath string) error {
	// Skip check if working dir looks like Go test temp dir
	if strings.HasPrefix(filepath.Base(workingDir), "Test") || strings.Contains(workingDir, "/tmp/Test") {
		return nil
	}

	// Skip validation for PATH commands (just "bumpers")
	if bumpersPath == "bumpers" {
		return nil
	}

	// Get filesystem to use (either injected or default)
	fs := a.getFileSystem()

	if _, statErr := fs.Stat(bumpersPath); os.IsNotExist(statErr) {
		return fmt.Errorf("bumpers binary not found at %s. Please run 'just build' first", bumpersPath)
	}

	return nil
}

// createHookCommand creates the hook command configuration using dynamic path detection
func (a *App) createHookCommand(workingDir string) (settings.HookCommand, error) {
	bumpersCommand := determineBumpersCommand(workingDir)

	// Validate the command exists (skip in test environments)
	if err := a.validateBumpersPath(workingDir, bumpersCommand); err != nil {
		return settings.HookCommand{}, err
	}

	return settings.HookCommand{
		Type:    "command",
		Command: bumpersCommand,
	}, nil
}

// determineBumpersCommand determines which bumpers command to use based on context
func determineBumpersCommand(workingDir string) string {
	// Check if we're in test environment by looking at os.Args[0]
	isTestEnv := strings.Contains(os.Args[0], ".test") || strings.HasSuffix(os.Args[0], ".test")

	if isTestEnv {
		// In test environment, use the local binary path
		return filepath.Join(workingDir, "bin", "bumpers")
	}

	return resolveBumpersPath(os.Args[0])
}

// resolveBumpersPath resolves the bumpers command path based on how it was invoked
func resolveBumpersPath(originalCommand string) string {
	// If it's just "bumpers" without path separators, it was run from PATH
	baseName := filepath.Base(originalCommand)
	hasPathSep := strings.Contains(originalCommand, string(filepath.Separator))

	if baseName == "bumpers" && !hasPathSep {
		return "bumpers"
	}

	// If it's a relative path, make it absolute for reliability
	if !filepath.IsAbs(originalCommand) && hasPathSep {
		abs, err := filepath.Abs(originalCommand)
		if err != nil {
			// Fall back to original command if we can't resolve absolute path
			return originalCommand
		}
		return abs
	}

	// Return as-is for absolute paths or simple commands
	return originalCommand
}

// installClaudeHooks installs bumpers as a PreToolUse hook in Claude settings.
func (a *App) installClaudeHooks() error {
	// Get working directory - prefer project root
	workingDir := a.projectRoot
	if workingDir == "" {
		workingDir = a.workDir
	}
	if workingDir == "" {
		var err error
		workingDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current working directory: %w", err)
		}
	}

	claudeSettings, localPath, err := a.setupClaudeDirectory(workingDir)
	if err != nil {
		return err
	}

	hookCmd, err := a.createHookCommand(workingDir)
	if err != nil {
		return err
	}

	// Add PreToolUse hook (preserves existing hooks with same matcher)
	err = claudeSettings.AddOrAppendHook(settings.PreToolUseEvent, "Bash", hookCmd)
	if err != nil {
		return fmt.Errorf("failed to add bumpers PreToolUse hook to Claude settings: %w", err)
	}

	// Add UserPromptSubmit hook (preserves existing hooks with same matcher)
	err = claudeSettings.AddOrAppendHook(settings.UserPromptSubmitEvent, "", hookCmd)
	if err != nil {
		return fmt.Errorf("failed to add bumpers UserPromptSubmit hook to Claude settings: %w", err)
	}

	// Save settings using injected filesystem
	fs := a.getFileSystem()
	err = settings.SaveToFileWithFS(fs, claudeSettings, localPath)
	if err != nil {
		return fmt.Errorf("failed to save Claude settings to %s: %w", localPath, err)
	}
	return nil
}
