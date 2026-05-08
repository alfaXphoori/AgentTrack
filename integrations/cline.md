# Cline Integration for AgentTrack

To enable auto-logging in Cline (formerly Claude Dev), add the following to your "Custom Instructions" or `.clinerules` file:

```markdown
## AgentTrack Auto-Logging
Mandatory: After completing a task or responding to the user, execute:
`atrack auto "<user_question>" "<ai_summary>" "<model_name>" 0 0`
```
