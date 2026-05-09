#!/bin/bash
set -e

echo "🚀 Installing AgentTrack..."

# 1. Check for Go
if ! command -v go &> /dev/null; then
    echo "❌ Error: Go is not installed. Please install Go first."
    exit 1
fi

# 2. Build and install Go binary
echo "📦 Building and installing AgentTrack CLI globally..."
go build -o atrack ./cmd/atrack
go install ./cmd/atrack

# Check if go/bin is in PATH
if [[ ":$PATH:" != *":$HOME/go/bin:"* ]]; then
    echo "⚠️ Warning: ~/go/bin is not in your PATH."
    echo "Adding ~/go/bin to ~/.zshrc..."
    echo 'export PATH="$PATH:$HOME/go/bin"' >> "$HOME/.zshrc"
fi

ZSHRC="$HOME/.zshrc"

# 3. Add GitHub Copilot Wrapper to Zsh
echo "🔧 Configuring GitHub Copilot auto-log wrapper..."
if ! grep -q "gh_copilot_wrapper" "$ZSHRC"; then
  cat << 'EOF' >> "$ZSHRC"

# AgentTrack GitHub Copilot Wrapper
gh_copilot_wrapper() {
  if [ "$1" = "copilot" ] && [ "$2" = "suggest" -o "$2" = "explain" ]; then
    command gh "$@"
    atrack auto "$*" "Copilot query executed" "gh-copilot" 0 0 >/dev/null 2>&1
  else
    command gh "$@"
  fi
}
alias gh="gh_copilot_wrapper"
EOF
  echo "  ✅ Copilot wrapper added."
else
  echo "  ⚡ Copilot wrapper already configured."
fi

# 4. Add Auto-Init Hook to Zsh (The Magic Magic)
echo "🪄 Configuring fully automatic background setup for AI agents..."
if ! grep -q "atrack_auto_init" "$ZSHRC"; then
  cat << 'EOF' >> "$ZSHRC"

# AgentTrack Auto-Init Hook
# Runs silently whenever you enter ANY directory
atrack_auto_init() {
  # Check if we have write permission and haven't already injected rules
  if [ -w "." ] && [ ! -f ".cursorrules" ]; then
      atrack init >/dev/null 2>&1
  fi
}
# Attach to chpwd hook in zsh (runs every time you cd into a directory)
autoload -U add-zsh-hook 2>/dev/null
add-zsh-hook chpwd atrack_auto_init 2>/dev/null
# Run once for the current startup directory
atrack_auto_init
EOF
  echo "  ✅ Auto-Init hook added."
else
  echo "  ⚡ Auto-Init hook already configured."
fi

# 5. Start Background Watchers
echo "👁️ Setting up Background Watchers (Gemini & VS Code Copilot)..."
bash ./scripts/gemini-cli-atrack.sh &>/dev/null &
bash ./scripts/vscode-copilot-watcher.sh &>/dev/null &

echo ""
echo "🎉 AgentTrack Installation Complete!"
echo "--------------------------------------------------------"
echo "All your AI tools (Cursor, Copilot, Gemini CLI, Claude Code) are now fully automated."
echo "Whenever you cd into a project directory, AgentTrack will secretly prepare the rule files."
echo ""
echo "👉 IMPORTANT: Run this command to activate the changes right now:"
echo "source ~/.zshrc"
