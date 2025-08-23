package config

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Rules    []Rule    `yaml:"rules,omitempty" mapstructure:"rules"`
	Commands []Command `yaml:"commands,omitempty" mapstructure:"commands"`
	Notes    []Note    `yaml:"notes,omitempty" mapstructure:"notes"`
}

type Rule struct {
	Pattern  string `yaml:"pattern" mapstructure:"pattern"`
	Tools    string `yaml:"tools,omitempty" mapstructure:"tools"`
	Message  string `yaml:"message" mapstructure:"message"`
	Prompt   string `yaml:"prompt,omitempty" mapstructure:"prompt"`
	Generate string `yaml:"generate,omitempty" mapstructure:"generate"`
}

type Command struct {
	Name    string `yaml:"name" mapstructure:"name"`
	Message string `yaml:"message" mapstructure:"message"`
}

type Note struct {
	Message string `yaml:"message" mapstructure:"message"`
}

func Load(path string) (*Config, error) {
	viperInstance := viper.New()
	viperInstance.SetConfigFile(path)

	if err := viperInstance.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config Config
	if err := viperInstance.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// LoadFromYAML loads config from YAML bytes - helper for tests
func LoadFromYAML(data []byte) (*Config, error) {
	viperInstance := viper.New()
	viperInstance.SetConfigType("yaml")

	if err := viperInstance.ReadConfig(strings.NewReader(string(data))); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config Config
	if err := viperInstance.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// Validate performs comprehensive config validation
func (c *Config) Validate() error {
	if len(c.Rules) == 0 && len(c.Commands) == 0 && len(c.Notes) == 0 {
		return errors.New("config must contain at least one rule, command, or note")
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

	// Validate tools regex pattern if provided
	if r.Tools != "" {
		if _, err := regexp.Compile(r.Tools); err != nil {
			return fmt.Errorf("invalid tools regex pattern '%s': %w", r.Tools, err)
		}
	}

	// Ensure at least one response mechanism is available
	if r.Message == "" && r.Generate == "" {
		return errors.New("rule must provide either a message or generate configuration")
	}

	// Validate generate mode if provided
	if r.Generate != "" {
		validModes := []string{"off", "once", "session", "always"}
		isValid := false
		for _, mode := range validModes {
			if r.Generate == mode {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("invalid generate mode '%s': must be one of: off, once, session, always", r.Generate)
		}
	}

	return nil
}
