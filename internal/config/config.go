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
	return parseMatchField(r.Match, "", nil)
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

// parseMatchField converts an interface{} Match field to a Match struct
func parseMatchField(matchField any, fallbackEvent string, fallbackSources []string) Match {
	if matchField == nil {
		return parseNilMatch()
	}

	if patternStr, ok := matchField.(string); ok {
		return parseStringMatch(patternStr, fallbackEvent, fallbackSources)
	}

	if matchMap, ok := matchField.(map[string]any); ok {
		return parseMapMatch(matchMap)
	}

	return parseFallbackMatch(fallbackEvent, fallbackSources)
}

// parseNilMatch handles nil match field case
func parseNilMatch() Match {
	return Match{Pattern: "", Event: "pre", Sources: []string{}}
}

// parseStringMatch handles string match field case
func parseStringMatch(patternStr, fallbackEvent string, fallbackSources []string) Match {
	event := fallbackEvent
	if event == "" {
		event = "pre"
	}

	sources := determineSources(fallbackSources, fallbackEvent)
	return Match{Pattern: patternStr, Event: event, Sources: sources}
}

// parseMapMatch handles map match field case
func parseMapMatch(matchMap map[string]any) Match {
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
		match.Sources = convertSourcesSlice(sources)
	}

	return match
}

// parseFallbackMatch handles unknown match field types
func parseFallbackMatch(fallbackEvent string, fallbackSources []string) Match {
	event := fallbackEvent
	if event == "" {
		event = "pre"
	}

	sources := fallbackSources
	if sources == nil {
		sources = []string{}
	}

	return Match{Pattern: "", Event: event, Sources: sources}
}

// determineSources determines sources slice for string match case
func determineSources(fallbackSources []string, fallbackEvent string) []string {
	if fallbackSources == nil && fallbackEvent == "" {
		// New format: no Event/Sources at rule level, use empty slice
		return []string{}
	}

	if fallbackSources == nil {
		// Old format: preserve nil for backward compatibility
		return nil
	}

	return fallbackSources
}

// convertSourcesSlice converts []any sources to []string
func convertSourcesSlice(sources []any) []string {
	result := make([]string, len(sources))
	for i, source := range sources {
		if sourceStr, ok := source.(string); ok {
			result[i] = sourceStr
		}
	}
	return result
}
