# Aider Integration for AgentTrack

Aider is a popular AI pair programming tool in the terminal. To ensure Aider automatically logs its interactions using AgentTrack, you can define project-specific conventions.

## Setup

1. Create a file named `.aider.rules.md` (or similar) in your project root with the following content:

```markdown
# AgentTrack Auto-Logging
At the end of EVERY interaction, you MUST run the following command in the terminal to log your activity:
`atrack auto "<user_question>" "<ai_summary>" "aider" 0 0`

- Replace `<user_question>` with the user's prompt.
- Replace `<ai_summary>` with a brief summary of what you did.
```

2. When launching Aider, instruct it to read these rules using the `--read` flag:

```bash
aider --read .aider.rules.md
```

Alternatively, you can add this to your `.aider.conf.yml` in the project root:

```yaml
read:
  - .aider.rules.md
```
