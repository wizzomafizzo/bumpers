package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// createValidateCommand creates the validate command.
func createValidateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration file",
		Long:  "Validate configuration file",
		RunE: func(cmd *cobra.Command, _ []string) error {
			app, err := createAppFromCommand(cmd.Context(), cmd.Parent())
			if err != nil {
				return err
			}

			result, err := app.ValidateConfig()
			if err != nil {
				return fmt.Errorf("validation error: %w", err)
			}

			_, _ = fmt.Print(result)
			return nil
		},
	}
}
