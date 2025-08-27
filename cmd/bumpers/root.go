package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wizzomafizzo/bumpers/internal/cli"
)

// createNewRootCommand creates the main root command that shows help by default.
func createNewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "bumpers",
		Short: "Claude Code hook guard",
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Show help when run without subcommands
			return cmd.Help()
		},
	}

	// Add persistent config flag
	rootCmd.PersistentFlags().StringP("config", "c", "bumpers.yml", "Path to config file")

	// Add subcommands
	rootCmd.AddCommand(
		createHookCommand(),
		createInstallCommand(),
		createRulesCommand(),
		createStatusCommand(),
		createValidateCommand(),
	)

	return rootCmd
}

// createAppFromCommand extracts config path and creates a CLI app
func createAppFromCommand(cmd *cobra.Command) (*cli.App, error) {
	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return nil, fmt.Errorf("failed to get config flag: %w", err)
	}

	app := cli.NewApp(configPath)
	return app, nil
}
