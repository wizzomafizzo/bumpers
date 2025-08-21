package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wizzomafizzo/bumpers/internal/claude/settings"
	"github.com/wizzomafizzo/bumpers/internal/cli"
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

// createClaudeCommandGroup creates the claude command group with backup/restore subcommands.
func createClaudeCommandGroup() *cobra.Command {
	claudeCmd := &cobra.Command{
		Use:   "claude",
		Short: "Claude settings management",
	}
	claudeCmd.AddCommand(createClaudeBackupCommand())
	claudeCmd.AddCommand(createClaudeRestoreCommand())
	return claudeCmd
}

// createRootCommand creates the root command with claude subcommands.
func createRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "bumpers",
		Short: "Claude Code hook guard",
	}

	// Add subcommands
	rootCmd.AddCommand(createClaudeCommandGroup())

	return rootCmd
}

// processHookCommand processes hook input and returns exit code and error message
func processHookCommand(configPath string, input io.Reader, _ io.Writer) (code int, response string) {
	app := cli.NewApp(configPath)

	response, err := app.ProcessHook(input)
	if err != nil {
		return 1, fmt.Sprintf("Error: %v", err)
	}

	if response != "" {
		return 2, response // Exit code 2 for Claude Code hook blocking
	}

	return 0, ""
}

// createMainRootCommand creates the main root command
func createMainRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "bumpers",
		Short:        "Claude Code hook guard",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			configFlag, _ := cmd.PersistentFlags().GetString("config")

			exitCode, message := processHookCommand(configFlag, cmd.InOrStdin(), cmd.ErrOrStderr())

			if message != "" {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", message)
			}

			if exitCode != 0 {
				os.Exit(exitCode) //nolint:revive // Claude Code hooks require specific exit codes
			}

			return nil
		},
	}
	return cmd
}

// createInstallCommand creates the install subcommand
func createInstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install bumpers configuration and Claude hooks",
		RunE: func(cmd *cobra.Command, _ []string) error {
			configFlag, _ := cmd.Parent().PersistentFlags().GetString("config")
			app := cli.NewApp(configFlag)

			err := app.Initialize()
			if err != nil {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Error: %v\n", err)
				return errors.New("installation failed")
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Bumpers installed successfully!")
			return nil
		},
	}
}

// createTestCommand creates the test subcommand
func createTestCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "test [command]",
		Short: "Test a command against current rules",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			configFlag, _ := cmd.Parent().PersistentFlags().GetString("config")
			app := cli.NewApp(configFlag)

			result, err := app.TestCommand(args[0])
			if err != nil {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Error: %v\n", err)
				return errors.New("test command failed")
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), result)
			return nil
		},
	}
}

// createStatusCommand creates the status subcommand
func createStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check hook status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			configPath, _ := cmd.Flags().GetString("config")
			app := cli.NewApp(configPath)
			status, err := app.Status()
			if err != nil {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Error: %v\n", err)
				return errors.New("status check failed")
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), status)
			return nil
		},
	}
}

// createValidateCommand creates the validate subcommand
func createValidateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration file",
		RunE: func(cmd *cobra.Command, _ []string) error {
			configFlag, _ := cmd.Parent().PersistentFlags().GetString("config")
			app := cli.NewApp(configFlag)

			result, err := app.ValidateConfig()
			if err != nil {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Error: %v\n", err)
				return errors.New("configuration validation failed")
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), result)
			return nil
		},
	}
}

// buildMainRootCommand creates the main root command with all subcommands.
func buildMainRootCommand() *cobra.Command {
	rootCmd := createMainRootCommand()

	// Add persistent config flag
	rootCmd.PersistentFlags().StringP("config", "c", "bumpers.yml", "Path to config file")

	// Add subcommands
	rootCmd.AddCommand(
		createInstallCommand(),
		createTestCommand(),
		createStatusCommand(),
		createValidateCommand(),
		createClaudeCommandGroup(),
	)

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
