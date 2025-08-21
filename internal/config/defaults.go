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
				Response: `Use "make test" instead for TDD integration:
- make test                        # Run all tests
- make test PKG=./internal/claude  # Test only a specific package
See Makefile for more information.`,
			},
			{
				Pattern:  "^(gci|go vet|goimports|gofumpt|go fmt)",
				Response: `Use "make lint-fix" instead to resolve lint/formatting issues.`,
			},
			{
				Pattern:  "^cd /tmp",
				Response: `Create a "tmp" directory in the project root instead.`,
			},
		},
		Logging: LoggingConfig{
			Level:      "info",
			MaxSize:    10,
			MaxBackups: 3,
			MaxAge:     30,
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
