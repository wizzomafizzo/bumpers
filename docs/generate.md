# AI Generation

Bumpers can enhance static template messages with AI-generated content using Claude API. This provides dynamic, contextual responses while maintaining the speed and reliability of template-based rules.

## Generation Modes

All configuration sections (rules, commands, session) support AI generation:

### Simple Configuration
```yaml
rules:
  - match: "dangerous_command"
    send: "Base message about safety"
    generate: "session"  # Mode only
```

### Advanced Configuration  
```yaml
rules:
  - match: "complex_error"
    send: "Something went wrong with {{.Command}}"
    generate:
      mode: "session"
      prompt: "Be specific about debugging steps and mention project context"
```

## Generation Modes Explained

### `off` (Default)
No AI generation - uses template message only.
```yaml
generate: "off"  # or omit generate field entirely
```
**Use case**: Fast, deterministic responses

### `once` 
Generate AI response once, cache permanently across all sessions.
```yaml
generate: "once"
```
**Use case**: Stable project advice that doesn't change (coding standards, architecture decisions)
**Performance**: Fastest after first generation
**Storage**: Permanent cache in `~/.local/share/bumpers/cache/`

### `session`
Generate once per Claude session, cache within session.
```yaml
generate: "session"
```  
**Use case**: Context that may evolve (current errors, recent changes, daily advice)
**Performance**: Fast within session, regenerates on new sessions
**Storage**: Session cache in `~/.local/share/bumpers/sessions/`

### `always`
Generate every time - no caching.
```yaml
generate: "always"
```
**Use case**: Time-sensitive or highly dynamic content
**Performance**: Slowest - requires API call every time
**Storage**: No caching

## Custom Prompts

Enhance AI responses with specific instructions:

```yaml
rules:
  - match: "test.*fail"
    send: "Tests are failing for {{.Command}}"
    generate:
      mode: "session"
      prompt: |
        The user is having test failures. Be encouraging and provide specific 
        debugging steps. Consider common causes like dependency issues, 
        environment problems, or test data setup. Reference project-specific 
        testing tools if mentioned in the context.
```

**Prompt behavior:**
- Combined with built-in Bumpers prompts
- Provides additional context to AI
- Can reference template variables
- Should be specific and actionable

## AI Integration Details

### API Requirements
- **Claude API key**: Set `ANTHROPIC_API_KEY` environment variable
- **Model**: Uses Claude Sonnet model
- **Fallback**: On AI failure, returns template message

### Context Provided to AI
The AI receives:
- **Template message**: Your base message after template processing
- **Command context**: Matched command and tool information  
- **Custom prompt**: Your specific instructions
- **Project context**: Basic project information when available
- **Error context**: For post-tool-use hooks, error details

### Response Enhancement
AI responses:
- **Extend** template messages (don't replace)
- **Maintain** encouraging, helpful tone
- **Include** specific, actionable advice
- **Preserve** template variables and formatting

## Caching Behavior

### Cache Keys
Cache entries are keyed by:
- **Rule pattern**: Different patterns get separate cache entries
- **Command name**: Different commands cache separately
- **Template content**: Changes to base message invalidate cache
- **Custom prompt**: Changes to prompt invalidate cache

### Cache Storage
Cache entries are stored in a BBolt database with project-specific buckets. The exact storage location follows XDG Base Directory specifications.

### Cache Management
- **Session expiry**: Session caches expire after 24 hours
- **Permanent caching**: "once" mode entries never expire

## Performance Considerations

### Generation Timing
- **Template processing**: ~1ms (always fast)
- **AI generation**: 1-3 seconds (depends on API latency)
- **Cache lookup**: ~1ms (nearly instant)

### Mode Selection Strategy
```yaml
# Fast, static advice
generate: "once"

# Session-aware but cached
generate: "session"  

# Dynamic but slower  
generate: "always"

# No AI overhead
generate: "off"
```

### Performance Notes
- Template processing is always fast (~1ms)
- AI generation depends on API latency (1-3 seconds)
- Cache lookups are nearly instant

## AI Generation Examples

### Error Analysis
```yaml
rules:
  - match:
      pattern: "error|exception|failed"
      event: "post"
      sources: ["tool_output"]
    send: "Command failed: {{.Command}}"
    generate:
      mode: "session"
      prompt: |
        Analyze the error and provide specific debugging steps.
        Consider common solutions and project-specific tools.
```

### Context-Aware Commands
```yaml
commands:
  - name: "help"
    send: "Available project commands"
    generate:
      mode: "once"  
      prompt: |
        List the most useful commands for this project based on the
        available build files and configurations. Be concise but helpful.
```

### Dynamic Session Context
```yaml
session:
  - add: "Current project status and reminders"
    generate:
      mode: "session"
      prompt: |
        Provide a brief status update and any important reminders
        for this development session. Check for uncommitted changes,
        failing tests, or pending tasks.
```

## Troubleshooting AI Generation

### Common Issues

#### API Key Not Set
```
ERROR AI generation failed error="missing ANTHROPIC_API_KEY"
```
**Solution**: Set environment variable: `export ANTHROPIC_API_KEY=your_key_here`

#### API Rate Limits  
```
WARN AI generation rate limited, using template message
```
**Solution**: Reduce usage of `generate: "always"`, use session caching

#### Network Issues
```
ERROR AI generation timeout error="context deadline exceeded"
```
**Solution**: Check network connection, fallback to template message automatically

### Debugging AI Responses
Enable debug logging to see AI generation details:
```bash
BUMPERS_LOG_LEVEL=debug bumpers hook < input.json
```

Logs show:
- Cache hits/misses
- AI prompt content  
- Generation timing
- Error details

### Testing AI Generation
Test AI responses without triggering hooks:
```bash
# Test rule matching
echo '{"tool_name": "Bash", "tool_input": {"command": "go test"}}' | bumpers hook

# Check cache status
ls -la ~/.local/share/bumpers/cache/
ls -la ~/.local/share/bumpers/sessions/
```

## Best Practices

### Prompt Writing
- **Be specific**: "Provide debugging steps for test failures" vs "help with tests"
- **Include context**: Reference project type, tools, common patterns
- **Stay focused**: Don't ask for general advice, target the specific situation
- **Be concise**: Long prompts increase API costs and latency

### Mode Selection
- **Use `once`** for stable project advice (coding standards, architecture)  
- **Use `session`** for evolving context (current errors, daily guidance)
- **Use `always`** sparingly for truly dynamic content
- **Use `off`** for fast, simple responses

### Performance Optimization
- **Cache strategically**: Most responses can be cached at session level
- **Template first**: Write good template messages, let AI enhance them
- **Test locally**: Use debug mode to verify prompts and responses
- **Monitor usage**: Check logs for API call frequency and errors