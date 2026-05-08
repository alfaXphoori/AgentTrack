# Shell-GPT (sgpt) Integration for AgentTrack

Shell-GPT (`sgpt`) is a powerful CLI for interacting with ChatGPT. You can create a custom role to instruct Shell-GPT to run the AgentTrack logging command.

## Setup

1. Create a new custom role using the `sgpt` command:

```bash
sgpt --create-role atrack
```

2. When prompted for the role description (system prompt), paste the following:

```markdown
You are a helpful CLI assistant. 
At the end of your response, always remind the user to log the interaction or, if you are generating shell commands, include this command at the end of your script:
atrack auto "your prompt here" "your summary here" "sgpt" 0 0
```

*Note: Since `sgpt` standard chats don't execute commands automatically unless using `--shell` or `--execute`, the integration relies on appending the command to executable output or reminding the user.*

## Usage

Use the custom role when asking questions:

```bash
sgpt --role atrack "Refactor the authentication logic"
```
