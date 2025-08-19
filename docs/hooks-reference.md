# Hooks reference - Anthropic

[Anthropic home page![light logo](https://mintlify.s3.us-west-1.amazonaws.com/anthropic/logo/light.svg)![dark logo](https://mintlify.s3.us-west-1.amazonaws.com/anthropic/logo/dark.svg)](/)

English

Search...

⌘K

Search...

Navigation

Reference

Hooks reference

[Welcome

](/en/home)[Developer Platform

](/en/docs/intro)[Claude Code

](/en/docs/claude-code/overview)[Model Context Protocol (MCP)

](/en/docs/mcp)[API Reference

](/en/api/messages)[Resources

](/en/resources/overview)[Release Notes

](/en/release-notes/overview)

##### Getting started

- [

  Overview

  ](/en/docs/claude-code/overview)

- [

  Quickstart

  ](/en/docs/claude-code/quickstart)

- [

  Common workflows

  ](/en/docs/claude-code/common-workflows)

##### Build with Claude Code

- [

  Claude Code SDK

  ](/en/docs/claude-code/sdk)

- [

  Subagents

  ](/en/docs/claude-code/sub-agents)

- [

  Output styles

  ](/en/docs/claude-code/output-styles)

- [

  Claude Code hooks

  ](/en/docs/claude-code/hooks-guide)

- [

  GitHub Actions

  ](/en/docs/claude-code/github-actions)

- [

  Model Context Protocol (MCP)

  ](/en/docs/claude-code/mcp)

- [

  Troubleshooting

  ](/en/docs/claude-code/troubleshooting)

##### Deployment

- [

  Overview

  ](/en/docs/claude-code/third-party-integrations)

- [

  Amazon Bedrock

  ](/en/docs/claude-code/amazon-bedrock)

- [

  Google Vertex AI

  ](/en/docs/claude-code/google-vertex-ai)

- [

  Corporate proxy

  ](/en/docs/claude-code/corporate-proxy)

- [

  LLM gateway

  ](/en/docs/claude-code/llm-gateway)

- [

  Development containers

  ](/en/docs/claude-code/devcontainer)

##### Administration

- [

  Advanced installation

  ](/en/docs/claude-code/setup)

- [

  Identity and Access Management

  ](/en/docs/claude-code/iam)

- [

  Security

  ](/en/docs/claude-code/security)

- [

  Data usage

  ](/en/docs/claude-code/data-usage)

- [

  Monitoring

  ](/en/docs/claude-code/monitoring-usage)

- [

  Costs

  ](/en/docs/claude-code/costs)

- [

  Analytics

  ](/en/docs/claude-code/analytics)

##### Configuration

- [

  Settings

  ](/en/docs/claude-code/settings)

- [

  Add Claude Code to your IDE

  ](/en/docs/claude-code/ide-integrations)

- [

  Terminal configuration

  ](/en/docs/claude-code/terminal-config)

- [

  Memory management

  ](/en/docs/claude-code/memory)

- [

  Status line configuration

  ](/en/docs/claude-code/statusline)

##### Reference

- [

  CLI reference

  ](/en/docs/claude-code/cli-reference)

- [

  Interactive mode

  ](/en/docs/claude-code/interactive-mode)

- [

  Slash commands

  ](/en/docs/claude-code/slash-commands)

- [

  Hooks reference

  ](/en/docs/claude-code/hooks)

##### Resources

- [

  Legal and compliance

  ](/en/docs/claude-code/legal-and-compliance)

Reference

# Hooks reference

Copy page

This page provides reference documentation for implementing hooks in Claude Code.

For a quickstart guide with examples, see [Get started with Claude Code hooks](/en/docs/claude-code/hooks-guide).

##

[​

](#configuration)

Configuration

Claude Code hooks are configured in your [settings files](/en/docs/claude-code/settings):

- `~/.claude/settings.json` - User settings
- `.claude/settings.json` - Project settings
- `.claude/settings.local.json` - Local project settings (not committed)
- Enterprise managed policy settings

###

[​

](#structure)

Structure

Hooks are organized by matchers, where each matcher can have multiple hooks:

Copy

```json
{
  "hooks": {
    "EventName": [
      {
        "matcher": "ToolPattern",
        "hooks": [
          {
            "type": "command",
            "command": "your-command-here"
          }
        ]
      }
    ]
  }
}
```

- **matcher**: Pattern to match tool names, case-sensitive (only applicable for `PreToolUse` and `PostToolUse`)
  - Simple strings match exactly: `Write` matches only the Write tool
  - Supports regex: `Edit|Write` or `Notebook.*`
  - Use `*` to match all tools. You can also use empty string (`""`) or leave `matcher` blank.
- **hooks**: Array of commands to execute when the pattern matches
  - `type`: Currently only `"command"` is supported
  - `command`: The bash command to execute (can use `$CLAUDE_PROJECT_DIR` environment variable)
  - `timeout`: (Optional) How long a command should run, in seconds, before canceling that specific command.

For events like `UserPromptSubmit`, `Notification`, `Stop`, and `SubagentStop` that don’t use matchers, you can omit the matcher field:

Copy

```json
{
  "hooks": {
    "UserPromptSubmit": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "/path/to/prompt-validator.py"
          }
        ]
      }
    ]
  }
}
```

###

[​

](#project-specific-hook-scripts)

Project-Specific Hook Scripts

You can use the environment variable `CLAUDE_PROJECT_DIR` (only available when Claude Code spawns the hook command) to reference scripts stored in your project, ensuring they work regardless of Claude’s current directory:

Copy

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Write|Edit",
        "hooks": [
          {
            "type": "command",
            "command": "$CLAUDE_PROJECT_DIR/.claude/hooks/check-style.sh"
          }
        ]
      }
    ]
  }
}
```

##

[​

](#hook-events)

Hook Events

###

[​

](#pretooluse)

PreToolUse

Runs after Claude creates tool parameters and before processing the tool call.

**Common matchers:**

- `Task` - Subagent tasks (see [subagents documentation](/en/docs/claude-code/sub-agents))
- `Bash` - Shell commands
- `Glob` - File pattern matching
- `Grep` - Content search
- `Read` - File reading
- `Edit`, `MultiEdit` - File editing
- `Write` - File writing
- `WebFetch`, `WebSearch` - Web operations

###

[​

](#posttooluse)

PostToolUse

Runs immediately after a tool completes successfully.

Recognizes the same matcher values as PreToolUse.

###

[​

](#notification)

Notification

Runs when Claude Code sends notifications. Notifications are sent when:

1.  Claude needs your permission to use a tool. Example: “Claude needs your permission to use Bash”
2.  The prompt input has been idle for at least 60 seconds. “Claude is waiting for your input”

###

[​

](#userpromptsubmit)

UserPromptSubmit

Runs when the user submits a prompt, before Claude processes it. This allows you to add additional context based on the prompt/conversation, validate prompts, or block certain types of prompts.

###

[​

](#stop)

Stop

Runs when the main Claude Code agent has finished responding. Does not run if the stoppage occurred due to a user interrupt.

###

[​

](#subagentstop)

SubagentStop

Runs when a Claude Code subagent (Task tool call) has finished responding.

###

[​

](#precompact)

PreCompact

Runs before Claude Code is about to run a compact operation.

**Matchers:**

- `manual` - Invoked from `/compact`
- `auto` - Invoked from auto-compact (due to full context window)

###

[​

](#sessionstart)

SessionStart

Runs when Claude Code starts a new session or resumes an existing session (which currently does start a new session under the hood). Useful for loading in development context like existing issues or recent changes to your codebase.

**Matchers:**

- `startup` - Invoked from startup
- `resume` - Invoked from `--resume`, `--continue`, or `/resume`
- `clear` - Invoked from `/clear`

##

[​

](#hook-input)

Hook Input

Hooks receive JSON data via stdin containing session information and event-specific data:

Copy

```typescript
{
  // Common fields
  session_id: string
  transcript_path: string  // Path to conversation JSON
  cwd: string              // The current working directory when the hook is invoked

  // Event-specific fields
  hook_event_name: string
  ...
}
```

###

[​

](#pretooluse-input)

PreToolUse Input

The exact schema for `tool_input` depends on the tool.

Copy

```json
{
  "session_id": "abc123",
  "transcript_path": "/Users/.../.claude/projects/.../00893aaf-19fa-41d2-8238-13269b9b3ca0.jsonl",
  "cwd": "/Users/...",
  "hook_event_name": "PreToolUse",
  "tool_name": "Write",
  "tool_input": {
    "file_path": "/path/to/file.txt",
    "content": "file content"
  }
}
```

###

[​

](#posttooluse-input)

PostToolUse Input

The exact schema for `tool_input` and `tool_response` depends on the tool.

Copy

```json
{
  "session_id": "abc123",
  "transcript_path": "/Users/.../.claude/projects/.../00893aaf-19fa-41d2-8238-13269b9b3ca0.jsonl",
  "cwd": "/Users/...",
  "hook_event_name": "PostToolUse",
  "tool_name": "Write",
  "tool_input": {
    "file_path": "/path/to/file.txt",
    "content": "file content"
  },
  "tool_response": {
    "filePath": "/path/to/file.txt",
    "success": true
  }
}
```

###

[​

](#notification-input)

Notification Input

Copy

```json
{
  "session_id": "abc123",
  "transcript_path": "/Users/.../.claude/projects/.../00893aaf-19fa-41d2-8238-13269b9b3ca0.jsonl",
  "cwd": "/Users/...",
  "hook_event_name": "Notification",
  "message": "Task completed successfully"
}
```

###

[​

](#userpromptsubmit-input)

UserPromptSubmit Input

Copy

```json
{
  "session_id": "abc123",
  "transcript_path": "/Users/.../.claude/projects/.../00893aaf-19fa-41d2-8238-13269b9b3ca0.jsonl",
  "cwd": "/Users/...",
  "hook_event_name": "UserPromptSubmit",
  "prompt": "Write a function to calculate the factorial of a number"
}
```

###

[​

](#stop-and-subagentstop-input)

Stop and SubagentStop Input

`stop_hook_active` is true when Claude Code is already continuing as a result of a stop hook. Check this value or process the transcript to prevent Claude Code from running indefinitely.

Copy

```json
{
  "session_id": "abc123",
  "transcript_path": "~/.claude/projects/.../00893aaf-19fa-41d2-8238-13269b9b3ca0.jsonl",
  "hook_event_name": "Stop",
  "stop_hook_active": true
}
```

###

[​

](#precompact-input)

PreCompact Input

For `manual`, `custom_instructions` comes from what the user passes into `/compact`. For `auto`, `custom_instructions` is empty.

Copy

```json
{
  "session_id": "abc123",
  "transcript_path": "~/.claude/projects/.../00893aaf-19fa-41d2-8238-13269b9b3ca0.jsonl",
  "hook_event_name": "PreCompact",
  "trigger": "manual",
  "custom_instructions": ""
}
```

###

[​

](#sessionstart-input)

SessionStart Input

Copy

```json
{
  "session_id": "abc123",
  "transcript_path": "~/.claude/projects/.../00893aaf-19fa-41d2-8238-13269b9b3ca0.jsonl",
  "hook_event_name": "SessionStart",
  "source": "startup"
}
```

##

[​

](#hook-output)

Hook Output

There are two ways for hooks to return output back to Claude Code. The output communicates whether to block and any feedback that should be shown to Claude and the user.

###

[​

](#simple%3A-exit-code)

Simple: Exit Code

Hooks communicate status through exit codes, stdout, and stderr:

- **Exit code 0**: Success. `stdout` is shown to the user in transcript mode (CTRL-R), except for `UserPromptSubmit` and `SessionStart`, where stdout is added to the context.
- **Exit code 2**: Blocking error. `stderr` is fed back to Claude to process automatically. See per-hook-event behavior below.
- **Other exit codes**: Non-blocking error. `stderr` is shown to the user and execution continues.

Reminder: Claude Code does not see stdout if the exit code is 0, except for the `UserPromptSubmit` hook where stdout is injected as context.

####

[​

](#exit-code-2-behavior)

Exit Code 2 Behavior

Hook Event

Behavior

`PreToolUse`

Blocks the tool call, shows stderr to Claude

`PostToolUse`

Shows stderr to Claude (tool already ran)

`Notification`

N/A, shows stderr to user only

`UserPromptSubmit`

Blocks prompt processing, erases prompt, shows stderr to user only

`Stop`

Blocks stoppage, shows stderr to Claude

`SubagentStop`

Blocks stoppage, shows stderr to Claude subagent

`PreCompact`

N/A, shows stderr to user only

`SessionStart`

N/A, shows stderr to user only

###

[​

](#advanced%3A-json-output)

Advanced: JSON Output

Hooks can return structured JSON in `stdout` for more sophisticated control:

####

[​

](#common-json-fields)

Common JSON Fields

All hook types can include these optional fields:

Copy

```json
{
  "continue": true, // Whether Claude should continue after hook execution (default: true)
  "stopReason": "string" // Message shown when continue is false
  "suppressOutput": true, // Hide stdout from transcript mode (default: false)
}
```

If `continue` is false, Claude stops processing after the hooks run.

- For `PreToolUse`, this is different from `"permissionDecision": "deny"`, which only blocks a specific tool call and provides automatic feedback to Claude.
- For `PostToolUse`, this is different from `"decision": "block"`, which provides automated feedback to Claude.
- For `UserPromptSubmit`, this prevents the prompt from being processed.
- For `Stop` and `SubagentStop`, this takes precedence over any `"decision": "block"` output.
- In all cases, `"continue" = false` takes precedence over any `"decision": "block"` output.

`stopReason` accompanies `continue` with a reason shown to the user, not shown to Claude.

####

[​

](#pretooluse-decision-control)

`PreToolUse` Decision Control

`PreToolUse` hooks can control whether a tool call proceeds.

- `"allow"` bypasses the permission system. `permissionDecisionReason` is shown to the user but not to Claude. (_Deprecated `"approve"` value + `reason` has the same behavior._)
- `"deny"` prevents the tool call from executing. `permissionDecisionReason` is shown to Claude. (_`"block"` value + `reason` has the same behavior._)
- `"ask"` asks the user to confirm the tool call in the UI. `permissionDecisionReason` is shown to the user but not to Claude.

Copy

```json
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "allow" | "deny" | "ask",
    "permissionDecisionReason": "My reason here (shown to user)"
  },
  "decision": "approve" | "block" | undefined, // Deprecated for PreToolUse but still supported
  "reason": "Explanation for decision" // Deprecated for PreToolUse but still supported
}
```

####

[​

](#posttooluse-decision-control)

`PostToolUse` Decision Control

`PostToolUse` hooks can control whether a tool call proceeds.

- `"block"` automatically prompts Claude with `reason`.
- `undefined` does nothing. `reason` is ignored.

Copy

```json
{
  "decision": "block" | undefined,
  "reason": "Explanation for decision"
}
```

####

[​

](#userpromptsubmit-decision-control)

`UserPromptSubmit` Decision Control

`UserPromptSubmit` hooks can control whether a user prompt is processed.

- `"block"` prevents the prompt from being processed. The submitted prompt is erased from context. `"reason"` is shown to the user but not added to context.
- `undefined` allows the prompt to proceed normally. `"reason"` is ignored.
- `"hookSpecificOutput.additionalContext"` adds the string to the context if not blocked.

Copy

```json
{
  "decision": "block" | undefined,
  "reason": "Explanation for decision",
  "hookSpecificOutput": {
    "hookEventName": "UserPromptSubmit",
    "additionalContext": "My additional context here"
  }
}
```

####

[​

](#stop%2Fsubagentstop-decision-control)

`Stop`/`SubagentStop` Decision Control

`Stop` and `SubagentStop` hooks can control whether Claude must continue.

- `"block"` prevents Claude from stopping. You must populate `reason` for Claude to know how to proceed.
- `undefined` allows Claude to stop. `reason` is ignored.

Copy

```json
{
  "decision": "block" | undefined,
  "reason": "Must be provided when Claude is blocked from stopping"
}
```

####

[​

](#sessionstart-decision-control)

`SessionStart` Decision Control

`SessionStart` hooks allow you to load in context at the start of a session.

- `"hookSpecificOutput.additionalContext"` adds the string to the context.

Copy

```json
{
  "hookSpecificOutput": {
    "hookEventName": "SessionStart",
    "additionalContext": "My additional context here"
  }
}
```

####

[​

](#exit-code-example%3A-bash-command-validation)

Exit Code Example: Bash Command Validation

Copy

```python
#!/usr/bin/env python3
import json
import re
import sys

