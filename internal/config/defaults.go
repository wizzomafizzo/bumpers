package config

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// DefaultConfig returns the default bumpers configuration
func DefaultConfig() *Config {
	return &Config{
		Rules: []Rule{
			{
				Pattern: "^go test",
				Message: `Use "just test" instead for TDD integration:
- just test                        # Run all tests
- just test PKG=./internal/claude  # Test only a specific package
See justfile for more information.`,
			},
			{
				Pattern: "^(gci|go vet|goimports|gofumpt|go fmt)",
				Message: `Use "just lint fix" instead to resolve lint/formatting issues.`,
			},
			{
				Pattern: "^cd /tmp",
				Message: `Create a "tmp" directory in the project root instead.`,
			},
		},
		Commands: []Command{
			{
				Message: "Available commands:\\n!help - Show this help\\n!status - Show project status\\n" +
					"!docs - Open documentation",
			},
			{
				Message: "Project Status: All systems operational",
			},
			{
				Message: "📚 Documentation: Visit https://github.com/wizzomafizzo/bumpers for usage guides and examples",
			},
		},
		Notes: []Note{
			{
				Message: "Use 'just test' instead of 'go test' for proper TDD integration",
			},
			{
				Message: "Check CLAUDE.md for project conventions and guidelines",
			},
			{
				Message: "Run 'just lint fix' to resolve formatting and linting issues",
			},
		},
	}
}

// DefaultConfigYAML returns the default configuration as YAML bytes
func DefaultConfigYAML() ([]byte, error) {
	config := DefaultConfig()
	data, err := yaml.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal default config to YAML: %w", err)
	}
	return data, nil
}
