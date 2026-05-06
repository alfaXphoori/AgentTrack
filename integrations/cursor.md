# Cursor Integration for TrackCLI

To enable auto-logging in Cursor, add the following to your `.cursorrules` file in the project root:

```markdown
## TrackCLI Auto-Logging
At the end of every interaction, you MUST run the following command to log the activity:
`trackcli auto "<user_question>" "<ai_summary>" "<model_name>" 0 0`

- Replace placeholders with the actual conversation details.
- Use `trackcli` if installed globally, or `go run .` if in the project root.
```
