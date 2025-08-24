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
	Session  []Session `yaml:"session,omitempty" mapstructure:"session"`
}

type Generate struct {
	Mode   string `yaml:"mode" mapstructure:"mode"`
	Prompt string `yaml:"prompt" mapstructure:"prompt"`
}

type Rule struct {
	Match    string   `yaml:"match" mapstructure:"match"`
	Tool     string   `yaml:"tool,omitempty" mapstructure:"tool"`
	Send     string   `yaml:"send" mapstructure:"send"`
	Generate Generate `yaml:"generate,omitempty" mapstructure:"generate"`
}

type Command struct {
	Name string `yaml:"name" mapstructure:"name"`
	Send string `yaml:"send" mapstructure:"send"`
}

type Session struct {
	Add string `yaml:"add" mapstructure:"add"`
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
	if len(c.Rules) == 0 && len(c.Commands) == 0 && len(c.Session) == 0 {
		return errors.New("config must contain at least one rule, command, or session")
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
	if r.Match == "" {
		return errors.New("match field is required and cannot be empty")
	}

	// Validate regex pattern
	if _, err := regexp.Compile(r.Match); err != nil {
		return fmt.Errorf("invalid regex pattern '%s': %w", r.Match, err)
	}

	// Validate tools regex pattern if provided
	if r.Tool != "" {
		if _, err := regexp.Compile(r.Tool); err != nil {
			return fmt.Errorf("invalid tools regex pattern '%s': %w", r.Tool, err)
		}
	}

	// Ensure at least one response mechanism is available
	if r.Send == "" && r.Generate.Mode == "" {
		return errors.New("rule must provide either a message or generate configuration")
	}

	// Validate generate mode if provided
	if r.Generate.Mode != "" {
		validModes := []string{"off", "once", "session", "always"}
		isValid := false
		for _, mode := range validModes {
			if r.Generate.Mode == mode {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("invalid generate mode '%s': must be one of: off, once, session, always", r.Generate.Mode)
		}
	}

	return nil
}
