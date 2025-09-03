package logging

import (
	"context"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helpers
func createTestConfig(writer *strings.Builder) Config {
	return Config{
		Writer:    writer,
		ProjectID: "test-project",
		Level:     InfoLevel,
	}
}

func TestGet_WithoutLogger(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	logger := Get(ctx)

	require.NotNil(t, logger)
	// When no logger is attached, zerolog.Ctx returns a disabled logger
	require.Equal(t, zerolog.Disabled, logger.GetLevel())
}

func TestNew_WithCustomWriter(t *testing.T) {
	t.Parallel()

	var buf strings.Builder
	config := createTestConfig(&buf)

	ctx, err := New(context.Background(), nil, config)

	require.NoError(t, err)
	require.NotNil(t, ctx)

	logger := Get(ctx)
	require.NotNil(t, logger)
	assert.Equal(t, InfoLevel, logger.GetLevel())
}

func TestNew_NoWriterNoFilesystem_ReturnsError(t *testing.T) {
	t.Parallel()

	config := Config{
		Writer:    nil, // No writer provided
		ProjectID: "test-project",
		Level:     InfoLevel,
	}

	ctx, err := New(context.Background(), nil, config) // No filesystem provided

	require.Error(t, err)
	assert.Contains(t, err.Error(), "filesystem required when no writer provided")
	assert.Nil(t, ctx)
}
