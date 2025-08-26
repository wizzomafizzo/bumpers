package testutil

import (
	"io"
	"sync"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var loggerInitOnce sync.Once

// InitTestLogger initializes the test logger to prevent race conditions.
// This should be called from all test files that need logger initialization.
// Implementation moved here to break import cycle with filesystem tests.
func InitTestLogger(t *testing.T) {
	t.Helper()
	loggerInitOnce.Do(func() {
		// Initialize test logger directly without importing logger package
		// This prevents the import cycle: filesystem -> testutil -> logger -> filesystem
		log.Logger = zerolog.New(io.Discard)
	})
}
