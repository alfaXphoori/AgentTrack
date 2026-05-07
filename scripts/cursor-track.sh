#!/bin/bash
# cursor-track.sh - Passive tracker for Cursor IDE
source "$(dirname "$0")/track-base.sh"

STATE_FILE=$(get_state_file "cursor")
DB_PATH=$(ls ~/Library/Application\ Support/Cursor/User/workspaceStorage/*/state.vscdb 2>/dev/null | head -n 1)

if [ -z "$DB_PATH" ]; then
    echo "❌ Cursor state DB not found."
    exit 1
fi

echo "🔍 Cursor tracker started..."

while true; do
    # Query the SQLite DB for chat messages
sqlite3 "$DB_PATH" "SELECT value FROM ItemTable WHERE key LIKE '%workbench.panel.chat%';" | while read -r value; do
    # Parsing logic would go here
    echo "Found chat data: ${value:0:50}..."
done

    sleep 5
done
