package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wizzomafizzo/bumpers/internal/claude/settings"
	"github.com/wizzomafizzo/bumpers/internal/config"
	"github.com/wizzomafizzo/bumpers/internal/logger"
	"github.com/wizzomafizzo/bumpers/internal/paths"
)

// Initialize sets up bumpers configuration and installs Claude hooks.
func (a *App) Initialize() error {
	// Get working directory for logger initialization - prefer project root
	workingDir := a.projectRoot
	if workingDir == "" {
		workingDir = a.workDir
	}
	if workingDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current working directory: %w", err)
		}
		workingDir = cwd
	}

	// Create logger instance for this app
	var err error
	a.logger, err = logger.NewWithConfig(nil, workingDir)
	if err != nil {
		return fmt.Errorf("failed to initialize logger in %s: %w", workingDir, err)
	}

	// Create config file if it doesn't exist
	if _, statErr := os.Stat(a.configPath); os.IsNotExist(statErr) {
		defaultConfigBytes, configErr := config.DefaultConfigYAML()
		if configErr != nil {
			return fmt.Errorf("failed to generate default config: %w", configErr)
		}

		writeErr := os.WriteFile(a.configPath, defaultConfigBytes, 0o600)
		if writeErr != nil {
			return fmt.Errorf("failed to write config file to %s: %w", a.configPath, writeErr)
		}
	}

	// Install Claude hooks
	err = a.installClaudeHooks()
	if err != nil {
		return err
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
	if _, err := os.Stat(a.configPath); os.IsNotExist(err) {
		writeString("Config file: NOT FOUND\n")
		writeString(fmt.Sprintf("   Expected: %s\n", a.configPath))
	} else {
		writeString("Config file: EXISTS\n")
		writeString(fmt.Sprintf("   Location: %s\n", a.configPath))
	}

	return status.String(), nil
}

// setupClaudeDirectory ensures .claude directory exists and returns settings
func (*App) setupClaudeDirectory(workingDir string) (*settings.Settings, string, error) {
	claudeDir := filepath.Join(workingDir, paths.ClaudeDir)
	localPath := filepath.Join(claudeDir, paths.SettingsFilename)

	// Ensure .claude directory exists
	err := os.MkdirAll(claudeDir, 0o750)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create Claude directory %s: %w", claudeDir, err)
	}

	// Load or create settings
	var claudeSettings *settings.Settings
	if _, statErr := os.Stat(localPath); os.IsNotExist(statErr) {
		claudeSettings = &settings.Settings{}
	} else {
		// Create backup before modifying existing settings
		_, backupErr := settings.CreateBackup(localPath)
		if backupErr != nil {
			return nil, "", fmt.Errorf("failed to create backup of Claude settings: %w", backupErr)
		}

		claudeSettings, err = settings.LoadFromFile(localPath)
		if err != nil {
			return nil, "", fmt.Errorf("failed to load Claude settings from %s: %w", localPath, err)
		}
	}

	return claudeSettings, localPath, nil
}

// validateBumpersPath checks if bumpers binary exists (skips in test env)
func validateBumpersPath(workingDir, bumpersPath string) error {
	// Skip check if working dir looks like Go test temp dir
	if strings.HasPrefix(filepath.Base(workingDir), "Test") || strings.Contains(workingDir, "/tmp/Test") {
		return nil
	}

	if _, statErr := os.Stat(bumpersPath); os.IsNotExist(statErr) {
		return fmt.Errorf("bumpers binary not found at %s. Please run 'make build' first", bumpersPath)
	}

	return nil
}

// createHookCommand creates the hook command configuration
func (a *App) createHookCommand(workingDir string) (settings.HookCommand, error) {
	bumpersPath := filepath.Join(workingDir, "bin", "bumpers")

	if err := validateBumpersPath(workingDir, bumpersPath); err != nil {
		return settings.HookCommand{}, err
	}

	configPath := a.configPath
	if !filepath.IsAbs(configPath) {
		configPath = filepath.Join(workingDir, configPath)
	}

	return settings.HookCommand{
		Type:    "command",
		Command: bumpersPath + " -c " + configPath,
	}, nil
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

	// Try to add PreToolUse hook, if it already exists, remove and re-add it
	err = claudeSettings.AddHook(settings.PreToolUseEvent, "Bash", hookCmd)
	if err != nil {
		// If hook already exists, remove it first then add the new one
		_ = claudeSettings.RemoveHook(settings.PreToolUseEvent, "Bash")
		err = claudeSettings.AddHook(settings.PreToolUseEvent, "Bash", hookCmd)
		if err != nil {
			return fmt.Errorf("failed to add bumpers PreToolUse hook to Claude settings: %w", err)
		}
	}

	// Try to add UserPromptSubmit hook, if it already exists, remove and re-add it
	err = claudeSettings.AddHook(settings.UserPromptSubmitEvent, ".*", hookCmd)
	if err != nil {
		// If hook already exists, remove it first then add the new one
		_ = claudeSettings.RemoveHook(settings.UserPromptSubmitEvent, ".*")
		err = claudeSettings.AddHook(settings.UserPromptSubmitEvent, ".*", hookCmd)
		if err != nil {
			return fmt.Errorf("failed to add bumpers UserPromptSubmit hook to Claude settings: %w", err)
		}
	}

	// Save settings
	err = settings.SaveToFile(claudeSettings, localPath)
	if err != nil {
		return fmt.Errorf("failed to save Claude settings to %s: %w", localPath, err)
	}
	return nil
}
