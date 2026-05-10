#!/bin/bash

# auto-run-service.sh
# Manages the AgentTrack background watcher based on config

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
ATRACK_BIN="$REPO_DIR/atrack"

if [ ! -x "$ATRACK_BIN" ]; then
	ATRACK_BIN="$(command -v atrack)"
fi

exec "$ATRACK_BIN" autostart run
