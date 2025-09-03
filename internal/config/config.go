package config

import (
	"errors"
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
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

// Match represents the match configuration for a rule
type Match struct {
	Pattern string   `yaml:"pattern" mapstructure:"pattern"`
	Event   string   `yaml:"event,omitempty" mapstructure:"event"`
	Sources []string `yaml:"sources,omitempty" mapstructure:"sources"`
}

type Rule struct {
	Generate any    `yaml:"generate,omitempty" mapstructure:"generate"`
	Match    any    `yaml:"match" mapstructure:"match"`
	Tool     string `yaml:"tool,omitempty" mapstructure:"tool"`
	Send     string `yaml:"send" mapstructure:"send"`
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
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// LoadFromYAML loads config from YAML bytes - helper for tests
func LoadFromYAML(data []byte) (*Config, error) {
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
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

	for i := range c.Rules {
		if err := c.Rules[i].Validate(); err != nil {
			return fmt.Errorf("rule %d validation failed: %w", i+1, err)
		}
	}

	return nil
}

// Validate performs rule-level validation
func (r *Rule) Validate() error {
	if err := r.validateRequiredFields(); err != nil {
		return err
	}
	if err := r.validateRegexPatterns(); err != nil {
		return err
	}
	if err := r.validateResponseMechanism(); err != nil {
		return err
	}
	if err := r.validateGenerateMode(); err != nil {
		return err
	}
	if err := r.validateEventValue(); err != nil {
		return fmt.Errorf("event validation failed: %w", err)
	}
	return nil
}

func (r *Rule) validateRequiredFields() error {
	if r.Match == nil {
		return errors.New("match field is required and cannot be empty")
	}
	match := r.GetMatch()
	if match.Pattern == "" {
		return errors.New("match field is required and cannot be empty")
	}
	return nil
}

func (r *Rule) validateRegexPatterns() error {
	match := r.GetMatch()
	if _, err := regexp.Compile(match.Pattern); err != nil {
		return fmt.Errorf("invalid regex pattern '%s': %w", match.Pattern, err)
	}
	if r.Tool != "" {
		if _, err := regexp.Compile(r.Tool); err != nil {
			return fmt.Errorf("invalid tools regex pattern '%s': %w", r.Tool, err)
		}
	}
	return nil
}

func (r *Rule) validateResponseMechanism() error {
	generate := r.GetGenerate()
	if r.Send == "" && generate.Mode == "off" {
		return errors.New("rule must provide either a message or generate configuration")
	}
	return nil
}

func (r *Rule) validateGenerateMode() error {
	generate := r.GetGenerate()
	if generate.Mode == "" {
		return nil
	}
	validModes := []string{"off", "once", "session", "always"}
	for _, mode := range validModes {
		if generate.Mode == mode {
			return nil
		}
	}
	return fmt.Errorf("invalid generate mode '%s': must be one of: off, once, session, always", generate.Mode)
}

// validateEventValue validates the event field in the match configuration
func (r *Rule) validateEventValue() error {
	match := r.GetMatch()

	// Validate event value (should be pre or post)
	if match.Event != "pre" && match.Event != "post" {
		return fmt.Errorf("invalid event '%s': must be 'pre' or 'post'", match.Event)
	}

	// No source validation - any source name is valid
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

// GetMatch converts the interface{} Match field to a Match struct
func (r *Rule) GetMatch() Match {
	if r.Match == nil {
		return Match{Pattern: "", Event: "pre", Sources: []string{}}
	}

	// Handle string form: match: "pattern"
	if patternStr, ok := r.Match.(string); ok {
		return Match{
			Pattern: patternStr,
			Event:   "pre",      // Default event
			Sources: []string{}, // Default sources (matches all fields)
		}
	}

	// Handle struct form: match: { pattern: "...", event: "...", sources: [...] }
	if matchMap, ok := r.Match.(map[string]any); ok {
		return parseMatchFromMap(matchMap)
	}

	// Invalid match field - return empty Match that will fail validation
	return Match{Pattern: "", Event: "pre", Sources: []string{}}
}

// parseMatchFromMap parses a match field from a map structure
func parseMatchFromMap(matchMap map[string]any) Match {
	match := Match{
		Event:   "pre",      // Default event
		Sources: []string{}, // Default sources
	}

	if pattern, ok := matchMap["pattern"].(string); ok {
		match.Pattern = pattern
	}

	if event, ok := matchMap["event"].(string); ok {
		match.Event = event
	}

	if sources, ok := matchMap["sources"].([]any); ok {
		convertedSources, err := convertSourcesSlice(sources)
		if err != nil {
			// Return invalid Match that will fail validation
			return Match{Pattern: "", Event: "pre", Sources: []string{}}
		}
		match.Sources = convertedSources
	}

	return match
}

// LoadPartial loads config from YAML bytes with partial parsing support
func LoadPartial(data []byte) (*PartialConfig, error) {
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
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
	for i := range c.Rules {
		rule := &c.Rules[i]
		if err := rule.Validate(); err != nil {
			warnings = append(warnings, ValidationWarning{
				RuleIndex: i,
				Rule:      *rule,
				Error:     err,
			})
		} else {
			validRules = append(validRules, *rule)
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

// convertSourcesSlice converts []any sources to []string
func convertSourcesSlice(sources []any) ([]string, error) {
	result := make([]string, len(sources))
	for i, source := range sources {
		sourceStr, ok := source.(string)
		if !ok {
			return nil, fmt.Errorf("source at index %d is not a string: %T", i, source)
		}
		result[i] = sourceStr
	}
	return result, nil
}

// Save writes config to file with proper YAML formatting
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// AddRule appends a rule to the config
func (c *Config) AddRule(rule Rule) {
	c.Rules = append(c.Rules, rule)
}

// DeleteRule removes a rule at the specified index
func (c *Config) DeleteRule(index int) error {
	if index < 0 || index >= len(c.Rules) {
		return fmt.Errorf("invalid index %d: must be between 1 and %d", index+1, len(c.Rules))
	}

	// Remove rule at index by slicing around it
	c.Rules = append(c.Rules[:index], c.Rules[index+1:]...)
	return nil
}

// UpdateRule replaces a rule at the specified index
func (c *Config) UpdateRule(index int, rule Rule) error {
	if index < 0 || index >= len(c.Rules) {
		return fmt.Errorf("invalid index %d: must be between 1 and %d", index+1, len(c.Rules))
	}

	c.Rules[index] = rule
	return nil
}
