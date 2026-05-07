# TrackCLI — AI Activity Tracker `v0.13`

A cross-platform terminal tool for tracking tasks and AI interactions. Automatically logs questions, answers, models, and token usage from your AI CLI sessions — with real-time Gemini CLI watching, activity summaries, and tag management.

## What's New in v0.13
- **`trackcli summary [today|week|month]`** — Activity summary with model breakdown, token totals, and tag usage
- **`trackcli tag list`** — View all tags and their counts across all logs
- **`trackcli stats today`** — Quick today-only token and log stats
- **`trackcli pricing sync [all|model ...]`** — Pull the latest OpenRouter pricing into local config
- **`gemini-watch.sh`** — Background watcher that auto-logs native Gemini CLI sessions in real-time (no wrapper needed)
- **Bug fix:** Timezone-aware date filtering in summary commands

## Features
- **Auto-Tracking:** Logs AI questions, answers, models, and token counts automatically
- **Gemini CLI Native Watch:** `gemini-watch.sh` monitors live Gemini sessions without any wrapper
- **Activity Summaries:** Daily, weekly, and monthly breakdowns with model and tag stats
- **Tag System:** Tag logs and search/list by tag
- **Manual Logging:** Add entries with categories, tags, and timestamps
- **Real-time Watch:** `trackcli watch` monitors new logs live
- **Stats & Cost:** Per-model token usage and estimated cost via OpenRouter pricing
- **OpenRouter Price Sync:** Pull the latest model prices from OpenRouter and save updates into config
- **Export:** Export logs to Markdown, CSV, or JSON
- **Date Filtering:** Filter any command with `--from` / `--to`
- **Config:** Timezone, token estimation, display preferences, model pricing
- **Monthly Log Rotation:** Logs stored per-month in `~/.trackcli/`

## Setup
Requires **Go (Golang)** installed. See [INSTALL.md](INSTALL.md) for full platform instructions.

```bash
git clone <repo>
cd Track_CLI
go build -o trackcli .
```

## Usage

### Auto-Tracking (AI Q&A)
```bash
trackcli auto "user question" "AI answer summary" "model_name" tokens_in tokens_out
```

### Manual Logging
```bash
trackcli log "Started research on project"
trackcli log "Fixed bug in main.go" -c "Bugfix"
trackcli log "Improved export flow" -c "Enhancement" -t "cli,export,go"
```

### View History
```bash
trackcli list                                        # all logs (newest first)
trackcli list last                                   # most recent entry
trackcli list 2026-05-07                             # specific date
trackcli list --from 2026-05-01 --to 2026-05-07      # date range
trackcli list model "gemini-3-flash-preview"         # by model
trackcli list model all                              # model usage summary
trackcli list category "Bugfix"                      # by category
trackcli list category all                           # category summary
```

### Search
```bash
trackcli search "bug"
trackcli search tag "export"
trackcli search model "gemini-2.5-pro"
trackcli search "fix" --from 2026-05-01 --to 2026-05-31
```

### Activity Summary *(v0.13)*
```bash
trackcli summary             # today (default)
trackcli summary today
trackcli summary week
trackcli summary month
```

### Tags *(v0.13)*
```bash
trackcli tag list            # all tags with counts
```

### Stats
```bash
trackcli stats               # overall totals
trackcli stats today         # today only
trackcli stats model         # per-model breakdown
trackcli stats cost          # estimated cost
```

### Pricing Sync
```bash
trackcli pricing sync                        # sync models found in logs/config
trackcli pricing sync gemini-2.5-pro         # sync one specific model
trackcli pricing sync all                    # sync every model from OpenRouter
```

### Edit & Delete
```bash
trackcli edit 5 "Corrected message"
trackcli edit 5 tags "bug,reviewed"
trackcli delete 5
```

### Export
```bash
trackcli export md
trackcli export csv
trackcli export json
```

### Real-time Watch
```bash
trackcli watch               # monitor new logs live
```

### Configure
```bash
trackcli config show
trackcli config set display.max_logs_view 25
trackcli config set pricing.gemini-2.5-pro.input_per_1k 0.00125
trackcli pricing sync gemini-2.5-pro
trackcli config reset
```

### Other
```bash
trackcli version
trackcli info
trackcli clear
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
Track_CLI/
├── main.go               # Core CLI (all commands)
├── main_test.go          # Unit tests
├── go.mod                # Go module
├── trackcli              # Compiled binary
├── gemitrack.sh          # Gemini CLI interactive wrapper
├── gemini-watch.sh       # Gemini CLI background session watcher
├── INSTALL.md            # Cross-platform install guide
├── AGENTS.md             # AI agent auto-logging rules
├── integrations/         # Per-agent integration guides
│   ├── gemini.md
│   ├── copilot.md
│   ├── cursor.md
│   └── ...
└── ~/.trackcli/          # Data directory (auto-created)
    ├── config.json
    └── trackcli_logs_YYYY_MM.json
```

## Testing
```bash
go test -v
```
Tests back up and restore existing data automatically.
