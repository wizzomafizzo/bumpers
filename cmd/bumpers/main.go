package main

import (
	"fmt"
	"os"
)

func main() {
	// Use the buildMainRootCommand from claude.go which has all commands including init
	rootCmd := buildMainRootCommand()

	// Execute
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
