package config

import (
	"errors"
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ClaudeBinary string `yaml:"claude_binary,omitempty"`
	Rules        []Rule `yaml:"rules"`
}

type Rule struct {
	Pattern      string   `yaml:"pattern"`
	Response     string   `yaml:"response,omitempty"`
	Prompt       string   `yaml:"prompt,omitempty"`
	Alternatives []string `yaml:"alternatives,omitempty"`
	UseClaude    bool     `yaml:"use_claude,omitempty"`
}

func LoadFromYAML(data []byte) (*Config, error) {
	var config Config
	err := yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err //nolint:wrapcheck // YAML parsing errors are self-explanatory
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

func LoadFromFile(filename string) (*Config, error) {
	data, err := os.ReadFile(filename) //nolint:gosec // Config file path is provided by user, not arbitrary
	if err != nil {
		return nil, err //nolint:wrapcheck // File read errors include filename context
	}
	return LoadFromYAML(data)
}

// Validate performs comprehensive config validation
func (c *Config) Validate() error {
	if len(c.Rules) == 0 {
		return errors.New("config must contain at least one rule")
	}

	for i, rule := range c.Rules {
		if err := rule.Validate(); err != nil {
			return fmt.Errorf("rule %d validation failed: %w", i+1, err)
		}
	}

	return nil
}

// Validate performs rule-level validation
func (r *Rule) Validate() error {
	if r.Pattern == "" {
		return errors.New("pattern field is required and cannot be empty")
	}

	// Validate regex pattern
	if _, err := regexp.Compile(r.Pattern); err != nil {
		return fmt.Errorf("invalid regex pattern '%s': %w", r.Pattern, err)
	}

	// If using Claude, ensure proper configuration
	if r.UseClaude {
		if r.Prompt == "" {
			return errors.New("prompt field is required when use_claude is true")
		}
	}

	// Ensure at least one response mechanism is available
	if r.Response == "" && !r.UseClaude && len(r.Alternatives) == 0 {
		return errors.New("rule must provide either a response, alternatives, or use_claude configuration")
	}

	return nil
}
