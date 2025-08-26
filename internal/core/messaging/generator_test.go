package messaging

import (
	"testing"

	testutil "github.com/wizzomafizzo/bumpers/internal/testing"
)

func TestNewGenerator(t *testing.T) {
	t.Parallel()

	projectRoot := "/test/project"

	generator := NewGenerator(projectRoot)

	testutil.AssertNotNil(t, generator, "generator should not be nil")
	testutil.AssertEqualMsg(t, projectRoot, generator.projectRoot, "projectRoot should be set correctly")
}
