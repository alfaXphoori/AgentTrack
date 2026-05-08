# AgentTrack AI Integrations

This directory contains integration guides and rule files to help various AI coding assistants and CLIs automatically log their activities using AgentTrack.

## Supported Integrations

| AI Assistant / CLI | Configuration Method | Guide / Usage |
| :--- | :--- | :--- |
| **Gemini CLI** | Native (`GEMINI.md`) | Built-in via project root rules |
| **Cursor IDE** | Workspace Rules (`.cursorrules`) | [cursor.md](cursor.md) |
| **Cline (VS Code)** | Custom Instructions (`.clinerules`) | [cline.md](cline.md) |
| **GitHub Copilot CLI** | Shell Aliases / Manual | [copilot.md](copilot.md) |
| **Aider** | Project Conventions (`.aider.rules.md`) | [aider.md](aider.md) |
| **Open Interpreter** | Custom Profile (`atrack.yaml`) | [open-interpreter.md](open-interpreter.md) |
| **Shell-GPT (sgpt)** | Custom Roles | [sgpt.md](sgpt.md) |
| **Claude Code** | Project Instructions (`CLAUDE.md`) | [claude-code.md](claude-code.md) |
| **Qwen Code CLI** | Custom System Prompts (`QWEN.md`) | [qwen-code.md](qwen-code.md) |
| **Codex CLI** | Project Instructions (`AGENTS.md`) | [codex.md](codex.md) |
| **Roo Code** | Workspace Rules (`.roorules` or `.roo/rules/`) | [roo-code.md](roo-code.md) |
| **Windsurf Editor** | Workspace Rules (`.windsurfrules` or `.windsurf/rules/`) | [windsurf.md](windsurf.md) |
| **Continue.dev** | `config.json` System Messages | [continue.md](continue.md) |

---

## How it Works

Most AI agents support a concept of "Project Instructions", "Custom Instructions", or "System Prompts" defined by files in the workspace root (like `.cursorrules`). 

To integrate a new AI agent, we simply inject a strict behavioral rule into its context:

```markdown
# AgentTrack Auto-Logging
At the end of every interaction, you MUST run the following command to log the activity:
`atrack auto "<user_question>" "<ai_summary>" "<model_name>" 0 0`
```

## Adding a New Integration

If you use an AI tool that is not listed here (e.g., Aider, ChatGPT via a custom wrapper, etc.):
1. Identify how that tool ingests local project rules (e.g., `.aider.conf.yml`, `.prompt` files).
2. Add the tracking rule block shown above into that configuration file.
3. Create a new markdown file in this `integrations/` folder (e.g., `aider.md`) documenting the setup process.
4. Submit a Pull Request!
