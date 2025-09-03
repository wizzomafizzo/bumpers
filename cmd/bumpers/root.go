package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wizzomafizzo/bumpers/internal/app"
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

// createApp creates a CLI app using the factory pattern
func createApp(ctx context.Context, configPath string) (*app.App, error) {
	factory := app.NewAppFactory()
	cliApp := factory.CreateAppWithComponentFactory(ctx, configPath)
	return cliApp, nil
}

// createAppFromCommand extracts config path and creates a CLI app
func createAppFromCommand(ctx context.Context, cmd *cobra.Command) (*app.App, error) {
	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return nil, fmt.Errorf("failed to get config flag: %w", err)
	}

	return createApp(ctx, configPath)
}
