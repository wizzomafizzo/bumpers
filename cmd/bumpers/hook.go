package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/wizzomafizzo/bumpers/internal/cli"
	"github.com/wizzomafizzo/bumpers/internal/core/logging"
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

// initLogging initializes logging for hook commands and returns context with logger
func initLogging(workingDir string) (context.Context, error) {
	fs := afero.NewOsFs()
	ctx, err := logging.New(context.Background(), fs, logging.Config{
		ProjectID: workingDir,
		Level:     logging.DebugLevel,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}
	return ctx, nil
}

// processHookCommand processes hook input and returns ProcessResult with exit code
func processHookCommand(
	ctx context.Context, app *cli.App, input io.Reader, _ io.Writer,
) (result cli.ProcessResult, exitCode int, err error) {
	// Read input for processing
	inputBytes, err := io.ReadAll(input)
	if err != nil {
		return cli.ProcessResult{}, 1, fmt.Errorf("error reading input: %w", err)
	}

	result, err = app.ProcessHook(ctx, strings.NewReader(string(inputBytes)))
	if err != nil {
		return cli.ProcessResult{}, 1, fmt.Errorf("error: %w", err)
	}

	// Use structured ProcessResult to determine exit code
	switch result.Mode {
	case cli.ProcessModeAllow:
		return result, 0, nil
	case cli.ProcessModeInformational:
		return result, 0, nil
	case cli.ProcessModeBlock:
		return result, 2, nil
	default:
		// Fallback for unknown modes
		return result, 1, fmt.Errorf("unknown process mode: %v", result.Mode)
	}
}

// runHookCommand handles the main execution logic for the hook command.
func runHookCommand(cmd *cobra.Command, _ []string) error {
	// Initialize logging only for hook command
	workingDir, err := findWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to find project root: %w", err)
	}

	ctx, err := initLogging(workingDir)
	if err != nil {
		return fmt.Errorf("logger init failed: %w", err)
	}

	app, err := createAppFromCommand(ctx, cmd.Parent())
	if err != nil {
		return err
	}

	result, exitCode, err := processHookCommand(ctx, app, cmd.InOrStdin(), cmd.ErrOrStderr())
	if err != nil {
		return err
	}

	// Output message based on ProcessResult mode instead of brittle string parsing
	if result.Message != "" && exitCode == 0 {
		// Only output informational messages (hookSpecificOutput)
		if result.Mode == cli.ProcessModeInformational {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", result.Message)
		}
	}

	if exitCode != 0 {
		return &HookExitError{Code: exitCode, Message: result.Message}
	}

	return nil
}

// createHookCommand creates the hook processing command.
func createHookCommand() *cobra.Command {
	return &cobra.Command{
		Use:          "hook",
		Short:        "Process hook input from Claude Code",
		Long:         "Process hook input from Claude Code and apply configured rules",
		SilenceUsage: true,
		RunE:         runHookCommand,
	}
}
