package main

import (
	"fmt"
	"os"

	"github.com/wizzomafizzo/bumpers/internal/logger"
	"github.com/wizzomafizzo/bumpers/internal/project"
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	workingDir, err := project.FindRoot()
	if err != nil {
		return err
	}

	if err := logger.Init(workingDir); err != nil {
		return err
	}

	return buildMainRootCommand().Execute()
}
