package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/wizzomafizzo/bumpers/configs"
	"github.com/wizzomafizzo/bumpers/internal/claude/settings"
	"github.com/wizzomafizzo/bumpers/internal/logger"
)

// Initialize sets up bumpers configuration and installs Claude hooks.
func (a *App) Initialize() error {
	// Get working directory for logger initialization
	workingDir := a.workDir
	if workingDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return err //nolint:wrapcheck // os error is descriptive
		}
		workingDir = cwd
	}

	// Initialize logger with working directory
	err := logger.Initialize(workingDir)
	if err != nil {
		return err //nolint:wrapcheck // logger initialization error is descriptive
	}

	// Create config file if it doesn't exist
	if _, statErr := os.Stat(a.configPath); os.IsNotExist(statErr) {
		writeErr := os.WriteFile(a.configPath, []byte(configs.DefaultConfig), 0o600)
		if writeErr != nil {
			return writeErr //nolint:wrapcheck // file operation error is self-explanatory
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
func (*App) Status() (string, error) {
	return "Bumpers status: configured", nil
}

// installClaudeHooks installs bumpers as a PreToolUse hook in Claude settings.
func (a *App) installClaudeHooks() error {
	// Use working directory from app, fallback to current working directory
	workingDir := a.workDir
	if workingDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return err //nolint:wrapcheck // os error is descriptive
		}
		workingDir = cwd
	}

	claudeDir := filepath.Join(workingDir, ".claude")
	localPath := filepath.Join(claudeDir, "settings.local.json")

	// Ensure .claude directory exists
	var err error
	err = os.MkdirAll(claudeDir, 0o750)
	if err != nil {
		return err //nolint:wrapcheck // directory creation error is descriptive
	}

	// Create settings.local.json if it doesn't exist
	var claudeSettings *settings.Settings
	if _, statErr := os.Stat(localPath); os.IsNotExist(statErr) {
		// Create empty settings
		claudeSettings = &settings.Settings{}
	} else {
		// Create backup before modifying existing settings
		_, backupErr := settings.CreateBackup(localPath)
		if backupErr != nil {
			return backupErr //nolint:wrapcheck // backup errors are already descriptive
		}

		// Load existing settings
		claudeSettings, err = settings.LoadFromFile(localPath)
		if err != nil {
			return err //nolint:wrapcheck // settings loading error is descriptive
		}
	}

	// Add bumpers hook for Bash commands with absolute paths
	bumpersPath := filepath.Join(workingDir, "bin", "bumpers")
	// Validate that bumpers binary exists
	if _, statErr := os.Stat(bumpersPath); os.IsNotExist(statErr) {
		return fmt.Errorf("bumpers binary not found at %s. Please run 'make build' first", bumpersPath)
	}
	configPath := a.configPath
	if !filepath.IsAbs(configPath) {
		configPath = filepath.Join(workingDir, configPath)
	}

	hookCmd := settings.HookCommand{
		Type:    "command",
		Command: bumpersPath + " -c " + configPath,
	}

	// Try to add hook, if it already exists, remove and re-add it
	err = claudeSettings.AddHook(settings.PreToolUseEvent, "Bash", hookCmd)
	if err != nil {
		// If hook already exists, remove it first then add the new one
		_ = claudeSettings.RemoveHook(settings.PreToolUseEvent, "Bash")
		err = claudeSettings.AddHook(settings.PreToolUseEvent, "Bash", hookCmd)
		if err != nil {
			return err //nolint:wrapcheck // hook addition error is descriptive
		}
	}

	// Save settings
	err = settings.SaveToFile(claudeSettings, localPath)
	if err != nil {
		return err //nolint:wrapcheck // settings save error is descriptive
	}

	return nil
}
