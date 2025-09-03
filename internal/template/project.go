// Package template provides template processing with project context.
package template

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// ProjectContext holds project identification information
type ProjectContext struct {
	ID   string
	Name string
	Path string
}

// New creates a new ProjectContext from a project path
func New(projectPath string) *ProjectContext {
	// Extract project name from path (last component)
	projectName := filepath.Base(projectPath)

	// Generate project ID
	projectID := GenerateProjectID(projectName, projectPath)

	return &ProjectContext{
		Path: projectPath,
		Name: projectName,
		ID:   projectID,
	}
}

// GenerateProjectID creates a deterministic project ID from name and path
func GenerateProjectID(projectName, projectPath string) string {
	// Sanitize project name (keep only alphanumeric, remove special chars including dashes)
	reg := regexp.MustCompile(`[^a-zA-Z0-9]`)
	sanitized := reg.ReplaceAllString(projectName, "")
	sanitized = strings.ToLower(sanitized)

	// Generate short hash from absolute path
	hash := sha256.Sum256([]byte(projectPath))
	shortHash := fmt.Sprintf("%x", hash)[:4]

	return fmt.Sprintf("%s-%s", sanitized, shortHash)
}
