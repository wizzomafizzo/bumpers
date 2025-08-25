package testutil

import (
	"strings"
	"testing"
)

func TestLoadTestdataFile(t *testing.T) {
	t.Parallel()

	content := LoadTestdataFile(t, "configs/basic-rule.yaml")
	if len(content) == 0 {
		t.Error("Expected non-empty content")
	}

	yamlStr := string(content)
	if !strings.Contains(yamlStr, "rules:") {
		t.Error("Expected YAML to contain rules")
	}
}
