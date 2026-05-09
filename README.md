<div align="center">

# 🤖 AgentTrack

### ⚡ AI Activity Tracker for the Terminal

[![Version](https://img.shields.io/badge/version-0.13.3-blue?style=for-the-badge)](https://github.com/alfaXphoori/AgentTrack/releases)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-green?style=for-the-badge)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-lightgrey?style=for-the-badge)](#installation)

Track every AI interaction across **Cursor, Copilot, Gemini CLI, Claude Code, Aider** and more — *automatically*.  
Token counts, costs, summaries, and a full TUI dashboard — all directly in your terminal.

[Installation](#-installation) • [Quick Start](#-quick-start) • [Features](#-features) • [Dashboard](#-dashboard) • [Usage](#-usage) • [Integrations](#-ai-agent-integrations)

</div>

---

## ✨ What's New in v0.13.3

| Feature | Description |
|:---|:---|
| 📈 **Dashboard: Trends** | 30-day daily activity bar chart — switch between Logs / Tokens / Cost. |
| 💰 **Dashboard: Cost** | Today / week / month / all-time cost summary + per-model breakdown. |
| 📊 **Dashboard: Stats** | Interactive ASCII bar chart per model with metric selector. |
| 🔍 **Dashboard: Logs** | Filter by Keyword / Model / Category + **Live Watch** (auto-refresh every 2s). |
| ⚙️ **Dashboard: Settings** | Export buttons (MD / CSV / JSON) at the top of the settings tab. |

---

## 📦 Installation

<details open>
<summary><b>🍺 Homebrew (macOS / Linux) — Recommended</b></summary>

```bash
brew tap alfaXphoori/agenttrack
brew install atrack
```

</details>

<details>
<summary><b>💻 macOS / Linux — Build from Source</b></summary>

```bash
git clone https://github.com/alfaXphoori/AgentTrack.git
cd AgentTrack
go build -o atrack .
go install .
```

</details>

<details>
<summary><b>🐹 macOS / Linux — via Go</b></summary>

```bash
go install github.com/alfaXphoori/AgentTrack@latest
```

</details>

<details>
<summary><b>🐧 Linux — Pre-compiled Binary</b></summary>

Download the latest release from [GitHub Releases](https://github.com/alfaXphoori/AgentTrack/releases):
```bash
tar -xzf atrack_linux_amd64.tar.gz
sudo mv atrack /usr/local/bin/
```

</details>

<details>
<summary><b>🪟 Windows — Build from Source</b></summary>

```powershell
git clone https://github.com/alfaXphoori/AgentTrack.git
cd AgentTrack
go build -o atrack.exe .
```

</details>

<br>

> 📚 See [docs/INSTALL.md](docs/INSTALL.md) for full platform instructions.

---

## 🚀 Quick Start

Get up and running in seconds:

```bash
# Initialize rules for all AI agents in your project
atrack init

# Open the TUI dashboard to view your tracking
atrack dashboard
```

---

## 🎯 Features

- 🔄 **Auto-Tracking:** Logs AI questions, answers, models, and token counts seamlessly in the background.
- 📺 **TUI Dashboard:** 7-tab interactive dashboard: *Overview · Logs · Stats · Trends · Cost · Search · Settings*.
- 🔴 **Live Watch:** Real-time log streaming directly inside the dashboard.
- 💰 **Cost Tracking:** Per-model cost estimation via OpenRouter pricing.
- 🔃 **OpenRouter Sync:** Pull the latest model prices with a single command.
- 🔍 **Search & Filter:** Full-text search with date range, model, category, and tag filters.
- 📤 **Export:** Easily export your logs to Markdown, CSV, or JSON formats.
- 🤖 **12+ Integrations:** Supports Cursor, Copilot, Gemini CLI, Claude Code, Aider, and many more.
- 🗓️ **Monthly Log Rotation:** Logs are automatically stored per-month in `~/.atrack/` to keep things tidy.

---

## 📺 Dashboard

Launch the dashboard with `atrack dashboard`. Navigate effortlessly using keys `1`–`7`:

| Key | Tab | Description |
|:---:|:---|:---|
| `1` | **Overview** | Daily / weekly / monthly snapshot + recent activity. |
| `2` | **Logs** | Full log table with filter bar + live watch mode. |
| `3` | **Stats** | ASCII bar chart showing usage by model. |
| `4` | **Trends** | 30-day activity bar chart for quick insights. |
| `5` | **Cost** | Cost summary + detailed per-model cost breakdown. |
| `6` | **Search** | Full-text search with a detailed view panel. |
| `7` | **Settings** | Configuration, export options, and log management. |

---

## 📖 Usage

### 📝 Logging

```bash
# Auto-Logging (AI Q&A)
atrack auto "user question" "AI answer summary" "model_name" tokens_in tokens_out

# Manual Logging
atrack log "Started research on project"
atrack log "Fixed auth bug" -c "Bugfix" -t "auth,security"
```

### 🔍 Viewing & Searching

```bash
# View Logs
atrack list                                       # all logs
atrack list last                                  # most recent
atrack list 2026-05-07                            # by date
atrack list --from 2026-05-01 --to 2026-05-07     # date range
atrack list model "gemini-2.5-pro"                # by model
atrack list category "Bugfix"                     # by category

# Search
atrack search "bug fix"
atrack search tag "export"
atrack search model "gemini-2.5-pro"
atrack search "auth" --from 2026-05-01 --to 2026-05-31
```

### 📊 Analytics & Cost

```bash
# Activity Summary
atrack summary             # today (default)
atrack summary week
atrack summary month

# Stats & Cost
atrack stats               # overall totals
atrack stats today         # today only
atrack stats model         # per-model breakdown
atrack stats cost          # estimated cost per model

# Pricing Sync (OpenRouter)
atrack pricing sync                    # sync models in logs/config
atrack pricing sync gemini-2.5-pro     # sync one model
atrack pricing sync all                # sync every model
```

### 🛠️ Management & Config

```bash
# Edit & Delete
atrack edit 5 "Corrected message"
atrack edit 5 tags "bug,reviewed"
atrack delete 5

# Export
atrack export md
atrack export csv
atrack export json

# Configure
atrack config show
atrack config set display.max_logs_view 25
atrack config set pricing.currency THB
atrack config reset
```

---

## 🤖 AI Agent Integrations

Run `atrack init` in any project to auto-generate rule files for all supported agents.

| Agent | Integration |
|:---|:---|
| **Gemini CLI** | [`gemini-cli-atrack.sh`](scripts/gemini-cli-atrack.sh) native watcher or [`gemiatrack.sh`](scripts/gemiatrack.sh) wrapper |
| **GitHub Copilot** | [integrations/copilot.md](integrations/copilot.md) |
| **Cursor IDE** | `.cursorrules` *(auto-generated)* |
| **Cline (VS Code)** | `.clinerules` *(auto-generated)* |
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

```text
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

## 📄 License

MIT © [alfaXphoori](https://github.com/alfaXphoori)

