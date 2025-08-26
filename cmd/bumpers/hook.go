package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wizzomafizzo/bumpers/internal/cli"
	"github.com/wizzomafizzo/bumpers/internal/core/messaging/context"
	"github.com/wizzomafizzo/bumpers/internal/infrastructure/logging"
	"github.com/wizzomafizzo/bumpers/internal/infrastructure/project"
)

// HookExitError represents an error with a specific exit code for hook processing
type HookExitError struct {
	Message string
	Code    int
}

func (e *HookExitError) Error() string {
	return e.Message
}

// findWorkingDir finds the project working directory
func findWorkingDir() (string, error) {
	root, err := project.FindRoot()
	if err != nil {
		return "", fmt.Errorf("failed to find project root: %w", err)
	}
	return root, nil
}

// initLogging initializes logging for hook commands
func initLogging(workingDir string) error {
	projectCtx := context.New(workingDir)
	if err := logger.InitWithProjectContext(projectCtx); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	return nil
}

// processHookCommand processes hook input and returns exit code and error message
func processHookCommand(app *cli.App, input io.Reader, _ io.Writer) (code int, response string) {
	// Read input for processing
	inputBytes, err := io.ReadAll(input)
	if err != nil {
		return 1, fmt.Sprintf("Error reading input: %v", err)
	}

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
			// Initialize logging only for hook command
			workingDir, err := findWorkingDir()
			if err != nil {
				return fmt.Errorf("failed to find project root: %w", err)
			}

			if initErr := initLogging(workingDir); initErr != nil {
				return fmt.Errorf("logger init failed: %w", initErr)
			}

			app, err := createAppFromCommand(cmd.Parent())
			if err != nil {
				return err
			}

			exitCode, message := processHookCommand(app, cmd.InOrStdin(), cmd.ErrOrStderr())

			if message != "" && exitCode == 0 {
				// Only output non-blocking messages (hookSpecificOutput)
				if strings.Contains(message, "hookEventName") {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", message)
				}
			}

			if exitCode != 0 {
				return &HookExitError{Code: exitCode, Message: message}
			}

			return nil
		},
	}
}
