# Claude Code Integration for AgentTrack

[Claude Code](https://docs.anthropic.com/en/docs/agents-and-tools/claude-code) is an official CLI coding assistant developed by Anthropic.

## Setup

To enforce auto-logging for Claude Code within your project, you can use its native convention of reading project-specific markdown instructions.

1. Create a `CLAUDE.md` file in the root of your project:

```markdown
# AgentTrack Auto-Logging
At the end of EVERY single interaction with the user, you MUST run the following command in the terminal to log your activity:
`atrack auto "<user_question>" "<ai_summary>" "claude-code" 0 0`

- Replace `<user_question>` with the core instruction the user provided.
- Replace `<ai_summary>` with a brief summary of the modifications you made.
```

2. When you run the `claude` command inside this directory, Claude Code will automatically ingest the `CLAUDE.md` file as part of its system prompt, ensuring it complies with your tracking requirements.
