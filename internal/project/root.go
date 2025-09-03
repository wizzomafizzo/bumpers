// Package project provides utilities for detecting project root directories.
package project

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindRoot finds the project root directory.
func FindRoot() (string, error) {
	// Check for Claude project directory first
	if root, found := checkClaudeProjectDir(); found {
		return root, nil
	}

	// Get current working directory as starting point
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Look for project markers
	if root, found := findProjectMarker(cwd); found {
		return root, nil
	}

	// Fall back to current working directory
	return cwd, nil
}

// FindProjectMarkerFrom finds the project root directory starting from the given directory.
// This is useful for testing when you want to find the project root from a specific directory
// rather than the current working directory.
func FindProjectMarkerFrom(startDir string) (string, bool) {
	return findProjectMarker(startDir)
}

// checkClaudeProjectDir checks if CLAUDE_PROJECT_DIR environment variable is set and valid
func checkClaudeProjectDir() (string, bool) {
	claudeDir := os.Getenv("CLAUDE_PROJECT_DIR")
	if claudeDir == "" {
		return "", false
	}

	abs, err := filepath.Abs(claudeDir)
	if err != nil {
		return "", false
	}

	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return "", false
	}

	return abs, true
}

// findProjectMarker searches for project root markers starting from the given directory
func findProjectMarker(startDir string) (string, bool) {
	markers := []string{".git", "go.mod", "package.json"}
	currentDir := startDir

	for {
		if hasProjectMarker(currentDir, markers) {
			return currentDir, true
		}

		// Move to parent directory
		parentDir := filepath.Dir(currentDir)

		// Stop if we've reached the filesystem root
		if parentDir == currentDir {
			break
		}

		currentDir = parentDir
	}

	return "", false
}

// hasProjectMarker checks if any of the given markers exist in the directory
func hasProjectMarker(dir string, markers []string) bool {
	for _, marker := range markers {
		markerPath := filepath.Join(dir, marker)
		if _, err := os.Stat(markerPath); err == nil {
			return true
		}
	}
	return false
}
