package constants

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDatabaseFilename(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "bumpers.db", DatabaseFilename)
}
