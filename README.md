# TrackCLI Activity Tracker

A program for tracking tasks and logging via the Terminal, developed to help record various activities while using AI CLIs. It features an Auto-Tracking system for automatically recording questions and answers.

## Features
- **(NEW) AI Integrations:** Supports automatic logging from multiple platforms (Gemini, Cursor, Cline, GitHub Copilot, Aider, Open Interpreter, Shell-GPT).
- **Auto-Tracking:** Automatically logs User questions and AI answers in every conversation.
- Manual logging with automatic timestamps.
- Categorize logs by type.
- View past history through the Terminal with clear Q&A formatting.
- Data is stored in a JSON file (`trackcli_logs.json`).

## Setup
This program is developed using **Go (Golang)**. You must have Go installed on your machine.

1. Check your Go installation with `go version`.
2. You can run it immediately using the command `go run .` or compile it into a binary with `go build`.

## Usage

### 1. Auto-Tracking System (Q&A)
The system is configured for AI (like Gemini CLI) to automatically run this command every time it answers a question:
```bash
go run . auto "user's question" "summary of AI's answer" "model_name" tokens_in tokens_out
```

### 2. Manual Logging
```bash
go run . log "Started research on project components"
```

Specify a category:
```bash
go run . log "Fixed a bug in main.go" -c "Bugfix"
```

Add tags:
```bash
go run . log "Improved export flow" -c "Enhancement" -t "cli,export,go"
```

### 3. View All History
```bash
go run . list
```
*(For AutoLogs, the system displays Q: and A: on separate lines for readability)*

Filter by date range:
```bash
go run . list --from 2026-05-01 --to 2026-05-06
```

List by model:
```bash
go run . list model "gemini-1.5-flash"
```

Show model usage counts, total tokens, and estimated cost for all logged models:
```bash
go run . list model all
```

List by category:
```bash
go run . list category "Bugfix"
```

Show usage counts for all categories:
```bash
go run . list category all
```

Search by keyword:
```bash
go run . search "bug"
```

Search by tag:
```bash
go run . search tag "export"
```

Search by model:
```bash
go run . search model "gemini-1.5-flash"
```

Search inside a date range:
```bash
go run . search "bug" --from 2026-05-01 --to 2026-05-31
```

### 4. Edit a Log
Update a manual log message:
```bash
go run . edit 5 "Corrected message"
```

Update a specific field:
```bash
go run . edit 5 tags "bug,reviewed"
```

### 5. Clear All Logs
```bash
go run . clear
```

### 6. Stats and Export
Show overall stats:
```bash
go run . stats
```

Show per-model token stats:
```bash
go run . stats model
```

Show estimated token cost using configured pricing:
```bash
go run . stats cost
```

Export to Markdown, CSV, or JSON:
```bash
go run . export md
go run . export csv
go run . export json
```

### 7. Configure the App
Show the current config:
```bash
go run . config show
```

Update one setting:
```bash
go run . config set display.max_logs_view 25
```

Read one setting:
```bash
go run . config get timezone
```

Set model pricing for cost estimation:
```bash
go run . config set pricing.gpt-5.4.input_per_1k 0.003
go run . config set pricing.gpt-5.4.output_per_1k 0.012
```

Reset everything to defaults:
```bash
go run . config reset
```

## AI Agent Integrations
TrackCLI is designed to work seamlessly with various AI Agents through rule files:
- **Gemini CLI:** Uses `GEMINI.md`
- **Cursor IDE:** Uses `.cursorrules`
- **Cline (VS Code):** Uses `.clinerules`
- **GitHub Copilot:** See the guide in `integrations/copilot.md`
- **Aider:** See the guide in `integrations/aider.md`
- **Open Interpreter:** See the guide in `integrations/open-interpreter.md`
- **Shell-GPT (sgpt):** See the guide in `integrations/sgpt.md`
- **Claude Code:** See the guide in `integrations/claude-code.md`
- **Qwen Code:** See the guide in `integrations/qwen-code.md`
- **Codex CLI:** See the guide in `integrations/codex.md`
- **Roo Code:** See the guide in `integrations/roo-code.md`
- **Windsurf Editor:** See the guide in `integrations/windsurf.md`
- **Continue.dev:** See the guide in `integrations/continue.md`

See the `integrations/` folder for more details.

## File Structure
- `main.go`: Main program file (Golang)
- `main_test.go`: Go Unit Test file
- `go.mod`: Go Dependency manager
- `config.json`: Configuration file (Timezone, Tokens, Display)
- `.gitignore`: Configured to prevent committing log and backup files to Git
- `trackcli_logs.json`: Log data file (Ignored by Git)
- `README.md`: Project details
- `GEMINI.md`: Rules forcing the AI to log automatically
- `INSTALL.md`: Cross-platform installation instructions

## Testing
You can run Unit Tests to verify functionality using the command:
```bash
go test -v
```
The system will back up existing data before testing and automatically restore it upon completion.
