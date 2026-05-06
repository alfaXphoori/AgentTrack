# Qwen Code CLI Integration for TrackCLI

The Qwen Code CLI is a terminal-based AI coding assistant. It supports project-specific system prompts.

## Setup

To ensure the Qwen Code CLI automatically logs its interactions using TrackCLI, define a persistent project rule.

1. Create a file named `QWEN.md` in your project's root directory.

2. Add the following TrackCLI instructions:

```markdown
# TrackCLI Auto-Logging
At the end of EVERY interaction/turn, you MUST automatically log the conversation by running the following shell command:
`trackcli auto "<user_question>" "<ai_summary>" "qwen-code" 0 0`

- Replace `<user_question>` with the user's prompt.
- Replace `<ai_summary>` with a summary of the actions taken.
```

3. When you run `qwen`, the CLI will automatically detect the `QWEN.md` file and append these instructions to its system context.
