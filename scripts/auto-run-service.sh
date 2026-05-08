#!/bin/bash

# auto-run-service.sh
# Manages the AgentTrack background watcher based on config

REPO_DIR="/Users/phoori/Library/CloudStorage/GoogleDrive-phoori.ch@ksu.ac.th/My Drive/KSU/Git/AgentTrack"
WATCHER_SCRIPT="$REPO_DIR/scripts/gemini-cli-atrack.sh"
ATRACK_BIN="$REPO_DIR/atrack"

# Function to check if auto_run is enabled in config
is_auto_run_enabled() {
    # Extract auto_run value using a simple grep/sed since we don't want to depend on jq
    # Assuming the config is at ~/.atrack/config.json
    local config_file="$HOME/.atrack/config.json"
    if [ ! -f "$config_file" ]; then
        return 1 # Disabled if no config
    fi
    
    local val=$(grep '"auto_run"' "$config_file" | cut -d: -f2 | tr -d ' ,"' | xargs)
    if [ "$val" == "true" ]; then
        return 0
    else
        return 1
    fi
}

while true; do
    if is_auto_run_enabled; then
        # Check if watcher is already running
        if ! pgrep -f "$WATCHER_SCRIPT" > /dev/null; then
            echo "$(date): AutoRun is ON. Starting watcher..."
            bash "$WATCHER_SCRIPT" &
        fi
    else
        # If running but disabled in config, kill it
        if pgrep -f "$WATCHER_SCRIPT" > /dev/null; then
            echo "$(date): AutoRun is OFF. Stopping watcher..."
            pkill -f "$WATCHER_SCRIPT"
        fi
    fi
    sleep 30 # Check every 30 seconds
done
