package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/wizzomafizzo/bumpers/internal/cli"
)

func main() {
	configPath := "bumpers.yaml"

	rootCmd := &cobra.Command{
		Use:   "bumpers",
		Short: "Claude Code hook guard",
		Run: func(_ *cobra.Command, _ []string) {
			// Default behavior: process hook from stdin
			app := cli.NewApp(configPath)

			response, err := app.ProcessHook(os.Stdin)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			// If there's a response, print it and exit with error code
			if response != "" {
				_, _ = fmt.Print(response)
				os.Exit(1)
			}

			// No response means command is allowed
			os.Exit(0)
		},
	}

	testCmd := &cobra.Command{
		Use:   "test [command]",
		Short: "Test a command against current rules",
		Args:  cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			app := cli.NewApp(configPath)

			result, err := app.TestCommand(args[0])
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			_, _ = fmt.Println(result)
		},
	}

	// TODO: this is a stub, it should eventually check for an "active" flag in the config AND confirm
	// at least one rule is loaded and at least one hook is set to run bumpers in the claude config
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Check hook status",
		Run: func(_ *cobra.Command, _ []string) {
			_, _ = fmt.Println("Bumpers guard is active")
			_, _ = fmt.Printf("Config: %s\n", configPath)
		},
	}

	// Add global config flag
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", configPath, "path to configuration file")

	// Add subcommands
	rootCmd.AddCommand(testCmd, statusCmd)

	// Execute
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