# Define validation rules as a list of (regex pattern, message) tuples
VALIDATION_RULES = [
    (
        r"\bgrep\b(?!.*\|)",
        "Use 'rg' (ripgrep) instead of 'grep' for better performance and features",
    ),
    (
        r"\bfind\s+\S+\s+-name\b",
        "Use 'rg --files | rg pattern' or 'rg --files -g pattern' instead of 'find -name' for better performance",
    ),
]


def validate_command(command: str) -> list[str]:
    issues = []
    for pattern, message in VALIDATION_RULES:
        if re.search(pattern, command):
            issues.append(message)
    return issues


try:
    input_data = json.load(sys.stdin)
except json.JSONDecodeError as e:
    print(f"Error: Invalid JSON input: {e}", file=sys.stderr)
    sys.exit(1)

tool_name = input_data.get("tool_name", "")
tool_input = input_data.get("tool_input", {})
command = tool_input.get("command", "")

if tool_name != "Bash" or not command:
    sys.exit(1)

# Validate the command
issues = validate_command(command)

if issues:
    for message in issues:
        print(f"• {message}", file=sys.stderr)
    # Exit code 2 blocks tool call and shows stderr to Claude
    sys.exit(2)
```

####

[​

](#json-output-example%3A-userpromptsubmit-to-add-context-and-validation)

JSON Output Example: UserPromptSubmit to Add Context and Validation

For `UserPromptSubmit` hooks, you can inject context using either method:

- Exit code 0 with stdout: Claude sees the context (special case for `UserPromptSubmit`)
- JSON output: Provides more control over the behavior

Copy

```python
#!/usr/bin/env python3
import json
import sys
import re
import datetime

