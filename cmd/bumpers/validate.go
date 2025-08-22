package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wizzomafizzo/bumpers/internal/cli"
)

// createValidateCommand creates the validate command.
func createValidateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration file",
		Long:  "Validate configuration file",
		RunE: func(cmd *cobra.Command, _ []string) error {
			configPath, err := cmd.Flags().GetString("config")
			if err != nil {
				return fmt.Errorf("config flag error: %w", err)
			}

			app := cli.NewApp(configPath)
			result, err := app.ValidateConfig()
			if err != nil {
				return fmt.Errorf("validation error: %w", err)
			}

			_, _ = fmt.Print(result)
			return nil
		},
	}
}
