package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/wizzomafizzo/bumpers/internal/logger"
	"github.com/wizzomafizzo/bumpers/internal/project"
)

func main() {
	if err := run(); err != nil {
		// Hooks have special exit code requirements
		var hookErr *HookExitError
		if errors.As(err, &hookErr) {
			os.Exit(hookErr.Code)
		}

		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	workingDir, err := project.FindRoot()
	if err != nil {
		return fmt.Errorf("failed to find project root: %w", err)
	}

	if err := logger.Init(workingDir); err != nil {
		return fmt.Errorf("logger init failed: %w", err)
	}

	if err := createNewRootCommand().Execute(); err != nil {
		return fmt.Errorf("command execution failed: %w", err)
	}
	return nil
}
