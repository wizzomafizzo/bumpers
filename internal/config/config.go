package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ClaudeBinary string `yaml:"claude_binary,omitempty"`
	Rules        []Rule `yaml:"rules"`
}

type Rule struct {
	Name         string   `yaml:"name"`
	Pattern      string   `yaml:"pattern"`
	Action       string   `yaml:"action"`
	Message      string   `yaml:"message,omitempty"`
	ClaudePrompt string   `yaml:"claude_prompt,omitempty"`
	Alternatives []string `yaml:"alternatives,omitempty"`
	UseClaude    bool     `yaml:"use_claude,omitempty"`
}

type RuleAction string

const (
	ActionAllow RuleAction = "allow"
	ActionDeny  RuleAction = "deny"
)

func LoadFromYAML(data []byte) (*Config, error) {
	var config Config
	err := yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err //nolint:wrapcheck // YAML parsing errors are self-explanatory
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