# Load input from stdin
try:
    input_data = json.load(sys.stdin)
except json.JSONDecodeError as e:
    print(f"Error: Invalid JSON input: {e}", file=sys.stderr)
    sys.exit(1)

prompt = input_data.get("prompt", "")

# Check for sensitive patterns
sensitive_patterns = [
    (r"(?i)\b(password|secret|key|token)\s*[:=]", "Prompt contains potential secrets"),
]

for pattern, message in sensitive_patterns:
    if re.search(pattern, prompt):
        # Use JSON output to block with a specific reason
        output = {
            "decision": "block",
            "reason": f"Security policy violation: {message}. Please rephrase your request without sensitive information."
        }
        print(json.dumps(output))
        sys.exit(0)

# Add current time to context
context = f"Current time: {datetime.datetime.now()}"
print(context)

"""
The following is also equivalent:
print(json.dumps({
  "hookSpecificOutput": {
    "hookEventName": "UserPromptSubmit",
    "additionalContext": context,
  },
}))
"""

# Allow the prompt to proceed with the additional context
sys.exit(0)
```

####

[​

](#json-output-example%3A-pretooluse-with-approval)

JSON Output Example: PreToolUse with Approval

Copy

```python
#!/usr/bin/env python3
import json
import sys

# Load input from stdin
try:
    input_data = json.load(sys.stdin)
