package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wizzomafizzo/bumpers/internal/claude/settings"
)

// createClaudeBackupCommand creates the claude backup command.
func createClaudeBackupCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "backup",
		Short: "Create a backup of Claude settings.json",
		Run: func(cmd *cobra.Command, _ []string) {
			output, err := runBackupCommandWithOutput()
			if err != nil {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Error creating backup: %v\n", err)
				return
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), output)
		},
	}
}

// createClaudeRestoreCommand creates the claude restore command.
func createClaudeRestoreCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "restore [backup_file]",
		Short: "Restore Claude settings from a backup",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			backupPath := args[0]

			currentDir, err := os.Getwd()
			if err != nil {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Error getting current directory: %v\n", err)
				return
			}

			settingsPath, err := findClaudeSettingsIn(currentDir)
			if err != nil {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Error finding settings.json in current directory: %v\n", err)
				return
			}

			if err := executeRestoreCommand(backupPath, settingsPath); err != nil {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Error restoring from backup: %v\n", err)
				return
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Successfully restored settings from %s\n", filepath.Base(backupPath))
		},
	}
}

// createRootCommand creates the root command with claude subcommands.
func createRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "bumpers",
		Short: "Claude Code hook guard",
	}

	// Create Claude commands group
	claudeCmd := &cobra.Command{
		Use:   "claude",
		Short: "Claude settings management",
	}
	claudeCmd.AddCommand(createClaudeBackupCommand())
	claudeCmd.AddCommand(createClaudeRestoreCommand())

	// Add subcommands
	rootCmd.AddCommand(claudeCmd)

	return rootCmd
}

// buildMainRootCommand creates the main root command with all subcommands.
func buildMainRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "bumpers",
		Short: "Claude Code hook guard",
	}

	testCmd := &cobra.Command{
		Use:   "test [command]",
		Short: "Test a command against current rules",
	}

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Check hook status",
	}

	// Create Claude commands group
	claudeCmd := &cobra.Command{
		Use:   "claude",
		Short: "Claude settings management",
	}
	claudeCmd.AddCommand(createClaudeBackupCommand())
	claudeCmd.AddCommand(createClaudeRestoreCommand())

	// Add subcommands
	rootCmd.AddCommand(testCmd, statusCmd, claudeCmd)

	return rootCmd
}

// findClaudeSettingsIn finds the Claude settings.json file in the given directory.
func findClaudeSettingsIn(dir string) (string, error) {
	settingsPath := filepath.Join(dir, "settings.json")

	// Check if settings.json exists in the directory
	if _, err := os.Stat(settingsPath); err != nil {
		return "", fmt.Errorf("failed to find settings file: %w", err)
	}

	return settingsPath, nil
}

// executeBackupCommand performs the backup operation for the Claude settings file.
func executeBackupCommand(dir string) (string, error) {
	// Find the settings file in the directory
	settingsPath, err := findClaudeSettingsIn(dir)
	if err != nil {
		return "", err
	}

	// Create backup using the settings package
	backupPath, err := settings.CreateBackup(settingsPath)
	if err != nil {
		return "", fmt.Errorf("failed to create backup: %w", err)
	}
	return backupPath, nil
}

// executeRestoreCommand restores settings from a backup file.
func executeRestoreCommand(backupPath, targetPath string) error {
	return settings.RestoreFromBackup(backupPath, targetPath) //nolint:wrapcheck // error already wrapped
}

// runBackupFromCurrentDirectory runs backup command using current working directory.
func runBackupFromCurrentDirectory() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err //nolint:wrapcheck // stdlib error
	}

	return executeBackupCommand(currentDir)
}

// runBackupCommandWithOutput runs backup and returns output message.
func runBackupCommandWithOutput() (string, error) {
	backupPath, err := runBackupFromCurrentDirectory()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Backup created: %s", filepath.Base(backupPath)), nil
}

// containsBackupInfo checks if output contains backup information.
func containsBackupInfo(output string) bool {
	return strings.Contains(output, "Backup created:")
}

// executeCLIBackupCommand executes the backup command and captures output.
