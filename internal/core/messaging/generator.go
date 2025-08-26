// Package messaging provides message generation utilities for bumpers.
package messaging

// Generator handles message generation including template processing and AI enhancement
type Generator struct {
	projectRoot string
}

// NewGenerator creates a new message generator
func NewGenerator(projectRoot string) *Generator {
	return &Generator{
		projectRoot: projectRoot,
	}
}