except json.JSONDecodeError as e:
    print(f"Error: Invalid JSON input: {e}", file=sys.stderr)
    sys.exit(1)

tool_name = input_data.get("tool_name", "")
tool_input = input_data.get("tool_input", {})

# Example: Auto-approve file reads for documentation files
if tool_name == "Read":
    file_path = tool_input.get("file_path", "")
    if file_path.endswith((".md", ".mdx", ".txt", ".json")):
        # Use JSON output to auto-approve the tool call
        output = {
            "decision": "approve",
            "reason": "Documentation file auto-approved",
            "suppressOutput": True  # Don't show in transcript mode
        }
        print(json.dumps(output))
        sys.exit(0)

# For other cases, let the normal permission flow proceed
sys.exit(0)
```

##

[​

](#working-with-mcp-tools)

Working with MCP Tools

Claude Code hooks work seamlessly with [Model Context Protocol (MCP) tools](/en/docs/claude-code/mcp). When MCP servers provide tools, they appear with a special naming pattern that you can match in your hooks.

###

[​

](#mcp-tool-naming)

MCP Tool Naming

MCP tools follow the pattern `mcp__<server>__<tool>`, for example:

- `mcp__memory__create_entities` - Memory server’s create entities tool
- `mcp__filesystem__read_file` - Filesystem server’s read file tool
- `mcp__github__search_repositories` - GitHub server’s search tool

###

[​

](#configuring-hooks-for-mcp-tools)

Configuring Hooks for MCP Tools

You can target specific MCP tools or entire MCP servers:

Copy

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "mcp__memory__.*",
        "hooks": [
          {
            "type": "command",
            "command": "echo 'Memory operation initiated' >> ~/mcp-operations.log"
          }
        ]
      },
      {
        "matcher": "mcp__.*__write.*",
        "hooks": [
          {
            "type": "command",
            "command": "/home/user/scripts/validate-mcp-write.py"
          }
        ]
      }
    ]
  }
}
```

