package apptypes

import (
	"context"

	"github.com/wizzomafizzo/bumpers/internal/config"
	"github.com/wizzomafizzo/bumpers/internal/matcher"
)

// ConfigLoader interface for loading configuration
type ConfigLoader interface {
	LoadConfigAndMatcher(ctx context.Context) (*config.Config, *matcher.RuleMatcher, error)
}

// ConfigValidator interface for validating configuration
type ConfigValidator interface {
	ConfigLoader
	ValidateConfig() (string, error)
	TestCommand(ctx context.Context, command string) (string, error)
}
