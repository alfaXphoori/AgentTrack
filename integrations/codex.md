# Codex CLI Integration for TrackCLI

[Codex CLI](https://github.com/openai/codex-cli) (OpenAI's open-source agentic terminal tool) allows for project-specific instructions that the AI agent follows implicitly during sessions.

## Setup

To enforce auto-logging for the Codex CLI within your project, you can use the `AGENTS.md` file convention. Codex automatically discovers this file in the project root and appends its contents to the system prompt.

1. Create an `AGENTS.md` file in the root of your project:

```markdown
# TrackCLI Auto-Logging
At the end of EVERY single interaction or completed task, you MUST run the following command in the terminal to log your activity:
`trackcli auto "<user_question>" "<ai_summary>" "codex-cli" 0 0`

- Replace `<user_question>` with the core instruction the user provided.
- Replace `<ai_summary>` with a brief summary of the modifications you made or the commands you executed.
```

2. When you run `codex` inside this directory, it will read `AGENTS.md` and adhere to the TrackCLI auto-logging requirements.

### Alternative: Custom Prompts
If you prefer not to use `AGENTS.md`, you can create a custom slash command prompt in your global configuration:
1. Create `~/.codex/prompts/log.md`:
   ```markdown
   ---
   description: Generate a TrackCLI log command
   ---
   Generate the exact shell command to log the recent session using TrackCLI:
   `trackcli auto "<user_question>" "<ai_summary>" "codex-cli" 0 0`. 
   Replace placeholders with context from our session. Do not execute it, just provide the command.
   ```
2. In Codex, type `/prompts:log` to generate the command.
