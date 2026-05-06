# Google Gemini CLI Integration

Track your Gemini CLI sessions automatically with TrackCLI.

## Auto-Logging Wrapper (Recommended)

Use the included `gemitrack.sh` wrapper to automatically log every Gemini question:

```bash
# Start interactive session — model is auto-detected from your last Gemini CLI session
./gemitrack.sh

# Override with a specific model
GEMINI_MODEL=gemini-2.5-pro ./gemitrack.sh
```

Inside the session:
- Type your question → Gemini answers → auto-logged to TrackCLI
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
trackcli auto "Your question here" "Summary of Gemini's answer" "gemini-2.5-flash" <tokens_in> <tokens_out>
```

## Check Current Model

Inside a Gemini CLI session:
```
/model
```

Start with a specific model:
```bash
gemini -m gemini-3.1-pro-preview
gemini -m gemini-3-flash-preview
gemini -m gemini-2.5-pro
```

## Shell Alias

Add to your `.zshrc` or `.bashrc` for quick logging:
```bash
alias glog='trackcli auto'
```

Then use:
```bash
glog "What is Docker?" "Docker is a containerization platform" "gemini-2.0-flash" 0 0
```

## View Gemini Usage Stats

```bash
# See all models used
trackcli list model all

# Filter logs by Gemini model
trackcli list model "gemini"

# Search for Gemini-related logs
trackcli search model "gemini-2.0-flash"

# View cost stats
trackcli stats cost
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
