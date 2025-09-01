package cli

import (
	"context"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"github.com/wizzomafizzo/bumpers/internal/config"
)

// Test helpers and factories

// mockMessageGenerator implements MessageGenerator for testing
type mockMessageGenerator struct {
	response  string
	callCount int
}

func (m *mockMessageGenerator) GenerateMessage(_ context.Context, _ string) (string, error) {
	m.callCount++
	return m.response, nil
}

// mockGenerateConfig implements GenerateConfig for testing
type mockGenerateConfig struct {
	generate config.Generate
}

func (m *mockGenerateConfig) GetGenerate() config.Generate {
	return m.generate
}

// Test factories
func newMockMessageGenerator(response string) *mockMessageGenerator {
	return &mockMessageGenerator{response: response}
}

func newMockGenerateConfig(mode, prompt string) *mockGenerateConfig {
	return &mockGenerateConfig{
		generate: config.Generate{
			Mode:   mode,
			Prompt: prompt,
		},
	}
}

func TestNewAIHelper(t *testing.T) {
	t.Parallel()

	mockGen := newMockMessageGenerator("test")
	fs := afero.NewMemMapFs()

	opts := AIHelperOptions{
		Generator:   mockGen,
		FileSystem:  fs,
		CachePath:   "/test/cache",
		ProjectRoot: "/test/root",
	}

	helper := NewAIHelper(opts)

	require.NotNil(t, helper)
	require.Equal(t, "/test/cache", helper.cachePath)
	require.Equal(t, "/test/root", helper.projectRoot)
	require.Equal(t, mockGen, helper.aiGenerator)
	require.Equal(t, fs, helper.fileSystem)
}

func TestNewAIHelper_AllOptions(t *testing.T) {
	t.Parallel()

	mockGen := newMockMessageGenerator("test")
	fs := afero.NewMemMapFs()

	opts := AIHelperOptions{
		Generator:   mockGen,
		FileSystem:  fs,
		CachePath:   "/test/cache",
		ProjectRoot: "/test/root",
	}

	helper := NewAIHelper(opts)

	require.NotNil(t, helper)
	require.Equal(t, "/test/cache", helper.cachePath)
	require.Equal(t, "/test/root", helper.projectRoot)
	require.Equal(t, mockGen, helper.aiGenerator)
	require.Equal(t, fs, helper.fileSystem)
}

func TestAIHelper_getFileSystem_Injected(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	helper := NewAIHelper(AIHelperOptions{FileSystem: fs})

	result := helper.getFileSystem()

	require.Equal(t, fs, result)
}

func TestAIHelper_getFileSystem_Default(t *testing.T) {
	t.Parallel()

	helper := NewAIHelper(AIHelperOptions{})

	result := helper.getFileSystem()

	// Should return OS filesystem
	require.NotNil(t, result)
	require.IsType(t, &afero.OsFs{}, result)
}

func TestProcessAIGenerationGeneric_ModeOff(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockGen := newMockMessageGenerator("should not be called")
	helper := NewAIHelper(AIHelperOptions{Generator: mockGen})

	generateConfig := newMockGenerateConfig("off", "")

	result, err := helper.ProcessAIGenerationGeneric(ctx, generateConfig, "original message", "pattern")

	require.NoError(t, err)
	require.Equal(t, "original message", result)
	require.Equal(t, 0, mockGen.callCount, "AI generator should not be called when mode is off")
}
