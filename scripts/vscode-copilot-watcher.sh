#!/bin/bash
# vscode-copilot-watcher.sh - Simple launcher for the AgentTrack VS Code Copilot watcher

ATRACK_BIN="$(command -v atrack)"

# Guard: only one instance running
LOCK_FILE="/tmp/vscode-copilot-atrack.lock"
if [ -f "$LOCK_FILE" ] && kill -0 "$(cat $LOCK_FILE)" 2>/dev/null; then
  exit 0
fi
echo $$ > "$LOCK_FILE"
trap "rm -f $LOCK_FILE" EXIT
trap '' HUP

exec "$ATRACK_BIN" internal-watch-copilot