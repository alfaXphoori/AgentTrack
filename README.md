<div align="center">

# 🤖 AgentTrack

### AI Activity Tracker for the Terminal

[![Version](https://img.shields.io/badge/version-0.13.3-blue?style=flat-square)](https://github.com/alfaXphoori/AgentTrack/releases)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-green?style=flat-square)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-lightgrey?style=flat-square)](#installation)

Track every AI interaction across **Cursor, Copilot, Gemini CLI, Claude Code, Aider** and more — automatically.  
Token counts, costs, summaries, and a full TUI dashboard — all in your terminal.

</div>

---

## ✨ What's New in v0.13.3

| Feature | Description |
|---|---|
| 📈 **Dashboard: Trends Tab** | 30-day daily activity bar chart — switch between Logs / Tokens / Cost |
| 💰 **Dashboard: Cost Tab** | Today / week / month / all-time cost summary + per-model breakdown |
| 📊 **Dashboard: Stats Tab** | Interactive ASCII bar chart per model with metric selector |
| 🔍 **Dashboard: Logs Tab** | Filter by Keyword / Model / Category + **Live Watch** (auto-refresh every 2s) |
| ⚙️ **Dashboard: Settings Tab** | Export buttons (MD / CSV / JSON) at the top |

---

## 📦 Installation

### macOS / Linux — Build from Source
```bash
git clone https://github.com/alfaXphoori/AgentTrack.git
cd AgentTrack
go build -o atrack .
go install .
```

### macOS / Linux — via Go
```bash
go install github.com/alfaXphoori/AgentTrack@latest
```

### Linux — Pre-compiled Binary
Download the latest release from [GitHub Releases](https://github.com/alfaXphoori/AgentTrack/releases):
```bash
tar -xzf atrack_linux_amd64.tar.gz
sudo mv atrack /usr/local/bin/
```

### Windows — Build from Source
```powershell
git clone https://github.com/alfaXphoori/AgentTrack.git
cd AgentTrack
go build -o atrack.exe .
```

> See [docs/INSTALL.md](docs/INSTALL.md) for full platform instructions.

---

## 🚀 Quick Start

```bash
# Clone and build
git clone https://github.com/alfaXphoori/AgentTrack.git
cd AgentTrack
go build -o atrack .
go install .

# Initialize rules for all AI agents in your project
atrack init

# Open the TUI dashboard
atrack dashboard
```

> See [docs/INSTALL.md](docs/INSTALL.md) for full platform instructions (macOS / Linux / Windows).

---

## 🎯 Features

- 🔄 **Auto-Tracking** — Logs AI questions, answers, models, and token counts automatically
- 📺 **TUI Dashboard** — 7-tab interactive dashboard: Overview · Logs · Stats · Trends · Cost · Search · Settings
- 🔴 **Live Watch** — Real-time log stream inside the dashboard
- 💰 **Cost Tracking** — Per-model cost estimation via OpenRouter pricing
- 🔃 **OpenRouter Sync** — Pull latest model prices with one command
- 🔍 **Search & Filter** — Full-text search with date range, model, category, tag filters
- 📤 **Export** — Export logs to Markdown, CSV, or JSON
- 🤖 **12+ Agent Integrations** — Cursor, Copilot, Gemini CLI, Claude Code, Aider, and more
- 🗓️ **Monthly Log Rotation** — Logs stored per-month in `~/.atrack/`

---

## 📺 Dashboard

Launch with `atrack dashboard` — navigate with keys `1`–`7`:

| Key | Tab | Description |
|:---:|---|---|
| `1` | **Overview** | Daily / weekly / monthly snapshot + recent activity |
| `2` | **Logs** | Full log table with filter bar + live watch mode |
| `3` | **Stats** | ASCII bar chart usage by model |
| `4` | **Trends** | 30-day activity bar chart |
| `5` | **Cost** | Cost summary + per-model cost breakdown |
| `6` | **Search** | Full-text search with detail panel |
| `7` | **Settings** | Config, export, and log management |

---

## 📖 Usage

### Auto-Logging (AI Q&A)
```bash
atrack auto "user question" "AI answer summary" "model_name" tokens_in tokens_out
```

### Manual Logging
```bash
atrack log "Started research on project"
atrack log "Fixed auth bug" -c "Bugfix" -t "auth,security"
```

### View Logs
```bash
atrack list                                       # all logs
atrack list last                                  # most recent
atrack list 2026-05-07                            # by date
atrack list --from 2026-05-01 --to 2026-05-07    # date range
atrack list model "gemini-2.5-pro"               # by model
atrack list category "Bugfix"                    # by category
```

### Search
```bash
atrack search "bug fix"
atrack search tag "export"
atrack search model "gemini-2.5-pro"
atrack search "auth" --from 2026-05-01 --to 2026-05-31
```

### Activity Summary
```bash
atrack summary             # today (default)
atrack summary week
atrack summary month
```

### Stats & Cost
```bash
atrack stats               # overall totals
atrack stats today         # today only
atrack stats model         # per-model breakdown
atrack stats cost          # estimated cost per model
```

### Pricing Sync (OpenRouter)
```bash
atrack pricing sync                    # sync models in logs/config
atrack pricing sync gemini-2.5-pro     # sync one model
atrack pricing sync all                # sync every model
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

### Configure
```bash
atrack config show
atrack config set display.max_logs_view 25
atrack config set pricing.currency THB
atrack config reset
```

---

## 🤖 AI Agent Integrations

Run `atrack init` in any project to auto-generate rule files for all supported agents.

| Agent | Integration |
|---|---|
| **Gemini CLI** | [`gemini-cli-atrack.sh`](scripts/gemini-cli-atrack.sh) native watcher or [`gemiatrack.sh`](scripts/gemiatrack.sh) wrapper |
| **GitHub Copilot** | [integrations/copilot.md](integrations/copilot.md) |
| **Cursor IDE** | `.cursorrules` (auto-generated) |
| **Cline (VS Code)** | `.clinerules` (auto-generated) |
| **Claude Code** | [integrations/claude-code.md](integrations/claude-code.md) |
| **Aider** | [integrations/aider.md](integrations/aider.md) |
| **Roo Code** | [integrations/roo-code.md](integrations/roo-code.md) |
| **Windsurf** | [integrations/windsurf.md](integrations/windsurf.md) |
| **Qwen Code** | [integrations/qwen-code.md](integrations/qwen-code.md) |
| **Codex CLI** | [integrations/codex.md](integrations/codex.md) |
| **Shell-GPT** | [integrations/sgpt.md](integrations/sgpt.md) |
| **Open Interpreter** | [integrations/open-interpreter.md](integrations/open-interpreter.md) |
| **Continue.dev** | [integrations/continue.md](integrations/continue.md) |

---

## 📁 Project Structure

```
AgentTrack/
├── main.go                    # Core CLI — all commands
├── dashboard.go               # TUI Dashboard (tview)
├── timezones.go               # Timezone utilities
├── go.mod / go.sum
├── scripts/
│   ├── gemini-cli-atrack.sh   # Gemini CLI background watcher
│   ├── gemiatrack.sh          # Gemini CLI interactive wrapper
│   └── atrack-base.sh         # Shared utilities
├── docs/
│   └── INSTALL.md             # Cross-platform install guide
├── integrations/              # Per-agent integration guides
└── ~/.atrack/                 # Data directory (auto-created)
    ├── config.json
    └── atrack_logs_YYYY_MM.json
```

---

## 🧪 Testing

```bash
go test ./...
```

Tests automatically back up and restore existing log data.

---

## 📄 License

MIT © [alfaXphoori](https://github.com/alfaXphoori)

