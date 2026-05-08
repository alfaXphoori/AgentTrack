#!/bin/bash
# vscode-copilot-watcher.sh - Background watcher for VS Code Copilot Chat sessions

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PYTHON_SCRIPT="$SCRIPT_DIR/vscode-copilot-watcher.py"

if [ ! -f "$PYTHON_SCRIPT" ]; then
    echo "❌ Error: Python script not found at $PYTHON_SCRIPT"
    exit 1
fi

python3 "$PYTHON_SCRIPT"
