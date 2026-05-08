#!/bin/bash
# gemini-cli-atrack.sh - Background watcher: auto-logs native Gemini CLI sessions to AgentTrack

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ATRACK_BIN="$(command -v atrack || echo "$SCRIPT_DIR/../atrack")"
PROJECT_DIR="$(pwd)"
POLL_INTERVAL=2
STATE_DIR="$PROJECT_DIR/.atrack_watch_state"
mkdir -p "$STATE_DIR"

# Find ~/.gemini/tmp/<project>/chats for current project
CHATS_DIR=$(python3 - "$PROJECT_DIR" << 'PYEOF'
import os, sys
cwd = sys.argv[1]
tmp_base = os.path.expanduser('~/.gemini/tmp')
if not os.path.exists(tmp_base): sys.exit(1)
for d in os.listdir(tmp_base):
    pr = os.path.join(tmp_base, d, '.project_root')
    if os.path.exists(pr):
        with open(pr) as f:
            if f.read().strip().lower() == cwd.lower():
                print(os.path.join(tmp_base, d, 'chats'))
                sys.exit(0)
sys.exit(1)
PYEOF
)

if [ -z "$CHATS_DIR" ]; then
  echo "❌ No Gemini session directory found for: $PROJECT_DIR"
  echo "   Open gemini in this directory first, then re-run."
  exit 1
fi

printf "\033[1;32m🔍 AgentTrack Gemini Watcher started\033[0m\n"
printf "   Watching: %s\n" "$CHATS_DIR"
printf "   Poll every: ${POLL_INTERVAL}s | Press Ctrl+C to stop\n\n"

# Process a session file: log any NEW complete Q&A pairs not yet tracked
process_session() {
  local FILE="$1"
  local HASH=$(echo "$FILE" | md5)
  local STATE_FILE="$STATE_DIR/${HASH}.logged"
  local LOGGED=$(cat "$STATE_FILE" 2>/dev/null || echo "0")

  python3 - "$FILE" "$LOGGED" "$ATRACK_BIN" "$STATE_FILE" << 'PYEOF'
import json, os, sys, subprocess, datetime

file_path   = sys.argv[1]
logged_pairs = int(sys.argv[2])
atrack      = sys.argv[3]
state_file  = sys.argv[4]

def parse_iso(ts):
    try: return datetime.datetime.fromisoformat(ts.replace('Z', '+00:00'))
    except: return None

# Parse all turns from the session
turns = []
session_id = ""
try:
    with open(file_path) as f:
        for line in f:
            line = line.strip()
            if not line: continue
            try:
                d = json.loads(line)
                if 'sessionId' in d:
                    session_id = d['sessionId']
                    continue
                
                t = d.get('type')
                model = d.get('model', '')
                content = d.get('content', '')
                ts = d.get('timestamp')
                tools = []
                if 'toolCalls' in d:
                    for call in d['toolCalls']:
                        if 'name' in call:
                            tools.append(call['name'])

                if t in ('user', 'gemini') and (content or tools):
                    text = content if isinstance(content, str) else ''
                    if isinstance(content, list):
                        for c in content:
                            if isinstance(c, dict) and 'text' in c:
                                text += c['text']
                    
                    if text.strip() or tools:
                        turns.append({
                            'type': t, 
                            'text': text.strip(), 
                            'model': model, 
                            'ts': parse_iso(ts),
                            'tools': tools
                        })
            except:
                pass
except:
    sys.exit(0)

# Build complete Q&A pairs
pairs = []
i = 0
last_model = 'gemini'
while i < len(turns):
    if turns[i]['type'] == 'user':
        if i+1 < len(turns) and turns[i+1]['type'] == 'gemini':
            model = turns[i+1].get('model') or last_model
            duration = 0.0
            if turns[i]['ts'] and turns[i+1]['ts']:
                delta = turns[i+1]['ts'] - turns[i]['ts']
                duration = delta.total_seconds()
            
            pairs.append({
                'question': turns[i]['text'],
                'answer':   turns[i+1]['text'],
                'model':    model,
                'duration': duration,
                'tools':    turns[i+1].get('tools', [])
            })
            last_model = model
            i += 2
        else:
            i += 1  # user turn with no response yet — skip
    else:
        last_model = turns[i].get('model') or last_model
        i += 1

# Log only NEW pairs
new_pairs = pairs[logged_pairs:]
for p in new_pairs:
    q  = p['question']
    a  = p['answer']
    m  = p['model']
    dur = p['duration']
    tools = ",".join(p['tools'])
    ti = max(1, len(q) // 4)
    to = max(1, len(a) // 4)
    summary = a.split('\n')[0][:80]
    
    # Position: auto q a m ti to dur sid status tools tags
    tags = "gemini-cli"
    cmd = [atrack, 'auto', q, summary, m, str(ti), str(to), str(dur), session_id, 'success', tools, tags]
    r = subprocess.run(cmd, capture_output=True, text=True)
    
    icon = "✅" if r.returncode == 0 else "⚠️ "
    print(f"{icon} [{m}] [{dur:.2f}s] {q[:60]}")

# Save updated count
with open(state_file, 'w') as f:
    f.write(str(len(pairs)))
PYEOF
}

# Main poll loop
while true; do
  for SESSION in "$CHATS_DIR"/session-*.jsonl; do
    [ -f "$SESSION" ] || continue
    process_session "$SESSION"
  done
  sleep "$POLL_INTERVAL"
done
