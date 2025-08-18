package config

import (
	"os"
	
	"gopkg.in/yaml.v3"
)

type Config struct {
	Rules []Rule `yaml:"rules"`
}

type Rule struct {
	Name         string   `yaml:"name"`
	Pattern      string   `yaml:"pattern"`
	Action       string   `yaml:"action"`
	Message      string   `yaml:"message,omitempty"`
	Alternatives []string `yaml:"alternatives,omitempty"`
	UseClaude    bool     `yaml:"use_claude,omitempty"`
	ClaudePrompt string   `yaml:"claude_prompt,omitempty"`
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
		return nil, err
	}
	return &config, nil
}

func LoadFromFile(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return LoadFromYAML(data)
}