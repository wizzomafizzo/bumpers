package main

import (
	"github.com/spf13/cobra"
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
		createStatusCommand(),
		createValidateCommand(),
	)

	return rootCmd
}
