# atrack-base.sh - Shared utilities for AgentTrack passive watchers
ATRACK_BIN="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/atrack"

# Log to AgentTrack
log_to_atrack() {
    local q="$1"
    local summary="$2"
    local m="$3"
    local ti="$4"
    local to="$5"
    
    "$ATRACK_BIN" auto "$q" "$summary" "$m" "$ti" "$to" >/dev/null 2>&1
}

# State directory for watchers
get_state_dir() {
    local state_dir="$(pwd)/.atrack_watch_state"
    mkdir -p "$state_dir"
    echo "$state_dir"
}
