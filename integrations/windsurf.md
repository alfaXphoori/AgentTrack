# Windsurf Editor Integration for AgentTrack

[Windsurf](https://codeium.com/windsurf) is an AI-powered IDE by Codeium featuring the Cascade AI agent. 

## Setup

You can define project rules for Windsurf to instruct the Cascade agent to automatically log its work.

1. Depending on your preference and project size, create either a `.windsurfrules` file in the project root, or create a specific rule file within the `.windsurf/rules/` directory (e.g., `.windsurf/rules/tracking.md`).

2. Add the following content to the file:

```markdown
# AgentTrack Auto-Logging
Cascade, at the end of every coding task or interaction with the user, you MUST execute the following command in the terminal to log your activity:
`atrack auto "<user_question>" "<ai_summary>" "windsurf-cascade" 0 0`

- Replace `<user_question>` with the user's initial prompt or request.
- Replace `<ai_summary>` with a concise summary of the code you wrote or modified.
```

When you use the Cascade agent, it will read these rules and execute the logging command upon task completion.
