package response

import (
	"github.com/wizzomafizzo/bumpers/internal/config"
)

func FormatResponse(rule *config.Rule) string {
	// Prefer new Response field over old Message field
	if rule.Response != "" {
		return rule.Response
	}
	return rule.Message
}
