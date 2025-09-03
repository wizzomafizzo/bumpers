package main

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/wizzomafizzo/bumpers/internal/app"
	"github.com/wizzomafizzo/bumpers/internal/logging"
	"github.com/wizzomafizzo/bumpers/internal/project"
)

// HookExitError represents an error with a specific exit code for hook processing
type HookExitError struct {
	Message string
	Code    int
}

func (e *HookExitError) Error() string {
	return e.Message
}

// initLoggingForHook initializes logging context specifically for hook processing
func initLoggingForHook() (context.Context, error) {
	// Detect project root for logging context
	projectRoot, err := project.FindRoot()
	if err != nil {
		// Fall back to current working directory if project root detection fails
		projectRoot = ""
	}

	fs := afero.NewOsFs()
	ctx, err := logging.New(context.Background(), fs, logging.Config{
		ProjectID: projectRoot,
		Level:     logging.DebugLevel,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}
	return ctx, nil
}

// processHookCommand processes hook input and returns ProcessResult with exit code
func processHookCommand(
	ctx context.Context, cliApp *app.App, input io.Reader, _ io.Writer,
) (result app.ProcessResult, exitCode int, err error) {
	// Read input for processing
	inputBytes, err := io.ReadAll(input)
	if err != nil {
		return app.ProcessResult{}, 1, fmt.Errorf("failed to read hook input: %w", err)
	}

	result, err = cliApp.ProcessHook(ctx, bytes.NewReader(inputBytes))
	if err != nil {
		return app.ProcessResult{}, 1, fmt.Errorf("failed to process hook: %w", err)
	}

	// Use structured ProcessResult to determine exit code
	switch result.Mode {
	case app.ProcessModeAllow:
		return result, 0, nil
	case app.ProcessModeInformational:
		return result, 0, nil
	case app.ProcessModeBlock:
		return result, 2, nil
	default:
		// Fallback for unknown modes
		return result, 1, fmt.Errorf("unknown process mode: %v", result.Mode)
	}
}

// runHookCommand handles the main execution logic for the hook command.
func runHookCommand(cmd *cobra.Command, _ []string) error {
	// Initialize logging context specifically for hook processing
	ctx, err := initLoggingForHook()
	if err != nil {
		return fmt.Errorf("logger init failed: %w", err)
	}

	cliApp, err := createAppFromCommand(ctx, cmd.Parent())
	if err != nil {
		return err
	}

	result, exitCode, err := processHookCommand(ctx, cliApp, cmd.InOrStdin(), cmd.ErrOrStderr())
	if err != nil {
		return err
	}

	// Output message based on ProcessResult mode instead of brittle string parsing
	if result.Message != "" && exitCode == 0 {
		// Only output informational messages (hookSpecificOutput)
		if result.Mode == app.ProcessModeInformational {
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
