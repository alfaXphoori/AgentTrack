#!/bin/bash
# gemiatrack - Gemini CLI wrapper with AgentTrack auto-logging

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ATRACK_BIN="$(command -v atrack || echo "$SCRIPT_DIR/../atrack")"

# Auto-detect live model from latest Gemini CLI session file
detect_live_model() {
  python3 - <<'PYEOF'
import json, os, glob, sys

def find_model(obj):
    if isinstance(obj, dict):
        for k, v in obj.items():
            if k == 'model' and isinstance(v, str) and 'gemini' in v.lower():
                return v
            r = find_model(v)
            if r: return r
    elif isinstance(obj, list):
        for i in obj:
            r = find_model(i)
            if r: return r
    return None

cwd = os.getcwd()
tmp_base = os.path.expanduser('~/.gemini/tmp')

# find matching project dir
target_dir = None
for d in os.listdir(tmp_base):
    pr = os.path.join(tmp_base, d, '.project_root')
    if os.path.exists(pr):
        with open(pr) as f:
            if f.read().strip().lower() == cwd.lower():
                target_dir = os.path.join(tmp_base, d)
                break

if not target_dir:
    sys.exit(1)

sessions = sorted(glob.glob(os.path.join(target_dir, 'chats', 'session-*.jsonl')), key=os.path.getmtime)
for s in reversed(sessions):
    model = None
    with open(s) as f:
        for line in f:
            line = line.strip()
            if not line: continue
            try:
                m = find_model(json.loads(line))
                if m: model = m
            except: pass
    if model:
        print(model)
        sys.exit(0)
sys.exit(1)
PYEOF
}

LIVE_MODEL=$(detect_live_model 2>/dev/null)
MODEL="${GEMINI_MODEL:-${LIVE_MODEL:-gemini-2.5-flash}}"

echo ""
printf "\033[1;32m╔══════════════════════════════════════════╗\033[0m\n"
printf "\033[1;32m║  Gemini CLI  +  AgentTrack Auto-Logger   ║\033[0m\n"
printf "\033[1;32m║  Model: %-33s║\033[0m\n" "$MODEL"
printf "\033[1;32m║  Type 'exit' or '/model' to switch       ║\033[0m\n"
printf "\033[1;32m╚══════════════════════════════════════════╝\033[0m\n"
echo ""

while true; do
  printf "\033[1;34mYou:\033[0m "
  read -r QUESTION
  
  [ -z "$QUESTION" ] && continue
  [[ "$QUESTION" == "exit" || "$QUESTION" == "quit" ]] && echo "Bye!" && break
  
  if [ "$QUESTION" = "/model" ]; then
    printf "Switch model (current: \033[1m$MODEL\033[0m): "
    read -r NEW_MODEL
    [ -n "$NEW_MODEL" ] && MODEL="$NEW_MODEL"
    printf "✅ Model set to: \033[1m$MODEL\033[0m\n\n"
    continue
  fi
  
  echo ""
  printf "\033[1;33mGemini ($MODEL):\033[0m\n"
  printf "─────────────────────────────────────────\n"
  
  ANSWER=$(gemini -m "$MODEL" -p "$QUESTION" 2>/dev/null)
  echo "$ANSWER"
  echo ""
  
  TOKENS_IN=$(( $(echo -n "$QUESTION" | wc -c | tr -d ' ') / 4 + 1 ))
  TOKENS_OUT=$(( $(echo -n "$ANSWER" | wc -c | tr -d ' ') / 4 + 1 ))
  ANSWER_SUMMARY=$(echo "$ANSWER" | head -1 | cut -c1-80)
  
  "$ATRACK_BIN" auto "$QUESTION" "$ANSWER_SUMMARY" "$MODEL" $TOKENS_IN $TOKENS_OUT 2>/dev/null
  
  printf "📋 \033[2mLogged → model: $MODEL | ~$TOKENS_IN in | ~$TOKENS_OUT out\033[0m\n\n"
done
