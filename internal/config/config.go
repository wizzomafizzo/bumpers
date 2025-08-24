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

// PartialConfig represents a configuration where some rules may be invalid
type PartialConfig struct {
	Config
	ValidationWarnings []ValidationWarning
}

// ValidationWarning represents a validation error for a specific rule
type ValidationWarning struct {
	Error     error
	Rule      Rule
	RuleIndex int
}

type Generate struct {
	Mode   string `yaml:"mode" mapstructure:"mode"`
	Prompt string `yaml:"prompt" mapstructure:"prompt"`
}

type Rule struct {
	Generate any      `yaml:"generate,omitempty" mapstructure:"generate"`
	Match    string   `yaml:"match" mapstructure:"match"`
	Tool     string   `yaml:"tool,omitempty" mapstructure:"tool"`
	Send     string   `yaml:"send" mapstructure:"send"`
	When     []string `yaml:"when,omitempty" mapstructure:"when"`
}

// ExpandWhen applies smart defaults and exclusion logic to the When field
func (r *Rule) ExpandWhen() []string {
	// Default to pre+input when When field is omitted
	if len(r.When) == 0 {
		return []string{"pre", "input"}
	}

	expanded := make(map[string]bool)
	excludes := make(map[string]bool)

	// First pass: collect base flags and exclusions
	for _, flag := range r.When {
		if strings.HasPrefix(flag, "!") {
			excludes[strings.TrimPrefix(flag, "!")] = true
		} else {
			expanded[flag] = true
		}
	}

	// Second pass: apply smart defaults
	for flag := range expanded {
		switch flag {
		case "reasoning":
			// reasoning implies post
			expanded["post"] = true
		case "post":
			// post implies output if no explicit source flags
			if !hasSourceFlag(r.When) {
				expanded["output"] = true
			}
		case "pre":
			// pre implies input if no explicit source flags
			if !hasSourceFlag(r.When) {
				expanded["input"] = true
			}
		}
	}

	// Third pass: remove excluded flags
	for exclude := range excludes {
		delete(expanded, exclude)
	}

	// Convert map to slice
	result := make([]string, 0, len(expanded))
	for flag := range expanded {
		result = append(result, flag)
	}

	return result
}

// hasSourceFlag checks if the when slice contains any explicit source flags (input, output, reasoning)
// Exclusions (flags starting with !) don't count as explicit inclusion
func hasSourceFlag(when []string) bool {
	sourceFlags := map[string]bool{"input": true, "output": true, "reasoning": true}
	for _, flag := range when {
		if !strings.HasPrefix(flag, "!") && sourceFlags[flag] {
			return true
		}
	}
	return false
}

type Command struct {
	Generate any    `yaml:"generate,omitempty" mapstructure:"generate"`
	Name     string `yaml:"name" mapstructure:"name"`
	Send     string `yaml:"send" mapstructure:"send"`
}

type Session struct {
	Generate any    `yaml:"generate,omitempty" mapstructure:"generate"`
	Add      string `yaml:"add" mapstructure:"add"`
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
	// Since generate now defaults to "session", we only fail if send is empty AND generate is explicitly set to "off"
	generate := r.GetGenerate()
	if r.Send == "" && generate.Mode == "off" {
		return errors.New("rule must provide either a message or generate configuration")
	}

	// Validate generate mode if provided
	if generate.Mode != "" {
		validModes := []string{"off", "once", "session", "always"}
		isValid := false
		for _, mode := range validModes {
			if generate.Mode == mode {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("invalid generate mode '%s': must be one of: off, once, session, always", generate.Mode)
		}
	}

	return nil
}

// GetGenerate converts the interface{} Generate field to a Generate struct
func (r *Rule) GetGenerate() Generate {
	// Handle the special case of Generate struct
	if generateValue, ok := r.Generate.(Generate); ok {
		return generateValue
	}
	return parseGenerateField(r.Generate, "session")
}

// LoadPartial loads config from YAML bytes with partial parsing support
func LoadPartial(data []byte) (*PartialConfig, error) {
	viperInstance := viper.New()
	viperInstance.SetConfigType("yaml")

	if err := viperInstance.ReadConfig(strings.NewReader(string(data))); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config Config
	if err := viperInstance.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Use partial validation to collect errors instead of failing
	validConfig, warnings := config.ValidatePartial()

	return &PartialConfig{
		Config:             validConfig,
		ValidationWarnings: warnings,
	}, nil
}

// ValidatePartial performs validation and returns valid config with warnings for invalid rules
func (c *Config) ValidatePartial() (Config, []ValidationWarning) {
	var validRules []Rule
	var warnings []ValidationWarning

	// Validate each rule separately
	for i, rule := range c.Rules {
		if err := rule.Validate(); err != nil {
			warnings = append(warnings, ValidationWarning{
				RuleIndex: i,
				Rule:      rule,
				Error:     err,
			})
		} else {
			validRules = append(validRules, rule)
		}
	}

	validConfig := Config{
		Rules:    validRules,
		Commands: c.Commands,
		Session:  c.Session,
	}

	return validConfig, warnings
}

// parseGenerateField converts an interface{} Generate field to a Generate struct with given default
func parseGenerateField(generateField any, defaultMode string) Generate {
	if generateField == nil {
		return Generate{Mode: defaultMode}
	}

	if modeStr, ok := generateField.(string); ok {
		return Generate{Mode: modeStr}
	}

	if generateMap, ok := generateField.(map[string]any); ok {
		gen := Generate{}
		if mode, ok := generateMap["mode"].(string); ok {
			gen.Mode = mode
		}
		if prompt, ok := generateMap["prompt"].(string); ok {
			gen.Prompt = prompt
		}
		if gen.Mode == "" {
			gen.Mode = defaultMode
		}
		return gen
	}

	return Generate{Mode: defaultMode}
}

// GetGenerate converts the interface{} Generate field to a Generate struct for Command
func (c *Command) GetGenerate() Generate {
	return parseGenerateField(c.Generate, "off")
}

// GetGenerate converts the interface{} Generate field to a Generate struct for Session
func (s *Session) GetGenerate() Generate {
	return parseGenerateField(s.Generate, "off")
}
