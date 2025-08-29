package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// createInstallCommand creates the install command.
func createInstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install bumpers configuration and Claude hooks",
		Long:  "Install bumpers configuration and Claude hooks",
		RunE: func(cmd *cobra.Command, _ []string) error {
			app, err := createAppFromCommand(cmd.Context(), cmd.Parent())
			if err != nil {
				return err
			}

			err = app.Initialize()
			if err != nil {
				return fmt.Errorf("failed to initialize: %w", err)
			}
			return nil
		},
	}
}
