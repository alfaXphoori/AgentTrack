#!/bin/bash
# gemini-cli-atrack.sh - Background watcher: auto-logs ALL Gemini CLI sessions to AgentTrack

ATRACK_BIN="$(command -v atrack)"

# Guard: only one instance running
LOCK_FILE="/tmp/gemini-cli-atrack.lock"
if [ -f "$LOCK_FILE" ] && kill -0 "$(cat $LOCK_FILE)" 2>/dev/null; then
  exit 0
fi
echo $$ > "$LOCK_FILE"
trap "rm -f $LOCK_FILE" EXIT
trap '' HUP

"$ATRACK_BIN" internal-watch-gemini