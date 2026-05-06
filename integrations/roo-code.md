# Roo Code Integration for TrackCLI

[Roo Code](https://roocode.com/) (formerly Roo Cline) is an advanced AI coding assistant that uses "Rules as Code."

## Setup

Roo Code allows for detailed rule structures. To integrate TrackCLI, you can define a workspace rule.

1. Create a `.roorules` file in the root of your project, or if you are using the directory structure, create a file like `.roo/rules/00-tracking.md`.

2. Add the following instruction:

```markdown
# TrackCLI Auto-Logging
At the end of EVERY interaction or task completion, you MUST run the following command in the terminal to log your activity:
`trackcli auto "<user_question>" "<ai_summary>" "roo-code" 0 0`

- Replace `<user_question>` with the core instruction the user provided.
- Replace `<ai_summary>` with a brief summary of the modifications you made.
```

**Note:** Because Roo Code shares lineage with Cline, it will also automatically read `.clinerules`, `.cursorrules`, or `.windsurfrules` if they exist in your project, making it highly compatible with existing setups.
