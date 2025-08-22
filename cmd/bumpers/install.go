package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wizzomafizzo/bumpers/internal/cli"
)

// createInstallCommand creates the install command.
func createInstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install bumpers configuration and Claude hooks",
		Long:  "Install bumpers configuration and Claude hooks",
		RunE: func(cmd *cobra.Command, _ []string) error {
			configPath, err := cmd.Flags().GetString("config")
			if err != nil {
				return fmt.Errorf("failed to get config flag: %w", err)
			}

			app := cli.NewApp(configPath)
			err = app.Initialize()
			if err != nil {
				return fmt.Errorf("failed to initialize: %w", err)
			}
			return nil
		},
	}
}
