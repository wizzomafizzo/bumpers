// Package engine provides the core rule processing engine for bumpers.
package engine

// Processor implements the rule processing engine
type Processor struct {
	configPath  string
	projectRoot string
}

// NewProcessor creates a new rule processing engine
func NewProcessor(configPath, projectRoot string) *Processor {
	return &Processor{
		configPath:  configPath,
		projectRoot: projectRoot,
	}
}
