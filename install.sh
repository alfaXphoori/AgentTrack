#!/bin/bash
set -e

echo "🚀 Installing AgentTrack..."

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 1. Check for Go
if ! command -v go &> /dev/null; then
    echo "❌ Error: Go is not installed. Please install Go first."
    exit 1
fi

# 2. Build and install Go binary
echo "📦 Building and installing AgentTrack CLI globally..."
go build -o atrack ./cmd/atrack
go install ./cmd/atrack

echo "🔧 Enabling AgentTrack auto-run..."
"$REPO_DIR/atrack" autostart install

# Detect user shell profile
PROFILE_FILE=""
if [[ "$SHELL" == *"zsh"* ]]; then
    PROFILE_FILE="$HOME/.zshrc"
elif [[ "$SHELL" == *"bash"* ]]; then
    if [ -f "$HOME/.bashrc" ]; then
        PROFILE_FILE="$HOME/.bashrc"
    elif [ -f "$HOME/.bash_profile" ]; then
        PROFILE_FILE="$HOME/.bash_profile"
    else
        PROFILE_FILE="$HOME/.bashrc"
    fi
else
    PROFILE_FILE="$HOME/.profile"
fi

# Check if go/bin is in PATH
if [[ ":$PATH:" != *":$HOME/go/bin:"* ]]; then
    echo "⚠️ Warning: ~/go/bin is not in your PATH."
    if [ -n "$PROFILE_FILE" ]; then
        echo "Adding ~/go/bin to $PROFILE_FILE..."
        echo 'export PATH="$PATH:$HOME/go/bin"' >> "$PROFILE_FILE"
    fi
fi

if [ -n "$PROFILE_FILE" ]; then
    # 3. Add GitHub Copilot Wrapper
    echo "🔧 Configuring GitHub Copilot auto-log wrapper in $PROFILE_FILE..."
    if ! grep -q "gh_copilot_wrapper" "$PROFILE_FILE" 2>/dev/null; then
      cat << 'EOF' >> "$PROFILE_FILE"

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

    # 4. Add Auto-Init Hook
    echo "🪄 Configuring fully automatic background setup for AI agents..."
    if ! grep -q "atrack_auto_init" "$PROFILE_FILE" 2>/dev/null; then
        if [[ "$SHELL" == *"zsh"* ]]; then
            cat << 'EOF' >> "$PROFILE_FILE"

# AgentTrack Auto-Init Hook (Zsh)
atrack_auto_init() {
  if [ -w "." ] && [ ! -f ".cursorrules" ]; then
      atrack init >/dev/null 2>&1
  fi
}
autoload -U add-zsh-hook 2>/dev/null
add-zsh-hook chpwd atrack_auto_init 2>/dev/null
atrack_auto_init
EOF
        else
            cat << 'EOF' >> "$PROFILE_FILE"

# AgentTrack Auto-Init Hook (Bash)
atrack_auto_init() {
  if [ -w "." ] && [ ! -f ".cursorrules" ]; then
      atrack init >/dev/null 2>&1
  fi
}
if [[ ! "$PROMPT_COMMAND" == *"atrack_auto_init"* ]]; then
    export PROMPT_COMMAND="atrack_auto_init; $PROMPT_COMMAND"
fi
atrack_auto_init
EOF
        fi
        echo "  ✅ Auto-Init hook added."
    else
        echo "  ⚡ Auto-Init hook already configured."
    fi
else
    echo "⚠️ Could not determine shell profile file. Skipping hooks."
fi

echo ""
echo "🎉 AgentTrack Installation Complete!"
echo "--------------------------------------------------------"
echo "All your AI tools (Cursor, Copilot, Gemini CLI, Claude Code) are now fully automated."
echo "Whenever you cd into a project directory, AgentTrack will secretly prepare the rule files."
echo ""
if [ -n "$PROFILE_FILE" ]; then
    echo "👉 IMPORTANT: Run this command to activate the changes right now:"
    echo "source $PROFILE_FILE"
fi
