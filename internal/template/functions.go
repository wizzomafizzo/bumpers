package template

import (
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"
	"unicode/utf8"

	"github.com/wizzomafizzo/bumpers/internal/filesystem"
	"github.com/wizzomafizzo/bumpers/internal/project"
)

// createFuncMap creates a function map with custom template functions
func createFuncMap(fs filesystem.FileSystem, commandCtx *CommandContext) template.FuncMap {
	return template.FuncMap{
		"readFile": func(filename string) string {
			return readFile(fs, filename)
		},
		"testPath": func(filename string) bool {
			return testPath(fs, filename)
		},
		"argc": func() int {
			return argc(commandCtx)
		},
		"argv": func(index int) string {
			return argv(commandCtx, index)
		},
	}
}

// readFile securely reads a file from within the project root
// Returns empty string if file doesn't exist, is outside project, or on error
// Text files returned as-is, binary files returned as base64 data URI
func readFile(fs filesystem.FileSystem, filename string) string {
	// Get project root
	projectRoot, err := project.FindRoot()
	if err != nil {
		return ""
	}

	// Clean the filename to prevent directory traversal
	cleanFilename := filepath.Clean(filename)

	// Convert to absolute path within project root
	fullPath := filepath.Join(projectRoot, cleanFilename)

	// Resolve any symlinks and get absolute path
	resolvedPath, err := filepath.Abs(fullPath)
	if err != nil {
		return ""
	}

	// Ensure the resolved path is still within the project root
	resolvedProjectRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return ""
	}

	// Check if resolved path is within project root
	if !strings.HasPrefix(resolvedPath, resolvedProjectRoot+string(filepath.Separator)) &&
		resolvedPath != resolvedProjectRoot {
		return ""
	}

	// Read the file
	content, err := fs.ReadFile(resolvedPath)
	if err != nil {
		return ""
	}

	// Check if content is valid UTF-8 (text file)
	if utf8.Valid(content) {
		return string(content)
	}

	// Binary file - encode as base64 data URI
	encoded := base64.StdEncoding.EncodeToString(content)
	return fmt.Sprintf("data:application/octet-stream;base64,%s", encoded)
}

// testPath securely checks if a file or directory exists within the project root
// Returns false if file doesn't exist, is outside project, or on error
func testPath(fs filesystem.FileSystem, filename string) bool {
	// Get project root
	projectRoot, err := project.FindRoot()
	if err != nil {
		return false
	}

	// Clean the filename to prevent directory traversal
	cleanFilename := filepath.Clean(filename)

	// Convert to absolute path within project root
	fullPath := filepath.Join(projectRoot, cleanFilename)

	// Resolve any symlinks and get absolute path
	resolvedPath, err := filepath.Abs(fullPath)
	if err != nil {
		return false
	}

	// Ensure the resolved path is still within the project root
	resolvedProjectRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return false
	}

	// Check if resolved path is within project root
	if !strings.HasPrefix(resolvedPath, resolvedProjectRoot+string(filepath.Separator)) &&
		resolvedPath != resolvedProjectRoot {
		return false
	}

	// Check if the file or directory exists
	_, err = fs.Stat(resolvedPath)
	return err == nil
}

// argc returns the count of arguments (excluding command name)
func argc(ctx *CommandContext) int {
	if ctx == nil || ctx.Argv == nil {
		return 0
	}
	// Return length minus 1 to exclude command name at index 0
	return len(ctx.Argv) - 1
}

// argv returns the argument at the specified index
// Index 0 returns command name, index 1+ returns actual arguments
// Returns empty string for out-of-bounds access
func argv(ctx *CommandContext, index int) string {
	if ctx == nil || ctx.Argv == nil || index < 0 || index >= len(ctx.Argv) {
		return ""
	}
	return ctx.Argv[index]
}
