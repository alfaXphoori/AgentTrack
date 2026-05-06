# Continue.dev Integration for TrackCLI

[Continue.dev](https://continue.dev/) is an open-source AI code assistant that works as an extension in VS Code and JetBrains IDEs.

## Setup

Continue uses a `config.json` file (typically located at `~/.continue/config.json`) and also supports workspace prompts via `.prompt` files or system messages.

### Method 1: System Message Configuration
You can update your Continue configuration to include a global or workspace-specific system message.

Open `~/.continue/config.json` and add or modify the `systemMessage` property:

```json
{
  "systemMessage": "You are an expert AI programmer. At the end of every interaction where you suggest or modify code, you MUST instruct the user to run, or attempt to run the following command if terminal access is available: `trackcli auto \"<user_question>\" \"<ai_summary>\" \"continue-dev\" 0 0`"
}
```

### Method 2: Custom Slash Command
You can create a custom slash command in `config.json` specifically for logging:

```json
{
  "customCommands": [
    {
      "name": "log",
      "prompt": "Summarize our recent conversation and generate the exact shell command to log it using TrackCLI: `trackcli auto \"<user_question>\" \"<ai_summary>\" \"continue-dev\" 0 0`. Do not execute it, just provide the command.",
      "description": "Generate a TrackCLI log command for the session."
    }
  ]
}
```
*Note: Because Continue primarily acts as a chat assistant and autocomplete tool without direct autonomous terminal execution (unlike Cline or Aider), you often need to manually copy-paste the generated command or use a custom slash command to finalize the log.*
