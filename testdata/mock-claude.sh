#!/bin/bash

# Mock Claude CLI script for testing
# Returns JSON responses based on input patterns

# Get the last argument (the prompt)
prompt="${@: -1}"

# Extract key patterns from the prompt
if [[ "$prompt" =~ "just test" ]]; then
    result="Great choice! Using 'just test' provides better integration with TDD guard and coverage reporting."
elif [[ "$prompt" =~ "go test" ]]; then
    result="Consider using 'just test' instead of 'go test' for enhanced testing capabilities with coverage and TDD integration."
elif [[ "$prompt" =~ "rm -rf" ]]; then
    result="For safety, consider using specific file paths instead of recursive removal. This helps prevent accidental data loss."
elif [[ "$prompt" =~ "password\|secret\|key" ]]; then
    result="Avoid hardcoding sensitive information in your code. Consider using environment variables or secure configuration management."
else
    result="Here's a more positive way to frame that guidance: $(echo "$prompt" | tail -c 100)"
fi

# Simulate realistic token usage
input_tokens=$((${#prompt} / 4))
output_tokens=$((${#result} / 4))
duration=$((200 + ${#prompt} / 10))

# Return JSON response matching CLIResponse structure
cat <<EOF
{
  "type": "completion",
  "subtype": "standard", 
  "is_error": false,
  "duration_ms": $duration,
  "duration_api_ms": $((duration - 50)),
  "num_turns": 1,
  "result": "$result",
  "session_id": "mock-session-$(date +%s)",
  "total_cost_usd": 0.001,
  "usage": {
    "input_tokens": $input_tokens,
    "cache_creation_input_tokens": 0,
    "cache_read_input_tokens": 0,
    "output_tokens": $output_tokens,
    "server_tool_use": {
      "web_search_requests": 0
    },
    "service_tier": "pro"
  },
  "permission_denials": [],
  "uuid": "mock-uuid-$(date +%s%N | cut -c1-8)"
}
EOF