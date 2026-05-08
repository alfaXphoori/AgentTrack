# AgentTrack — AI Activity Tracker `v0.13`

A cross-platform terminal tool for tracking tasks and AI interactions. Automatically logs questions, answers, models, and token usage from your AI CLI sessions — with real-time Gemini CLI watching, activity summaries, and tag management.

## What's New in v0.13
- **`atrack summary [today|week|month]`** — Activity summary with model breakdown, token totals, and tag usage
- **`atrack tag list`** — View all tags and their counts across all logs
- **`atrack stats today`** — Quick today-only token and log stats
- **`atrack pricing sync [all|model ...]`** — Pull the latest OpenRouter pricing into local config
- **`gemini-watch.sh`** — Background watcher that auto-logs native Gemini CLI sessions in real-time (no wrapper needed)
- **Bug fix:** Timezone-aware date filtering in summary commands

## Features
- **Auto-Tracking:** Logs AI questions, answers, models, and token counts automatically
- **Gemini CLI Native Watch:** `gemini-watch.sh` monitors live Gemini sessions without any wrapper
- **Activity Summaries:** Daily, weekly, and monthly breakdowns with model and tag stats
- **Tag System:** Tag logs and search/list by tag
- **Manual Logging:** Add entries with categories, tags, and timestamps
- **Real-time Watch:** `atrack watch` monitors new logs live
- **Stats & Cost:** Per-model token usage and estimated cost via OpenRouter pricing
- **OpenRouter Price Sync:** Pull the latest model prices from OpenRouter and save updates into config
- **Export:** Export logs to Markdown, CSV, or JSON
- **Date Filtering:** Filter any command with `--from` / `--to`
- **Config:** Timezone, token estimation, display preferences, model pricing
- **Monthly Log Rotation:** Logs stored per-month in `~/.atrack/`

## Setup
Requires **Go (Golang)** installed. See [INSTALL.md](INSTALL.md) for full platform instructions.

```bash
git clone <repo>
cd AgentTrack
go build -o atrack .
```

## Usage

### Auto-Tracking (AI Q&A)
```bash
atrack auto "user question" "AI answer summary" "model_name" tokens_in tokens_out
```

### Manual Logging
```bash
atrack log "Started research on project"
atrack log "Fixed bug in main.go" -c "Bugfix"
atrack log "Improved export flow" -c "Enhancement" -t "cli,export,go"
```

### View History
```bash
atrack list                                        # all logs (newest first)
atrack list last                                   # most recent entry
atrack list 2026-05-07                             # specific date
atrack list --from 2026-05-01 --to 2026-05-07      # date range
atrack list model "gemini-3-flash-preview"         # by model
atrack list model all                              # model usage summary
atrack list category "Bugfix"                      # by category
atrack list category all                           # category summary
```

### Search
```bash
atrack search "bug"
atrack search tag "export"
atrack search model "gemini-2.5-pro"
atrack search "fix" --from 2026-05-01 --to 2026-05-31
```

### Activity Summary *(v0.13)*
```bash
atrack summary             # today (default)
atrack summary today
atrack summary week
atrack summary month
```

### Tags *(v0.13)*
```bash
atrack tag list            # all tags with counts
```

### Stats
```bash
atrack stats               # overall totals
atrack stats today         # today only
atrack stats model         # per-model breakdown
atrack stats cost          # estimated cost
```

### Pricing Sync
```bash
atrack pricing sync                        # sync models found in logs/config
atrack pricing sync gemini-2.5-pro         # sync one specific model
atrack pricing sync all                    # sync every model from OpenRouter
```

### Edit & Delete
```bash
atrack edit 5 "Corrected message"
atrack edit 5 tags "bug,reviewed"
atrack delete 5
```

### Export
```bash
atrack export md
atrack export csv
atrack export json
```

### Real-time Watch
```bash
atrack watch               # monitor new logs live
```

### Configure
```bash
atrack config show
atrack config set display.max_logs_view 25
atrack config set pricing.gemini-2.5-pro.input_per_1k 0.00125
atrack pricing sync gemini-2.5-pro
atrack config reset
```

### Other
```bash
atrack version
atrack info
atrack clear
```

## Gemini CLI Integration

### Option 1 — Native Watch (Recommended)
Run `gemini-watch.sh` in the background. It monitors your Gemini session files and auto-logs every Q&A in real-time:
```bash
./gemini-watch.sh &          # start background watcher
gemini                       # use Gemini CLI normally — everything is tracked
```

### Option 2 — Interactive Wrapper
Use `gemitrack.sh` as a Gemini CLI wrapper with per-question logging and live model detection:
```bash
./gemitrack.sh
```

See [integrations/gemini.md](integrations/gemini.md) for full setup details.

## AI Agent Integrations

| Agent | Method |
|---|---|
| **Gemini CLI** | `gemini-watch.sh` (native) or `gemitrack.sh` (wrapper) |
| **GitHub Copilot** | See [integrations/copilot.md](integrations/copilot.md) |
| **Cursor IDE** | `.cursorrules` |
| **Cline (VS Code)** | `.clinerules` |
| **Claude Code** | See [integrations/claude-code.md](integrations/claude-code.md) |
| **Aider** | See [integrations/aider.md](integrations/aider.md) |
| **Shell-GPT** | See [integrations/sgpt.md](integrations/sgpt.md) |
| **Open Interpreter** | See [integrations/open-interpreter.md](integrations/open-interpreter.md) |
| **Qwen Code** | See [integrations/qwen-code.md](integrations/qwen-code.md) |
| **Codex CLI** | See [integrations/codex.md](integrations/codex.md) |
| **Roo Code** | See [integrations/roo-code.md](integrations/roo-code.md) |
| **Windsurf** | See [integrations/windsurf.md](integrations/windsurf.md) |
| **Continue.dev** | See [integrations/continue.md](integrations/continue.md) |

## File Structure
```
AgentTrack/
├── main.go               # Core CLI (all commands)
├── main_test.go          # Unit tests
├── go.mod                # Go module
├── atrack                # Compiled binary
├── gemitrack.sh          # Gemini CLI interactive wrapper
├── gemini-watch.sh       # Gemini CLI background session watcher
├── INSTALL.md            # Cross-platform install guide
├── AGENTS.md             # AI agent auto-logging rules
├── integrations/         # Per-agent integration guides
│   ├── gemini.md
│   ├── copilot.md
│   ├── cursor.md
│   └── ...
└── ~/.atrack/          # Data directory (auto-created)
    ├── config.json
    └── atrack_logs_YYYY_MM.json
```

## Testing
```bash
go test -v
```
Tests back up and restore existing data automatically.