##

[​

](#examples)

Examples

For practical examples including code formatting, notifications, and file protection, see [More Examples](/en/docs/claude-code/hooks-guide#more-examples) in the get started guide.

##

[​

](#security-considerations)

Security Considerations

###

[​

](#disclaimer)

Disclaimer

**USE AT YOUR OWN RISK**: Claude Code hooks execute arbitrary shell commands on your system automatically. By using hooks, you acknowledge that:

- You are solely responsible for the commands you configure
- Hooks can modify, delete, or access any files your user account can access
- Malicious or poorly written hooks can cause data loss or system damage
- Anthropic provides no warranty and assumes no liability for any damages resulting from hook usage
- You should thoroughly test hooks in a safe environment before production use

Always review and understand any hook commands before adding them to your configuration.

###

[​

](#security-best-practices)

Security Best Practices

Here are some key practices for writing more secure hooks:

1.  **Validate and sanitize inputs** - Never trust input data blindly
2.  **Always quote shell variables** - Use `"$VAR"` not `$VAR`
3.  **Block path traversal** - Check for `..` in file paths
4.  **Use absolute paths** - Specify full paths for scripts (use `$CLAUDE_PROJECT_DIR` for the project path)
5.  **Skip sensitive files** - Avoid `.env`, `.git/`, keys, etc.

###

[​

](#configuration-safety)

Configuration Safety

Direct edits to hooks in settings files don’t take effect immediately. Claude Code:

1.  Captures a snapshot of hooks at startup
2.  Uses this snapshot throughout the session
3.  Warns if hooks are modified externally
4.  Requires review in `/hooks` menu for changes to apply

This prevents malicious hook modifications from affecting your current session.

##

[​

](#hook-execution-details)

Hook Execution Details

- **Timeout**: 60-second execution limit by default, configurable per command.
  - A timeout for an individual command does not affect the other commands.
- **Parallelization**: All matching hooks run in parallel
- **Environment**: Runs in current directory with Claude Code’s environment
  - The `CLAUDE_PROJECT_DIR` environment variable is available and contains the absolute path to the project root directory
- **Input**: JSON via stdin
- **Output**:
  - PreToolUse/PostToolUse/Stop: Progress shown in transcript (Ctrl-R)
  - Notification: Logged to debug only (`--debug`)

##

[​

](#debugging)

Debugging

###

[​

](#basic-troubleshooting)

Basic Troubleshooting

If your hooks aren’t working:

1.  **Check configuration** - Run `/hooks` to see if your hook is registered
2.  **Verify syntax** - Ensure your JSON settings are valid
3.  **Test commands** - Run hook commands manually first
4.  **Check permissions** - Make sure scripts are executable
5.  **Review logs** - Use `claude --debug` to see hook execution details

Common issues:

- **Quotes not escaped** - Use `\"` inside JSON strings
- **Wrong matcher** - Check tool names match exactly (case-sensitive)
- **Command not found** - Use full paths for scripts

###

[​

](#advanced-debugging)

Advanced Debugging

For complex hook issues:

1.  **Inspect hook execution** - Use `claude --debug` to see detailed hook execution
2.  **Validate JSON schemas** - Test hook input/output with external tools
3.  **Check environment variables** - Verify Claude Code’s environment is correct
4.  **Test edge cases** - Try hooks with unusual file paths or inputs
5.  **Monitor system resources** - Check for resource exhaustion during hook execution
6.  **Use structured logging** - Implement logging in your hook scripts

###

[​

](#debug-output-example)

Debug Output Example

Use `claude --debug` to see hook execution details:

Copy

```
[DEBUG] Executing hooks for PostToolUse:Write
[DEBUG] Getting matching hook commands for PostToolUse with query: Write
[DEBUG] Found 1 hook matchers in settings
[DEBUG] Matched 1 hooks for query "Write"
[DEBUG] Found 1 hook commands to execute
[DEBUG] Executing hook command: <Your command> with timeout 60000ms
[DEBUG] Hook command completed with status 0: <Your stdout>
```

Progress messages appear in transcript mode (Ctrl-R) showing:

- Which hook is running
- Command being executed
- Success/failure status
- Output or error messages

Was this page helpful?

YesNo

[Slash commands](/en/docs/claude-code/slash-commands)[Legal and compliance](/en/docs/claude-code/legal-and-compliance)

On this page

- [Configuration](#configuration)
- [Structure](#structure)
- [Project-Specific Hook Scripts](#project-specific-hook-scripts)
- [Hook Events](#hook-events)
- [PreToolUse](#pretooluse)
- [PostToolUse](#posttooluse)
- [Notification](#notification)
- [UserPromptSubmit](#userpromptsubmit)
- [Stop](#stop)
- [SubagentStop](#subagentstop)
- [PreCompact](#precompact)
- [SessionStart](#sessionstart)
- [Hook Input](#hook-input)
- [PreToolUse Input](#pretooluse-input)
- [PostToolUse Input](#posttooluse-input)
- [Notification Input](#notification-input)
- [UserPromptSubmit Input](#userpromptsubmit-input)
- [Stop and SubagentStop Input](#stop-and-subagentstop-input)
- [PreCompact Input](#precompact-input)
- [SessionStart Input](#sessionstart-input)
- [Hook Output](#hook-output)
- [Simple: Exit Code](#simple%3A-exit-code)
- [Exit Code 2 Behavior](#exit-code-2-behavior)
- [Advanced: JSON Output](#advanced%3A-json-output)
- [Common JSON Fields](#common-json-fields)
- [PreToolUse Decision Control](#pretooluse-decision-control)
- [PostToolUse Decision Control](#posttooluse-decision-control)
- [UserPromptSubmit Decision Control](#userpromptsubmit-decision-control)
- [Stop/SubagentStop Decision Control](#stop%2Fsubagentstop-decision-control)
- [SessionStart Decision Control](#sessionstart-decision-control)
- [Exit Code Example: Bash Command Validation](#exit-code-example%3A-bash-command-validation)
- [JSON Output Example: UserPromptSubmit to Add Context and Validation](#json-output-example%3A-userpromptsubmit-to-add-context-and-validation)
- [JSON Output Example: PreToolUse with Approval](#json-output-example%3A-pretooluse-with-approval)
- [Working with MCP Tools](#working-with-mcp-tools)
- [MCP Tool Naming](#mcp-tool-naming)
- [Configuring Hooks for MCP Tools](#configuring-hooks-for-mcp-tools)
- [Examples](#examples)
- [Security Considerations](#security-considerations)
- [Disclaimer](#disclaimer)
- [Security Best Practices](#security-best-practices)
- [Configuration Safety](#configuration-safety)
- [Hook Execution Details](#hook-execution-details)
- [Debugging](#debugging)
- [Basic Troubleshooting](#basic-troubleshooting)
- [Advanced Debugging](#advanced-debugging)
- [Debug Output Example](#debug-output-example)
