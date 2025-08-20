package cli

import (
	"os"
	"path/filepath"

	"github.com/wizzomafizzo/bumpers/configs"
	"github.com/wizzomafizzo/bumpers/internal/claude/settings"
)

// Initialize sets up bumpers configuration and installs Claude hooks.
func (a *App) Initialize() error {
	// Create config file if it doesn't exist
	if _, err := os.Stat(a.configPath); os.IsNotExist(err) {
		err := os.WriteFile(a.configPath, []byte(configs.DefaultConfig), 0o600)
		if err != nil {
			return err //nolint:wrapcheck // file operation error is self-explanatory
		}
	}

	// Install Claude hooks
	err := a.installClaudeHooks()
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
	// Use current working directory's .claude folder
	cwd, err := os.Getwd()
	if err != nil {
		return err //nolint:wrapcheck // os error is descriptive
	}

	claudeDir := filepath.Join(cwd, ".claude")
	localPath := filepath.Join(claudeDir, "settings.local.json")

	// Ensure .claude directory exists
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
	bumpersPath := filepath.Join(cwd, "bin", "bumpers")
	configPath := a.configPath
	if !filepath.IsAbs(configPath) {
		configPath = filepath.Join(cwd, configPath)
	}

	hookCmd := settings.HookCommand{
		Type:    "command",
		Command: bumpersPath + " -c " + configPath,
		Timeout: 30,
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
