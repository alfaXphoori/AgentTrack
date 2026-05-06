# Cline Integration for TrackCLI

To enable auto-logging in Cline (formerly Claude Dev), add the following to your "Custom Instructions" or `.clinerules` file:

```markdown
## TrackCLI Auto-Logging
Mandatory: After completing a task or responding to the user, execute:
`trackcli auto "<user_question>" "<ai_summary>" "<model_name>" 0 0`
```
