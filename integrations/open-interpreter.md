# Open Interpreter Integration for AgentTrack

Open Interpreter lets language models run code on your computer. You can configure it to automatically log activities using a custom profile.

## Setup

1. Create a new Open Interpreter profile by creating a YAML file (e.g., `atrack.yaml`) in your profiles directory (usually `~/.config/open-interpreter/profiles/` or `%APPDATA%\Open Interpreter\profiles\` on Windows).

2. Add the following system message instruction to the `atrack.yaml` profile:

```yaml
system_message: |
  You are an AI assistant using Open Interpreter.
  
  ## AgentTrack Auto-Logging
  At the end of EVERY single interaction with the user, you MUST run the following shell command to log the activity:
  `atrack auto "<user_question>" "<ai_summary>" "open-interpreter" 0 0`
  
  - Replace `<user_question>` with the user's core request.
  - Replace `<ai_summary>` with a summary of the code you executed and the results.
```

3. Run Open Interpreter with this profile:

```bash
interpreter --profile atrack.yaml
```
