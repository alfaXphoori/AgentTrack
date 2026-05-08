# Google Gemini CLI Integration

Track your Gemini CLI sessions automatically with AgentTrack.

## Recommended: Passive Background Tracking

The most efficient way to track Gemini CLI is by using the included background watcher script. This method is passive: it monitors Gemini's local log files directly, saving the AI's context window and time.

To start tracking, run the watcher script in your terminal (you can run it in the background by appending `&`):

```bash
./gemini-cli-track.sh &
```

The script will automatically detect the current Gemini project and quietly parse the `~/.gemini/tmp/<project>/chats/session-*.jsonl` files. Every time you ask a question and Gemini replies, it will silently log the interaction into AgentTrack.

### Stop the Watcher
If you started it in the background, you can kill the process using standard job control (e.g., `kill %1` or finding its PID via `ps aux | grep gemini-cli-track`).

## Alternative: Native Integration (`GEMINI.md`)

If you cannot run background scripts, you can instruct the Gemini CLI to manually run the log command itself at the end of every interaction. 

1. Create or edit `GEMINI.md` in the root of your project.
2. Add the following rule:

```markdown
# AgentTrack Auto-Logging
At the end of every interaction, you MUST run the following shell command to log the activity:
`atrack auto "<user_question>" "<ai_summary>" "<model_name>" 0 0`
```

*(Note: This approach consumes more tokens and takes slightly longer as the AI has to actively execute the command).*


If you prefer a lightweight wrapper script for quick, single-shot questions without invoking the full Gemini CLI agent loop, you can use the included `gemitrack.sh` wrapper:

```bash
# Start interactive session — model is auto-detected from your last Gemini CLI session
./gemitrack.sh

# Override with a specific model
GEMINI_MODEL=gemini-2.5-pro ./gemitrack.sh
```

Inside the session:
- Type your question → Gemini answers → auto-logged to AgentTrack
- `/model` → switch model interactively
- `exit` → quit

### Live Model Auto-Detection

`gemitrack.sh` automatically detects which model you last used in Gemini CLI by reading the session files at:

```
~/.gemini/tmp/<project>/chats/session-*.jsonl
```

**Model priority order:**
1. `GEMINI_MODEL` env var (manual override)
2. Live session file detection ← auto-detects your current Gemini CLI model
3. Fallback: `gemini-2.5-flash`

To refresh the detected model, simply restart `gemitrack.sh` after switching models in Gemini CLI.

## Manual Logging

After a Gemini CLI session, log the interaction manually:

```bash
atrack auto "Your question here" "Summary of Gemini's answer" "gemini-2.5-flash" <tokens_in> <tokens_out>
```

## Shell Alias

Add to your `.zshrc` or `.bashrc` for quick logging:
```bash
alias glog='atrack auto'
```

Then use:
```bash
glog "What is Docker?" "Docker is a containerization platform" "gemini-2.0-flash" 0 0
```

## View Gemini Usage Stats

```bash
# See all models used
atrack list model all

# Filter logs by Gemini model
atrack list model "gemini"

# Search for Gemini-related logs
atrack search model "gemini-2.0-flash"

# View cost stats
atrack stats cost
```

## Available Gemini Models

| # | Model ID | Notes |
|---|---|---|
| 1 | `gemini-3.1-pro-preview` | Latest Pro preview |
| 2 | `gemini-3-flash-preview` | Latest Flash preview |
| 3 | `gemini-3.1-flash-lite-preview` | Lightweight Flash preview |
| 4 | `gemini-2.5-pro` | Stable Pro |
| 5 | `gemini-2.5-flash` | Stable Flash |
| 6 | `gemini-2.5-flash-lite` | Lightweight Flash |

> Use `/model` inside a Gemini CLI session to switch models interactively.
