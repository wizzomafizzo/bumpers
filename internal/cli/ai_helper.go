package cli

import (
	"context"
	"fmt"

	"github.com/spf13/afero"
	ai "github.com/wizzomafizzo/bumpers/internal/platform/claude/api"
	"github.com/wizzomafizzo/bumpers/internal/platform/storage"
)

// AIHelper provides shared AI generation functionality
type AIHelper struct {
	aiGenerator ai.MessageGenerator
	fileSystem  afero.Fs
	projectRoot string
}

// NewAIHelper creates a new AI helper
func NewAIHelper(projectRoot string, aiGenerator ai.MessageGenerator, fileSystem afero.Fs) *AIHelper {
	return &AIHelper{
		projectRoot: projectRoot,
		aiGenerator: aiGenerator,
		fileSystem:  fileSystem,
	}
}

// getFileSystem returns the filesystem to use - either injected or defaults to OS
func (h *AIHelper) getFileSystem() afero.Fs {
	if h.fileSystem != nil {
		return h.fileSystem
	}
	return afero.NewOsFs()
}

// ProcessAIGenerationGeneric method that accepts any type with GetGenerate()
func (h *AIHelper) ProcessAIGenerationGeneric(
	ctx context.Context,
	generateConfig GenerateConfig,
	message, pattern string,
) (string, error) {
	generate := generateConfig.GetGenerate()
	// Skip if generation mode is "off"
	if generate.Mode == "off" {
		return message, nil
	}

	// Use XDG-compliant cache path
	storageManager := storage.New(h.getFileSystem())
	cachePath, err := storageManager.GetCachePath()
	if err != nil {
		return message, fmt.Errorf("failed to get cache path: %w", err)
	}

	// Create AI generator with mock launcher if available
	var generator *ai.Generator
	if h.aiGenerator != nil {
		generator, err = ai.NewGeneratorWithLauncher(cachePath, h.projectRoot, h.aiGenerator)
	} else {
		generator, err = ai.NewGenerator(cachePath, h.projectRoot)
	}
	if err != nil {
		return message, fmt.Errorf("failed to create AI generator: %w", err)
	}
	defer func() {
		if closeErr := generator.Close(); closeErr != nil {
			// Log error but don't fail the hook - generator.Close() error is non-critical
			_ = closeErr // Silence linter about empty block
		}
	}()

	// Create request
	req := &ai.GenerateRequest{
		OriginalMessage: message,
		CustomPrompt:    generate.Prompt,
		GenerateMode:    generate.Mode,
		Pattern:         pattern,
	}

	// Generate message
	result, err := generator.GenerateMessage(ctx, req)
	if err != nil {
		return message, fmt.Errorf("failed to generate AI message: %w", err)
	}

	return result, nil
}
