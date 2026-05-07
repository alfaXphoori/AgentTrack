#!/bin/bash
# track-base.sh - Shared utilities for TrackCLI passive watchers

TRACKCLI_BIN="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/trackcli"

# Log to TrackCLI
log_activity() {
    local q="$1"
    local a="$2"
    local m="$3"
    local ti="${4:-0}"
    local to="${5:-0}"
    
    # Use head to summarize answer if too long
    local summary=$(echo "$a" | head -n 1 | cut -c1-80)
    
    "$TRACKCLI_BIN" auto "$q" "$summary" "$m" "$ti" "$to" >/dev/null 2>&1
}

# State management
get_state_file() {
    local tool_name="$1"
    local state_dir="$(pwd)/.trackcli_watch_state"
    mkdir -p "$state_dir"
    echo "$state_dir/${tool_name}.state"
}
