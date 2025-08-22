package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wizzomafizzo/bumpers/internal/cli"
)

// createStatusCommand creates the status command.
func createStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check hook status",
		Long:  "Check hook status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			configPath, err := cmd.Flags().GetString("config")
			if err != nil {
				return fmt.Errorf("failed to get config flag: %w", err)
			}

			app := cli.NewApp(configPath)
			status, err := app.Status()
			if err != nil {
				return fmt.Errorf("failed to get status: %w", err)
			}

			_, err = fmt.Print(status)
			if err != nil {
				return fmt.Errorf("failed to print status: %w", err)
			}
			return nil
		},
	}
}
