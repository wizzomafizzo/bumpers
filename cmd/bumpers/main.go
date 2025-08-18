package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/wizzomafizzo/bumpers/internal/cli"
)

func main() {
	configPath := "bumpers.yaml"

	var rootCmd = &cobra.Command{
		Use:   "bumpers",
		Short: "Simple Claude Code hook guard with positive guidance",
		Run: func(cmd *cobra.Command, args []string) {
			// Default behavior: process hook from stdin
			app := cli.NewApp(configPath)
			
			response, err := app.ProcessHook(os.Stdin)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			
			// If there's a response, print it and exit with error code
			if response != "" {
				fmt.Print(response)
				os.Exit(1)
			}
			
			// No response means command is allowed
			os.Exit(0)
		},
	}

	var testCmd = &cobra.Command{
		Use:   "test [command]",
		Short: "Test a command against current rules",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			app := cli.NewApp(configPath)
			
			result, err := app.TestCommand(args[0])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			
			fmt.Println(result)
		},
	}

	var statusCmd = &cobra.Command{
		Use:   "status",
		Short: "Check hook status",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Bumpers guard is active")
			fmt.Printf("Config: %s\n", configPath)
		},
	}

	// Add global config flag
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", configPath, "path to configuration file")

	// Add subcommands  
	rootCmd.AddCommand(testCmd, statusCmd)

	// Execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}