package config

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	ClaudeBinary string        `yaml:"claude_binary,omitempty" mapstructure:"claude_binary"`
	Rules        []Rule        `yaml:"rules" mapstructure:"rules"`
	Logging      LoggingConfig `yaml:"logging,omitempty" mapstructure:"logging"`
}

type LoggingConfig struct {
	Level      string `yaml:"level,omitempty" mapstructure:"level"`             // debug, info, warn, error
	Path       string `yaml:"path,omitempty" mapstructure:"path"`               // custom log file path
	MaxSize    int    `yaml:"max_size,omitempty" mapstructure:"max_size"`       // MB per file
	MaxBackups int    `yaml:"max_backups,omitempty" mapstructure:"max_backups"` // number of old files to keep
	MaxAge     int    `yaml:"max_age,omitempty" mapstructure:"max_age"`         // days to keep old files
}

type Rule struct {
	Pattern      string   `yaml:"pattern" mapstructure:"pattern"`
	Response     string   `yaml:"response,omitempty" mapstructure:"response"`
	Prompt       string   `yaml:"prompt,omitempty" mapstructure:"prompt"`
	Alternatives []string `yaml:"alternatives,omitempty" mapstructure:"alternatives"`
	UseClaude    bool     `yaml:"use_claude,omitempty" mapstructure:"use_claude"`
}

func Load(path string) (*Config, error) {
	viperInstance := viper.New()
	viperInstance.SetConfigFile(path)

	// Set defaults for logging
	viperInstance.SetDefault("logging.level", "info")
	viperInstance.SetDefault("logging.max_size", 10)
	viperInstance.SetDefault("logging.max_backups", 3)
	viperInstance.SetDefault("logging.max_age", 30)

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

	// Set defaults for logging
	viperInstance.SetDefault("logging.level", "info")
	viperInstance.SetDefault("logging.max_size", 10)
	viperInstance.SetDefault("logging.max_backups", 3)
	viperInstance.SetDefault("logging.max_age", 30)

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
