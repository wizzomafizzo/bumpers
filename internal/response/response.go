package response

import (
	"github.com/wizzomafizzo/bumpers/internal/config"
)

func FormatResponse(rule *config.Rule) string {
	return rule.Response
}
