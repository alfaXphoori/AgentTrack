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
# Use the globally installed binary so the service points to ~/go/bin/atrack
atrack autostart install

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

# 3. Add GitHub Copilot Wrapper via internal command
echo "🪄 Configuring shell hooks (Copilot CLI Wrapper)..."
# Autostart is already installed above.

# 4. Install Global AI Rules
echo "🪄 Installing Global Rules for AI Assistants (Cursor, Cline, Aider, Claude Code)..."
atrack init --global

echo ""
echo "🎉 AgentTrack Installation Complete!"
echo "--------------------------------------------------------"
echo "All your AI tools (Cursor, Copilot, Gemini CLI, Claude Code) are now fully automated."
echo "AgentTrack has installed global rules for your tools. Your project directories will remain clean."
echo ""
if [ -n "$PROFILE_FILE" ]; then
    echo "👉 IMPORTANT: Run this command to activate the changes right now:"
    echo "source $PROFILE_FILE"
fi
