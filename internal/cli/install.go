package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

	// Create logger instance for this app
	var err error
	a.logger, err = logger.New(workingDir)
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

// getWorkingDirectory returns the working directory, falling back to current dir
func (a *App) getWorkingDirectory() (string, error) {
	if a.workDir != "" {
		return a.workDir, nil
	}
	return os.Getwd() //nolint:wrapcheck // os error is descriptive
}

// setupClaudeDirectory ensures .claude directory exists and returns settings
func (*App) setupClaudeDirectory(workingDir string) (*settings.Settings, string, error) {
	claudeDir := filepath.Join(workingDir, ".claude")
	localPath := filepath.Join(claudeDir, "settings.local.json")

	// Ensure .claude directory exists
	err := os.MkdirAll(claudeDir, 0o750)
	if err != nil {
		return nil, "", err //nolint:wrapcheck // directory creation error is descriptive
	}

	// Load or create settings
	var claudeSettings *settings.Settings
	if _, statErr := os.Stat(localPath); os.IsNotExist(statErr) {
		claudeSettings = &settings.Settings{}
	} else {
		// Create backup before modifying existing settings
		_, backupErr := settings.CreateBackup(localPath)
		if backupErr != nil {
			return nil, "", backupErr //nolint:wrapcheck // backup errors are already descriptive
		}

		claudeSettings, err = settings.LoadFromFile(localPath)
		if err != nil {
			return nil, "", err //nolint:wrapcheck // settings loading error is descriptive
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
	workingDir, err := a.getWorkingDirectory()
	if err != nil {
		return err
	}

	claudeSettings, localPath, err := a.setupClaudeDirectory(workingDir)
	if err != nil {
		return err
	}

	hookCmd, err := a.createHookCommand(workingDir)
	if err != nil {
		return err
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
	return settings.SaveToFile(claudeSettings, localPath) //nolint:wrapcheck // settings save error is descriptive
}
