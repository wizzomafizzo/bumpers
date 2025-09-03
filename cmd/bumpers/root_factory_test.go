package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateApp_ShouldUseAppFactory(t *testing.T) {
	t.Parallel()
	// Given
	ctx := context.Background()
	configPath := "test-config.yml"

	// When
	app, err := createApp(ctx, configPath)

	// Then
	require.NoError(t, err)
	assert.NotNil(t, app)
}
