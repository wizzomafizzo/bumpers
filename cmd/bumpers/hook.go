package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wizzomafizzo/bumpers/internal/cli"
)

// HookExitError represents an error with a specific exit code for hook processing
type HookExitError struct {
	Message string
	Code    int
}

func (e *HookExitError) Error() string {
	return e.Message
}

// processHookCommand processes hook input and returns exit code and error message
func processHookCommand(configPath string, input io.Reader, _ io.Writer) (code int, response string) {
	// Read input for processing
	inputBytes, err := io.ReadAll(input)
	if err != nil {
		return 1, fmt.Sprintf("Error reading input: %v", err)
	}

	app := cli.NewApp(configPath)

	// Create a new reader from the bytes we just read
	response, err = app.ProcessHook(strings.NewReader(string(inputBytes)))
	if err != nil {
		return 1, fmt.Sprintf("Error: %v", err)
	}

	if response != "" {
		// Check if response is hookSpecificOutput format (should exit 0 and print to stdout)
		if strings.Contains(response, "hookEventName") {
			return 0, response
		}
		// Otherwise it's a blocking response (exit code 2)
		return 2, response
	}

	return 0, ""
}

// createHookCommand creates the hook processing command.
func createHookCommand() *cobra.Command {
	return &cobra.Command{
		Use:          "hook",
		Short:        "Process hook input from Claude Code",
		Long:         "Process hook input from Claude Code and apply configured rules",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			configFlag, _ := cmd.Parent().PersistentFlags().GetString("config")

			exitCode, message := processHookCommand(configFlag, cmd.InOrStdin(), cmd.ErrOrStderr())

			if message != "" {
				// Check if this is a hookSpecificOutput response that should go to stdout
				if strings.Contains(message, "hookEventName") {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", message)
				} else {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", message)
				}
			}

			if exitCode != 0 {
				return &HookExitError{Code: exitCode, Message: message}
			}

			return nil
		},
	}
}
