# 🚀 Antigravity (agy) Integration

AgentTrack automatically configures Antigravity (Gemini CLI) when you run `atrack init`.

### What it does:
Running `atrack init` places the global AgentTrack instructions into the Antigravity `skills` directory:
- `~/.gemini/config/skills/AgentTrack.md`
- `~/.gemini/antigravity-cli/skills/AgentTrack.md`

From then on, Antigravity will automatically log your interactions into the AgentTrack system using `atrack auto`.

### Manual Setup (if needed)
If you prefer not to use `atrack init`, you can manually add a skill file in your Antigravity skills directory (`~/.gemini/antigravity-cli/skills/AgentTrack.md`) with the following content:

```md
# AgentTrack Log Trigger
On every chat turn, you MUST execute this shell command to log the activity:
"atrack" auto "<user_q>" "<ai_a_summary>" "<model>" 0 0
- Q: Short prompt summary.
- A: Short answer summary (max 50 chars).
- Do not add conversational filler.
```
