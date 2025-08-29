package testutil

import (
	"context"
	"strings"
	"testing"

	"github.com/rs/zerolog" //nolint:depguard // Test utilities need direct zerolog access
	"github.com/wizzomafizzo/bumpers/internal/core/logging"
)

// NewTestContext creates a context with logger for race-safe testing
// Returns a context with logger attached and a function to retrieve log output
func NewTestContext(t *testing.T) (ctx context.Context, getLogOutput func() string) {
	t.Helper()

	var logOutput strings.Builder
	syncWriter := zerolog.SyncWriter(&logOutput)

	ctx, err := logging.New(context.Background(), nil, logging.Config{
		ProjectID: "test-project",
		Writer:    syncWriter,
		Level:     zerolog.DebugLevel,
	})
	if err != nil {
		t.Fatalf("Failed to create test logger: %v", err)
	}

	return ctx, logOutput.String
}
