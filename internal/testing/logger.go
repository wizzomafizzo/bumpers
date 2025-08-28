package testutil

import (
	"context"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

// NewTestContext creates a context with a logger that captures output for a single test.
// Returns the context and a function to get captured output.
// Safe for parallel tests - no global state modification.
func NewTestContext(t *testing.T) (ctx context.Context, getLogOutput func() string) {
	t.Helper()
	var logOutput strings.Builder
	syncWriter := zerolog.SyncWriter(&logOutput)
	logger := zerolog.New(syncWriter).Level(zerolog.DebugLevel)

	ctx = logger.WithContext(context.Background())

	return ctx, func() string { //nolint:gocritic // unlambda: closure needed for deferred string capture
		return logOutput.String()
	}
}
