package template

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	validTemplate = "Hello {{.Name}}"
)

// Test helpers
func createTemplateOfSize(size int) string {
	return strings.Repeat("a", size)
}

func TestValidateTemplate_ValidSize(t *testing.T) {
	t.Parallel()

	err := ValidateTemplate(validTemplate)

	require.NoError(t, err)
}

func TestValidateTemplate_ExceedsMaxSize(t *testing.T) {
	t.Parallel()

	largeTemplate := createTemplateOfSize(MaxTemplateSize + 1)

	err := ValidateTemplate(largeTemplate)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum allowed size")
}

func TestValidateTemplate_AtMaxSize(t *testing.T) {
	t.Parallel()

	templateAtLimit := createTemplateOfSize(MaxTemplateSize)

	err := ValidateTemplate(templateAtLimit)

	require.NoError(t, err)
}
