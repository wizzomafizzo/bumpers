package main

import (
	"errors"
	"fmt"
	"os"
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
	if err := createNewRootCommand().Execute(); err != nil {
		return fmt.Errorf("command execution failed: %w", err)
	}
	return nil
}
