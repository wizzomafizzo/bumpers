package engine

import (
	"testing"

	testutil "github.com/wizzomafizzo/bumpers/internal/testing"
)

func TestNewProcessor(t *testing.T) {
	t.Parallel()

	configPath := "/test/config.yml"
	projectRoot := "/test/project"

	processor := NewProcessor(configPath, projectRoot)

	testutil.AssertNotNil(t, processor, "processor should not be nil")
	testutil.AssertEqualMsg(t, configPath, processor.configPath, "configPath should be set correctly")
	testutil.AssertEqualMsg(t, projectRoot, processor.projectRoot, "projectRoot should be set correctly")
}
